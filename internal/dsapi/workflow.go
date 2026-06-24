package dsapi

import (
	"encoding/json"
	"errors"
	"net/url"
	"strconv"
	"strings"
)

type SingleTaskWorkflow struct {
	ProjectCode     int64
	WorkflowName    string
	Description     string
	TaskName        string
	TaskCode        int64
	TaskType        string
	Script          string
	WorkerGroup     string
	EnvironmentCode int64
	ReleaseState    string
	// GlobalParams is the workflow-level global params JSON array. Empty means
	// "[]" (no global params).
	GlobalParams string
}

func SingleTaskWorkflowForm(in SingleTaskWorkflow) (url.Values, error) {
	if in.ProjectCode == 0 {
		return nil, errors.New("project code is required")
	}
	if strings.TrimSpace(in.WorkflowName) == "" {
		return nil, errors.New("workflow name is required")
	}
	if strings.TrimSpace(in.TaskName) == "" {
		return nil, errors.New("task name is required")
	}
	if in.TaskCode == 0 {
		return nil, errors.New("task code is required")
	}
	taskType := strings.ToUpper(strings.TrimSpace(in.TaskType))
	if taskType == "" {
		taskType = "SHELL"
	}
	if taskType != "SHELL" && taskType != "PYTHON" {
		return nil, errors.New("task type must be SHELL or PYTHON")
	}
	workerGroup := strings.TrimSpace(in.WorkerGroup)
	if workerGroup == "" {
		workerGroup = "default"
	}
	envCode := in.EnvironmentCode
	if envCode == 0 {
		envCode = -1
	}
	releaseState := strings.ToUpper(strings.TrimSpace(in.ReleaseState))
	if releaseState == "" {
		releaseState = "OFFLINE"
	}
	globalParams := strings.TrimSpace(in.GlobalParams)
	if globalParams == "" {
		globalParams = "[]"
	} else if !json.Valid([]byte(globalParams)) {
		return nil, errors.New("global params is not valid JSON")
	}

	taskParams := map[string]any{
		"resourceList":     []any{},
		"localParams":      []any{},
		"rawScript":        in.Script,
		"dependence":       map[string]any{},
		"conditionResult":  map[string]any{"successNode": []any{}, "failedNode": []any{}},
		"waitStartTimeout": map[string]any{},
		"switchResult":     map[string]any{},
	}
	tasks := []map[string]any{{
		"code":                  in.TaskCode,
		"name":                  in.TaskName,
		"version":               1,
		"description":           "",
		"delayTime":             0,
		"taskType":              taskType,
		"taskParams":            taskParams,
		"flag":                  "YES",
		"taskPriority":          "MEDIUM",
		"workerGroup":           workerGroup,
		"failRetryTimes":        0,
		"failRetryInterval":     1,
		"timeoutFlag":           "CLOSE",
		"timeoutNotifyStrategy": "WARN",
		"timeout":               0,
		"environmentCode":       envCode,
	}}
	relations := []map[string]any{{
		"name":            "",
		"preTaskCode":     0,
		"preTaskVersion":  0,
		"postTaskCode":    in.TaskCode,
		"postTaskVersion": 1,
		"conditionType":   0,
		"conditionParams": "{}",
	}}
	locations := []map[string]any{{
		"taskCode": in.TaskCode,
		"x":        320,
		"y":        180,
	}}
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
	values := url.Values{}
	values.Set("name", in.WorkflowName)
	values.Set("description", in.Description)
	values.Set("globalParams", globalParams)
	values.Set("locations", string(locationJSON))
	values.Set("timeout", "0")
	values.Set("taskRelationJson", string(relationJSON))
	values.Set("taskDefinitionJson", string(taskJSON))
	values.Set("executionType", "PARALLEL")
	values.Set("releaseState", releaseState)
	values.Set("projectCode", strconv.FormatInt(in.ProjectCode, 10))
	return values, nil
}
