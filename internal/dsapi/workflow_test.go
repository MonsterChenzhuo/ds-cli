package dsapi

import (
	"encoding/json"
	"testing"
)

func TestSingleTaskWorkflowFormBuildsShellDAG(t *testing.T) {
	form, err := SingleTaskWorkflowForm(SingleTaskWorkflow{
		ProjectCode:     123,
		WorkflowName:    "daily_shell",
		TaskName:        "extract",
		TaskCode:        987,
		TaskType:        "SHELL",
		Script:          "echo hello",
		WorkerGroup:     "",
		EnvironmentCode: 0,
	})
	if err != nil {
		t.Fatalf("SingleTaskWorkflowForm returned error: %v", err)
	}

	values := form
	if values.Get("name") != "daily_shell" {
		t.Fatalf("workflow name = %q", values.Get("name"))
	}
	if values.Get("executionType") != "PARALLEL" {
		t.Fatalf("executionType = %q", values.Get("executionType"))
	}

	var tasks []map[string]any
	if err := json.Unmarshal([]byte(values.Get("taskDefinitionJson")), &tasks); err != nil {
		t.Fatalf("taskDefinitionJson is invalid JSON: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("task count = %d, want 1", len(tasks))
	}
	if tasks[0]["code"].(float64) != 987 {
		t.Fatalf("task code = %v", tasks[0]["code"])
	}
	if tasks[0]["taskType"] != "SHELL" {
		t.Fatalf("task type = %v", tasks[0]["taskType"])
	}
	params := tasks[0]["taskParams"].(map[string]any)
	if params["rawScript"] != "echo hello" {
		t.Fatalf("rawScript = %v", params["rawScript"])
	}
	if tasks[0]["workerGroup"] != "default" {
		t.Fatalf("workerGroup = %v", tasks[0]["workerGroup"])
	}
	if tasks[0]["environmentCode"].(float64) != -1 {
		t.Fatalf("environmentCode = %v", tasks[0]["environmentCode"])
	}

	var relations []map[string]any
	if err := json.Unmarshal([]byte(values.Get("taskRelationJson")), &relations); err != nil {
		t.Fatalf("taskRelationJson is invalid JSON: %v", err)
	}
	if len(relations) != 1 {
		t.Fatalf("relation count = %d, want 1", len(relations))
	}
	if relations[0]["preTaskCode"].(float64) != 0 || relations[0]["postTaskCode"].(float64) != 987 {
		t.Fatalf("relation = %#v", relations[0])
	}

	if values.Get("globalParams") != "[]" {
		t.Fatalf("default globalParams = %q, want []", values.Get("globalParams"))
	}
}

func TestSingleTaskWorkflowFormCarriesGlobalParams(t *testing.T) {
	gp := `[{"prop":"biz_date","direct":"IN","type":"VARCHAR","value":"$[yyyy-MM-dd-1]"}]`
	form, err := SingleTaskWorkflowForm(SingleTaskWorkflow{
		ProjectCode:  123,
		WorkflowName: "daily_shell",
		TaskName:     "extract",
		TaskCode:     987,
		Script:       "echo hello",
		GlobalParams: gp,
	})
	if err != nil {
		t.Fatalf("SingleTaskWorkflowForm returned error: %v", err)
	}
	if form.Get("globalParams") != gp {
		t.Fatalf("globalParams = %q, want %q", form.Get("globalParams"), gp)
	}
}

func TestSingleTaskWorkflowFormRejectsInvalidGlobalParams(t *testing.T) {
	_, err := SingleTaskWorkflowForm(SingleTaskWorkflow{
		ProjectCode:  123,
		WorkflowName: "daily_shell",
		TaskName:     "extract",
		TaskCode:     987,
		Script:       "echo hello",
		GlobalParams: "not-json",
	})
	if err == nil {
		t.Fatal("expected error for invalid global params JSON, got nil")
	}
}
