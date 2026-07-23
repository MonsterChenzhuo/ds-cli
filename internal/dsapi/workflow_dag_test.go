package dsapi

import (
	"encoding/json"
	"testing"
)

// decodeRelations unmarshals the taskRelationJson form value into a slice.
func decodeRelations(t *testing.T, form interface{ Get(string) string }) []map[string]any {
	t.Helper()
	var relations []map[string]any
	if err := json.Unmarshal([]byte(form.Get("taskRelationJson")), &relations); err != nil {
		t.Fatalf("taskRelationJson invalid JSON: %v", err)
	}
	return relations
}

func TestMultiTaskWorkflowFormSerialChain(t *testing.T) {
	form, err := MultiTaskWorkflowForm(MultiTaskDAG{
		Name: "serial",
		Tasks: []DAGTask{
			{Name: "a", Code: 1, Script: "echo a"},
			{Name: "b", Code: 2, Script: "echo b", Deps: []string{"a"}},
			{Name: "c", Code: 3, Script: "echo c", Deps: []string{"b"}},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if form.Get("name") != "serial" {
		t.Fatalf("name = %q", form.Get("name"))
	}
	if form.Get("executionType") != "PARALLEL" {
		t.Fatalf("executionType = %q", form.Get("executionType"))
	}
	if form.Get("globalParams") != "[]" {
		t.Fatalf("globalParams = %q, want []", form.Get("globalParams"))
	}

	var tasks []map[string]any
	if err := json.Unmarshal([]byte(form.Get("taskDefinitionJson")), &tasks); err != nil {
		t.Fatalf("taskDefinitionJson invalid JSON: %v", err)
	}
	if len(tasks) != 3 {
		t.Fatalf("task count = %d, want 3", len(tasks))
	}
	// Defaults applied.
	if tasks[0]["workerGroup"] != "default" {
		t.Fatalf("workerGroup = %v", tasks[0]["workerGroup"])
	}
	if tasks[0]["environmentCode"].(float64) != -1 {
		t.Fatalf("environmentCode = %v", tasks[0]["environmentCode"])
	}
	if tasks[0]["failRetryInterval"].(float64) != 1 {
		t.Fatalf("failRetryInterval = %v, want default 1", tasks[0]["failRetryInterval"])
	}
	if tasks[0]["taskType"] != "SHELL" {
		t.Fatalf("taskType = %v, want default SHELL", tasks[0]["taskType"])
	}

	relations := decodeRelations(t, form)
	if len(relations) != 3 {
		t.Fatalf("relation count = %d, want 3", len(relations))
	}
	// Exactly one root (preTaskCode 0) pointing at task a.
	roots := 0
	for _, r := range relations {
		if r["preTaskCode"].(float64) == 0 {
			roots++
			if r["postTaskCode"].(float64) != 1 {
				t.Fatalf("root points at %v, want 1", r["postTaskCode"])
			}
		}
	}
	if roots != 1 {
		t.Fatalf("root count = %d, want 1", roots)
	}

	// Layout: levels 0,1,2 -> x 200,460,720.
	var locations []map[string]any
	if err := json.Unmarshal([]byte(form.Get("locations")), &locations); err != nil {
		t.Fatalf("locations invalid JSON: %v", err)
	}
	wantX := map[float64]float64{1: 200, 2: 460, 3: 720}
	for _, l := range locations {
		if got := l["x"].(float64); got != wantX[l["taskCode"].(float64)] {
			t.Fatalf("task %v x = %v, want %v", l["taskCode"], got, wantX[l["taskCode"].(float64)])
		}
	}
}

func TestMultiTaskWorkflowFormDiamond(t *testing.T) {
	// a -> b, a -> c, b -> d, c -> d
	form, err := MultiTaskWorkflowForm(MultiTaskDAG{
		Name: "diamond",
		Tasks: []DAGTask{
			{Name: "a", Code: 1, Script: "x"},
			{Name: "b", Code: 2, Script: "x", Deps: []string{"a"}},
			{Name: "c", Code: 3, Script: "x", Deps: []string{"a"}},
			{Name: "d", Code: 4, Script: "x", Deps: []string{"b", "c"}},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	relations := decodeRelations(t, form)
	// 1 root (a) + a->b + a->c + b->d + c->d = 5 edges.
	if len(relations) != 5 {
		t.Fatalf("relation count = %d, want 5", len(relations))
	}

	var locations []map[string]any
	if err := json.Unmarshal([]byte(form.Get("locations")), &locations); err != nil {
		t.Fatalf("locations invalid JSON: %v", err)
	}
	// d has longest path a->b->d (or a->c->d) = level 2 -> x 720.
	for _, l := range locations {
		if l["taskCode"].(float64) == 4 && l["x"].(float64) != 720 {
			t.Fatalf("d x = %v, want 720 (longest-path level 2)", l["x"])
		}
	}
}

func TestMultiTaskWorkflowFormCycle(t *testing.T) {
	_, err := MultiTaskWorkflowForm(MultiTaskDAG{
		Name: "cycle",
		Tasks: []DAGTask{
			{Name: "a", Code: 1, Script: "x", Deps: []string{"b"}},
			{Name: "b", Code: 2, Script: "x", Deps: []string{"a"}},
		},
	})
	if err == nil {
		t.Fatal("expected cycle error, got nil")
	}
}

func TestMultiTaskWorkflowFormSelfDependency(t *testing.T) {
	_, err := MultiTaskWorkflowForm(MultiTaskDAG{
		Name:  "self",
		Tasks: []DAGTask{{Name: "a", Code: 1, Script: "x", Deps: []string{"a"}}},
	})
	if err == nil {
		t.Fatal("expected self-dependency error, got nil")
	}
}

func TestMultiTaskWorkflowFormUnknownDep(t *testing.T) {
	_, err := MultiTaskWorkflowForm(MultiTaskDAG{
		Name:  "unknown",
		Tasks: []DAGTask{{Name: "a", Code: 1, Script: "x", Deps: []string{"ghost"}}},
	})
	if err == nil {
		t.Fatal("expected unknown-dependency error, got nil")
	}
}

func TestMultiTaskWorkflowFormDuplicateName(t *testing.T) {
	_, err := MultiTaskWorkflowForm(MultiTaskDAG{
		Name: "dup",
		Tasks: []DAGTask{
			{Name: "a", Code: 1, Script: "x"},
			{Name: "a", Code: 2, Script: "y"},
		},
	})
	if err == nil {
		t.Fatal("expected duplicate-name error, got nil")
	}
}

func TestMultiTaskWorkflowFormEmptyTasks(t *testing.T) {
	_, err := MultiTaskWorkflowForm(MultiTaskDAG{Name: "empty"})
	if err == nil {
		t.Fatal("expected empty-tasks error, got nil")
	}
}

func TestMultiTaskWorkflowFormMissingScript(t *testing.T) {
	_, err := MultiTaskWorkflowForm(MultiTaskDAG{
		Name:  "noscript",
		Tasks: []DAGTask{{Name: "a", Code: 1}},
	})
	if err == nil {
		t.Fatal("expected missing-script error, got nil")
	}
}

func TestMultiTaskWorkflowFormMissingCode(t *testing.T) {
	_, err := MultiTaskWorkflowForm(MultiTaskDAG{
		Name:  "nocode",
		Tasks: []DAGTask{{Name: "a", Script: "x"}},
	})
	if err == nil {
		t.Fatal("expected missing-code error, got nil")
	}
}

func TestMultiTaskWorkflowFormBadTaskType(t *testing.T) {
	_, err := MultiTaskWorkflowForm(MultiTaskDAG{
		Name:  "badtype",
		Tasks: []DAGTask{{Name: "a", Code: 1, Script: "x", TaskType: "SPARK"}},
	})
	if err == nil {
		t.Fatal("expected bad-task-type error, got nil")
	}
}

func TestMultiTaskWorkflowFormInvalidGlobalParams(t *testing.T) {
	_, err := MultiTaskWorkflowForm(MultiTaskDAG{
		Name:         "badgp",
		GlobalParams: "not-json",
		Tasks:        []DAGTask{{Name: "a", Code: 1, Script: "x"}},
	})
	if err == nil {
		t.Fatal("expected invalid-global-params error, got nil")
	}
}

func TestMultiTaskWorkflowFormCarriesGlobalParamsAndOverrides(t *testing.T) {
	gp := `[{"prop":"biz_date","direct":"IN","type":"VARCHAR","value":"$[yyyy-MM-dd-1]"}]`
	form, err := MultiTaskWorkflowForm(MultiTaskDAG{
		Name:          "gp",
		ExecutionType: "SERIAL",
		GlobalParams:  gp,
		Tasks: []DAGTask{
			{Name: "a", Code: 1, Script: "x", TaskType: "python", WorkerGroup: "wg1", EnvironmentCode: 555, FailRetryTimes: 2880, FailRetryInterval: 1},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if form.Get("globalParams") != gp {
		t.Fatalf("globalParams = %q", form.Get("globalParams"))
	}
	if form.Get("executionType") != "SERIAL" {
		t.Fatalf("executionType = %q, want SERIAL", form.Get("executionType"))
	}
	var tasks []map[string]any
	if err := json.Unmarshal([]byte(form.Get("taskDefinitionJson")), &tasks); err != nil {
		t.Fatalf("taskDefinitionJson invalid JSON: %v", err)
	}
	if tasks[0]["taskType"] != "PYTHON" {
		t.Fatalf("taskType = %v, want PYTHON (normalized)", tasks[0]["taskType"])
	}
	if tasks[0]["workerGroup"] != "wg1" {
		t.Fatalf("workerGroup = %v", tasks[0]["workerGroup"])
	}
	if tasks[0]["environmentCode"].(float64) != 555 {
		t.Fatalf("environmentCode = %v", tasks[0]["environmentCode"])
	}
	if tasks[0]["failRetryTimes"].(float64) != 2880 {
		t.Fatalf("failRetryTimes = %v", tasks[0]["failRetryTimes"])
	}
}
