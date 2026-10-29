package dsapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync"
	"testing"
)

func TestPatchWorkflowTaskResourceOnlyPreservesTaskAndWorkflowFields(t *testing.T) {
	var mu sync.Mutex
	var requests []string
	var updatedTask map[string]any
	var siblingTask map[string]any
	var updatedWorkflow url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests = append(requests, r.Method+" "+r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/projects/10/workflow-definition/20":
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{
				"workflowDefinition": map[string]any{
					"name": "wf", "description": "keep me", "globalParams": "[]", "locations": "[]",
					"timeout": 15, "executionType": "PARALLEL", "releaseState": "ONLINE",
				},
				"workflowTaskRelationList": []map[string]any{{"preTaskCode": 0, "postTaskCode": 7}},
				"taskDefinitionList": []map[string]any{
					{"id": 99, "code": 7, "name": "monitor", "taskType": "SHELL", "workerGroup": "default", "environmentCode": -1,
						"taskParams": map[string]any{"rawScript": "old", "localParams": []any{map[string]any{"prop": "x"}}, "resourceList": []any{}}},
					{"code": 8, "name": "sibling", "taskType": "SHELL", "workerGroup": "default", "taskParams": map[string]any{"rawScript": "sibling"}},
				},
			}})
		case r.Method == http.MethodPost && r.URL.Path == "/projects/10/workflow-definition/20/release":
			_ = r.ParseForm()
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{}})
		case r.Method == http.MethodPut && r.URL.Path == "/projects/10/workflow-definition/20":
			_ = r.ParseForm()
			updatedWorkflow = r.Form
			var tasks []map[string]any
			if err := json.Unmarshal([]byte(r.FormValue("taskDefinitionJson")), &tasks); err != nil {
				t.Fatal(err)
			}
			for _, task := range tasks {
				if int64(task["code"].(float64)) == 7 {
					updatedTask = task
				} else if int64(task["code"].(float64)) == 8 {
					siblingTask = task
				}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"version": 2}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient(Profile{APIURL: server.URL, Token: "test-token"})
	resources := []ResourceReference{{ResourceName: "/default/resources/ods_success_monitor/check_ods_success.py"}}
	result, err := PatchWorkflowTask(context.Background(), client, WorkflowPatchInput{
		ProjectCode: 10, WorkflowCode: 20, TaskCode: 7, Resources: &resources,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.PrevReleaseState != "ONLINE" || result.FinalReleaseState != "ONLINE" || result.NewWorkflowVersion != 2 {
		t.Fatalf("unexpected result: %#v", result)
	}
	params, ok := updatedTask["taskParams"].(map[string]any)
	if !ok {
		t.Fatalf("updated task params missing: %#v", updatedTask)
	}
	if params["rawScript"] != "old" {
		t.Fatalf("rawScript changed unexpectedly: %#v", params)
	}
	if _, ok := params["localParams"]; !ok {
		t.Fatalf("localParams was not preserved: %#v", params)
	}
	gotResources, ok := params["resourceList"].([]any)
	if !ok || len(gotResources) != 1 || gotResources[0].(map[string]any)["resourceName"] != resources[0].ResourceName {
		t.Fatalf("resourceList = %#v", params["resourceList"])
	}
	if updatedWorkflow.Get("description") != "keep me" || updatedWorkflow.Get("timeout") != strconv.Itoa(15) {
		t.Fatalf("workflow metadata was not preserved: %#v", updatedWorkflow)
	}
	if siblingTask["name"] != "sibling" || siblingTask["taskParams"].(map[string]any)["rawScript"] != "sibling" {
		t.Fatalf("sibling task was changed: %#v", siblingTask)
	}
	if _, ok := updatedTask["id"]; ok {
		t.Fatalf("task audit id leaked into update: %#v", updatedTask)
	}
	mu.Lock()
	defer mu.Unlock()
	want := []string{
		"GET /projects/10/workflow-definition/20",
		"POST /projects/10/workflow-definition/20/release",
		"PUT /projects/10/workflow-definition/20",
		"POST /projects/10/workflow-definition/20/release",
	}
	if len(requests) != len(want) {
		t.Fatalf("requests = %#v", requests)
	}
	for i := range want {
		if requests[i] != want[i] {
			t.Fatalf("requests = %#v, want %#v", requests, want)
		}
	}
}

func TestPatchWorkflowTaskResourceClearAndScriptEnvironmentUpdate(t *testing.T) {
	tests := []struct {
		name      string
		resources *[]ResourceReference
		wantCount int
		wantFirst string
	}{
		{name: "nil preserves resources", wantCount: 1, wantFirst: "/default/resources/old.py"},
		{name: "empty array clears resources", resources: &[]ResourceReference{}, wantCount: 0},
		{name: "replacement is exact", resources: &[]ResourceReference{{ResourceName: "/default/resources/new.py"}}, wantCount: 1, wantFirst: "/default/resources/new.py"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var updated map[string]any
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.Method == http.MethodGet {
					_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{
						"workflowDefinition": map[string]any{"name": "wf", "releaseState": "OFFLINE"},
						"taskDefinitionList": []map[string]any{{"code": 7, "environmentCode": -1,
							"taskParams": map[string]any{"rawScript": "old", "resourceList": []ResourceReference{{ResourceName: "/default/resources/old.py"}}}}},
						"workflowTaskRelationList": []map[string]any{{"preTaskCode": 0, "postTaskCode": 7}},
					}})
					return
				}
				if r.Method != http.MethodPut {
					t.Errorf("unexpected release request for offline workflow: %s", r.Method)
				}
				_ = r.ParseForm()
				var tasks []map[string]any
				if err := json.Unmarshal([]byte(r.FormValue("taskDefinitionJson")), &tasks); err != nil {
					t.Error(err)
				}
				updated = tasks[0]
				_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"version": 3}})
			}))
			defer server.Close()
			env := int64(123)
			_, err := PatchWorkflowTask(context.Background(), NewClient(Profile{APIURL: server.URL, Token: "test-token"}), WorkflowPatchInput{
				ProjectCode: 10, WorkflowCode: 20, TaskCode: 7,
				NewRawScript: "new", Resources: tt.resources, EnvironmentCode: &env,
			})
			if err != nil {
				t.Fatal(err)
			}
			if updated["environmentCode"].(float64) != 123 {
				t.Fatalf("environmentCode = %#v", updated["environmentCode"])
			}
			params := updated["taskParams"].(map[string]any)
			if params["rawScript"] != "new" {
				t.Fatalf("rawScript = %#v", params["rawScript"])
			}
			resources, ok := params["resourceList"].([]any)
			if !ok || len(resources) != tt.wantCount {
				t.Fatalf("resourceList = %#v", params["resourceList"])
			}
			if tt.wantCount > 0 && resources[0].(map[string]any)["resourceName"] != tt.wantFirst {
				t.Fatalf("resourceList = %#v", resources)
			}
		})
	}
}

func TestPatchWorkflowTaskRejectsInvalidResourcesBeforeNetwork(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
	defer server.Close()
	client := NewClient(Profile{APIURL: server.URL, Token: "test-token"})
	resources := []ResourceReference{{ResourceName: "/default/resources/a"}, {ResourceName: "/default/resources/a"}}
	_, err := PatchWorkflowTask(context.Background(), client, WorkflowPatchInput{ProjectCode: 1, WorkflowCode: 2, TaskCode: 3, Resources: &resources})
	if err == nil {
		t.Fatal("expected duplicate resource validation error")
	}
	if called {
		t.Fatal("invalid resources should be rejected before making an HTTP request")
	}
}
