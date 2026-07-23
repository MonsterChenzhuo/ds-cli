package dsapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// DAGTask is one node of a multi-task workflow. Code is assigned by the caller
// (via gen-task-codes) before the form is built; Deps references upstream task
// names, not codes.
type DAGTask struct {
	Name              string
	Description       string
	TaskType          string
	Script            string
	WorkerGroup       string
	EnvironmentCode   int64
	FailRetryTimes    int
	FailRetryInterval int
	Deps              []string
	Code              int64
}

// MultiTaskDAG describes a whole workflow: metadata, global params, and a set of
// tasks connected by name-based dependencies.
type MultiTaskDAG struct {
	Name          string
	Description   string
	ExecutionType string
	Timeout       int
	// GlobalParams is the workflow-level global params JSON array (verbatim). Empty means "[]".
	GlobalParams string
	Tasks        []DAGTask
}

// ValidateDAG checks everything about a DAG that does not depend on task codes:
// name/tasks presence, unique non-empty task names, task type, non-empty scripts,
// dependency existence, no self-dependency, no cycles, and valid globalParams JSON.
// Callers should run this BEFORE allocating task codes / any network call, so a
// malformed DAG fails locally without leaving partial state or wasting codes.
// It normalizes each task's TaskType to upper case in place.
func ValidateDAG(in MultiTaskDAG) error {
	if strings.TrimSpace(in.Name) == "" {
		return errors.New("workflow name is required")
	}
	if len(in.Tasks) == 0 {
		return errors.New("at least one task is required")
	}

	byName := make(map[string]*DAGTask, len(in.Tasks))
	for i := range in.Tasks {
		t := &in.Tasks[i]
		name := strings.TrimSpace(t.Name)
		if name == "" {
			return fmt.Errorf("task[%d]: name is required", i)
		}
		if _, dup := byName[name]; dup {
			return fmt.Errorf("duplicate task name %q", name)
		}
		taskType := strings.ToUpper(strings.TrimSpace(t.TaskType))
		if taskType == "" {
			taskType = "SHELL"
		}
		if taskType != "SHELL" && taskType != "PYTHON" {
			return fmt.Errorf("task %q: task type must be SHELL or PYTHON", name)
		}
		t.TaskType = taskType
		if t.Script == "" {
			return fmt.Errorf("task %q: script is required", name)
		}
		byName[name] = t
	}

	for i := range in.Tasks {
		t := &in.Tasks[i]
		for _, dep := range t.Deps {
			dep = strings.TrimSpace(dep)
			if dep == "" {
				continue
			}
			if dep == t.Name {
				return fmt.Errorf("task %q: self-dependency is not allowed", t.Name)
			}
			if _, ok := byName[dep]; !ok {
				return fmt.Errorf("task %q: dependency %q does not match any task name", t.Name, dep)
			}
		}
	}

	if strings.TrimSpace(in.GlobalParams) != "" && !json.Valid([]byte(in.GlobalParams)) {
		return errors.New("global params is not valid JSON")
	}

	// Cycle detection (levels discarded here; recomputed in MultiTaskWorkflowForm).
	if _, err := topoLevels(in.Tasks, byName); err != nil {
		return err
	}
	return nil
}

// MultiTaskWorkflowForm validates the DAG and produces the form values for
// POST /projects/{code}/workflow-definition. It never sets releaseState: callers
// create the workflow OFFLINE and online it in a separate step, which is the path
// verified against DS 3.4.1. Every task must already have a non-zero Code.
func MultiTaskWorkflowForm(in MultiTaskDAG) (url.Values, error) {
	if err := ValidateDAG(in); err != nil {
		return nil, err
	}

	byName := make(map[string]*DAGTask, len(in.Tasks))
	for i := range in.Tasks {
		t := &in.Tasks[i]
		if t.Code == 0 {
			return nil, fmt.Errorf("task %q: code is required (assign via gen-task-codes before building the form)", t.Name)
		}
		byName[t.Name] = t
	}

	// Cycle-free already guaranteed by ValidateDAG; recompute levels for layout.
	levels, err := topoLevels(in.Tasks, byName)
	if err != nil {
		return nil, err
	}

	// Build task definitions.
	tasks := make([]map[string]any, 0, len(in.Tasks))
	for i := range in.Tasks {
		t := &in.Tasks[i]
		workerGroup := strings.TrimSpace(t.WorkerGroup)
		if workerGroup == "" {
			workerGroup = "default"
		}
		envCode := t.EnvironmentCode
		if envCode == 0 {
			envCode = -1
		}
		retryInterval := t.FailRetryInterval
		if retryInterval == 0 {
			retryInterval = 1
		}
		tasks = append(tasks, map[string]any{
			"code":        t.Code,
			"name":        t.Name,
			"description": t.Description,
			"taskType":    t.TaskType,
			"taskParams": map[string]any{
				"localParams":  []any{},
				"rawScript":    t.Script,
				"resourceList": []any{},
			},
			"flag":                  "YES",
			"taskPriority":          "MEDIUM",
			"workerGroup":           workerGroup,
			"environmentCode":       envCode,
			"failRetryTimes":        t.FailRetryTimes,
			"failRetryInterval":     retryInterval,
			"timeoutFlag":           "CLOSE",
			"timeoutNotifyStrategy": "",
			"timeout":               0,
			"delayTime":             0,
			"taskExecuteType":       "BATCH",
			"cpuQuota":              -1,
			"memoryMax":             -1,
		})
	}

	relations := buildRelations(in.Tasks, byName)
	locations := buildLocations(in.Tasks, levels)

	taskJSON, err := json.Marshal(tasks)
	if err != nil {
		return nil, err
	}
	relationJSON, err := json.Marshal(relations)
	if err != nil {
		return nil, err
	}
	locationJSON, err := json.Marshal(locations)
	if err != nil {
		return nil, err
	}

	globalParams := strings.TrimSpace(in.GlobalParams)
	if globalParams == "" {
		globalParams = "[]"
	}
	executionType := strings.TrimSpace(in.ExecutionType)
	if executionType == "" {
		executionType = "PARALLEL"
	}

	values := url.Values{}
	values.Set("name", in.Name)
	values.Set("description", in.Description)
	values.Set("globalParams", globalParams)
	values.Set("locations", string(locationJSON))
	values.Set("timeout", strconv.Itoa(in.Timeout))
	values.Set("executionType", executionType)
	values.Set("taskDefinitionJson", string(taskJSON))
	values.Set("taskRelationJson", string(relationJSON))
	return values, nil
}

// buildRelations produces one relation per (dep -> task) edge, plus a root
// relation (preTaskCode=0) for every task with no dependencies. This mirrors how
// the DS UI submits relations and what was verified against DS 3.4.1.
func buildRelations(tasks []DAGTask, byName map[string]*DAGTask) []map[string]any {
	relations := make([]map[string]any, 0, len(tasks))
	for i := range tasks {
		t := &tasks[i]
		deps := dedupeDeps(t.Deps)
		if len(deps) == 0 {
			relations = append(relations, relation(0, t.Code))
			continue
		}
		for _, dep := range deps {
			relations = append(relations, relation(byName[dep].Code, t.Code))
		}
	}
	return relations
}

func relation(pre, post int64) map[string]any {
	return map[string]any{
		"preTaskCode":     pre,
		"preTaskVersion":  0,
		"postTaskCode":    post,
		"postTaskVersion": 0,
		"conditionType":   "NONE",
		"conditionParams": map[string]any{},
		"name":            "",
	}
}

// buildLocations lays tasks out left-to-right by dependency level, stacking
// tasks that share a level vertically.
func buildLocations(tasks []DAGTask, levels map[string]int) []map[string]any {
	slot := make(map[int]int)
	locations := make([]map[string]any, 0, len(tasks))
	for i := range tasks {
		t := &tasks[i]
		lvl := levels[t.Name]
		y := 200 + slot[lvl]*160
		slot[lvl]++
		locations = append(locations, map[string]any{
			"taskCode": t.Code,
			"x":        200 + lvl*260,
			"y":        y,
		})
	}
	return locations
}

// topoLevels assigns each task its longest-path depth from a root (level 0) and
// detects cycles via Kahn's algorithm. A cycle is reported when not every task
// can be processed.
func topoLevels(tasks []DAGTask, byName map[string]*DAGTask) (map[string]int, error) {
	// indegree = number of (deduped, valid) dependencies.
	indegree := make(map[string]int, len(tasks))
	// dependents[dep] = tasks that depend on dep.
	dependents := make(map[string][]string, len(tasks))
	for i := range tasks {
		t := &tasks[i]
		deps := dedupeDeps(t.Deps)
		indegree[t.Name] = len(deps)
		for _, dep := range deps {
			dependents[dep] = append(dependents[dep], t.Name)
		}
	}

	level := make(map[string]int, len(tasks))
	queue := make([]string, 0, len(tasks))
	for i := range tasks {
		if indegree[tasks[i].Name] == 0 {
			queue = append(queue, tasks[i].Name)
			level[tasks[i].Name] = 0
		}
	}

	processed := 0
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		processed++
		for _, child := range dependents[name] {
			if level[child] < level[name]+1 {
				level[child] = level[name] + 1
			}
			indegree[child]--
			if indegree[child] == 0 {
				queue = append(queue, child)
			}
		}
	}

	if processed != len(tasks) {
		return nil, errors.New("dependency cycle detected among tasks")
	}
	return level, nil
}

// dedupeDeps trims, drops empties, and removes duplicate dependency names while
// preserving first-seen order.
func dedupeDeps(deps []string) []string {
	seen := make(map[string]struct{}, len(deps))
	out := make([]string, 0, len(deps))
	for _, d := range deps {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		if _, ok := seen[d]; ok {
			continue
		}
		seen[d] = struct{}{}
		out = append(out, d)
	}
	return out
}
