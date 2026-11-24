package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"sync"
	"testing"
)

func TestTaskInstanceListAllCompact(t *testing.T) {
	var pages []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		page := query.Get("pageNo")
		pages = append(pages, page)
		if r.URL.Path != "/projects/42/task-instances" || query.Get("taskName") != "task" {
			t.Errorf("unexpected request %s", r.URL)
		}
		rows := map[string]string{
			"1": `[{"id":1,"name":"task","taskCode":9007199254740993,"workflowInstanceId":10,"taskParams":"large"}]`,
			"2": `[{"id":1,"name":"task"},{"id":2,"name":"task"}]`,
		}
		writeTaskTestPage(w, 2, 2, rows[page])
	}))
	defer server.Close()
	t.Setenv("DSCLI_API_URL", server.URL)
	t.Setenv("DSCLI_TOKEN", "tok")

	stdout, err := executeRoot(t, "task-instance", "list", "--project-code", "42", "--task-name", "task", "--all", "--compact")
	if err != nil {
		t.Fatalf("list: %v\n%s", err, stdout)
	}
	var result struct {
		Data struct {
			Data struct {
				Returned int                          `json:"returned"`
				Rows     []map[string]json.RawMessage `json:"totalList"`
			} `json:"data"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatal(err)
	}
	rows := result.Data.Data.Rows
	if !slices.Equal(pages, []string{"1", "2"}) || result.Data.Data.Returned != 2 || len(rows) != 2 {
		t.Fatalf("unexpected pages %v or output %s", pages, stdout)
	}
	if string(rows[0]["taskCode"]) != "9007199254740993" || rows[0]["taskParams"] != nil {
		t.Fatalf("unexpected compact row: %s", stdout)
	}
}

func TestTaskInstanceStopRunningDryRunAndWait(t *testing.T) {
	var mu sync.Mutex
	var runningScans, queueChecks int
	var posts []string
	const batch = `{"id":10,"state":"RUNNING_EXECUTION","taskExecuteType":"BATCH","workflowInstanceId":100}`
	const otherBatch = `{"id":11,"state":"RUNNING_EXECUTION","taskExecuteType":"BATCH","workflowInstanceId":100}`
	const stream = `{"id":20,"state":"RUNNING_EXECUTION","taskExecuteType":"STREAM","workflowInstanceId":200}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch r.URL.Path {
		case "/projects/42/task-instances":
			query := r.URL.Query()
			state, kind := query.Get("stateType"), query.Get("taskExecuteType")
			if kind == "" {
				t.Error("DolphinScheduler defaults an omitted taskExecuteType to BATCH")
			}
			if state != "RUNNING_EXECUTION" {
				if state == "SUBMITTED_SUCCESS" && kind == "BATCH" {
					queueChecks++
					if queueChecks == 1 {
						writeTaskTestPage(w, 1, 1, "[]")
						return
					}
				}
				writeTaskTestPage(w, 0, 1, "[]")
				return
			}
			if kind == "BATCH" {
				runningScans++
			}
			switch runningScans {
			case 1, 2:
				if kind == "BATCH" {
					writeTaskTestPage(w, 2, 1, "["+batch+","+otherBatch+"]")
					return
				}
			case 4:
				if kind == "STREAM" {
					writeTaskTestPage(w, 1, 1, "["+stream+"]")
					return
				}
			}
			writeTaskTestPage(w, 0, 1, "[]")
		case "/projects/42/workflow-instances/100":
			_, _ = w.Write([]byte(`{"code":0,"data":{"state":"RUNNING_EXECUTION"}}`))
		case "/projects/42/executors/execute":
			if err := r.ParseForm(); err != nil || r.Form.Get("workflowInstanceId") != "100" || r.Form.Get("executeType") != "STOP" {
				t.Errorf("unexpected workflow control request: %v, %v", r.Form, err)
			}
			fallthrough
		case "/projects/42/task-instances/20/stop":
			if r.Method != http.MethodPost {
				t.Errorf("stop method = %s", r.Method)
			}
			posts = append(posts, r.URL.Path)
			_, _ = w.Write([]byte(`{"code":0}`))
		default:
			t.Errorf("unexpected request: %s", r.URL)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	t.Setenv("DSCLI_API_URL", server.URL)
	t.Setenv("DSCLI_TOKEN", "tok")

	args := []string{"task-instance", "stop-running", "--project-code", "42"}
	preview, err := executeRoot(t, append(slices.Clone(args), "--dry-run")...)
	if err != nil {
		t.Fatalf("dry run: %v\n%s", err, preview)
	}
	result := decodeStopRunningResult(t, preview)
	if !result.DryRun || result.Verified || len(posts) != 0 || len(result.Targets) != 1 ||
		!slices.Equal(result.Targets[0].TaskIDs, []int{10, 11}) || result.Targets[0].Status != "planned" {
		t.Fatalf("unexpected dry run: %s", preview)
	}
	stdout, err := executeRoot(t, append(args, "--wait", "--poll-interval", "2ms", "--quiet-period", "1ms", "--wait-timeout", "1s")...)
	if err != nil {
		t.Fatalf("stop running: %v\n%s", err, stdout)
	}
	result = decodeStopRunningResult(t, stdout)
	if result.DryRun || !result.Verified || result.RunningTotal != 0 || queueChecks < 2 ||
		!reflect.DeepEqual(posts, []string{"/projects/42/executors/execute", "/projects/42/task-instances/20/stop"}) {
		t.Fatalf("posts = %v, queue checks = %d, output = %s", posts, queueChecks, stdout)
	}
}

func TestTaskInstanceStopRunningReturnsPartialResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/projects/42/task-instances":
			if r.URL.Query().Get("taskExecuteType") != "STREAM" {
				writeTaskTestPage(w, 0, 1, "[]")
				return
			}
			writeTaskTestPage(w, 2, 1, `[{"id":10,"state":"RUNNING_EXECUTION","taskExecuteType":"STREAM"},{"id":11,"state":"RUNNING_EXECUTION","taskExecuteType":"STREAM"}]`)
		case "/projects/42/task-instances/10/stop":
			_, _ = w.Write([]byte(`{"code":42,"msg":"stop rejected"}`))
		case "/projects/42/task-instances/11/stop":
			_, _ = w.Write([]byte(`{"code":0}`))
		default:
			t.Errorf("unexpected request: %s", r.URL)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	t.Setenv("DSCLI_API_URL", server.URL)
	t.Setenv("DSCLI_TOKEN", "tok")

	stdout, err := executeRoot(t, "task-instance", "stop-running", "--project-code", "42")
	if err == nil {
		t.Fatalf("expected an error: %s", stdout)
	}
	result := decodeStopRunningResult(t, stdout)
	if result.Verified || len(result.Targets) != 2 || result.Targets[0].Status != "error" ||
		result.Targets[0].Error == "" || result.Targets[1].Status != "accepted" {
		t.Fatalf("unexpected partial result: %s", stdout)
	}
}

func TestStopRunningReportsAcceptedTargetOnTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/projects/42/task-instances":
			if r.URL.Query().Get("taskExecuteType") == "STREAM" {
				writeTaskTestPage(w, 1, 1, `[{"id":10,"state":"RUNNING_EXECUTION","taskExecuteType":"STREAM"}]`)
				return
			}
			writeTaskTestPage(w, 0, 1, "[]")
		case "/projects/42/task-instances/10/stop":
			_, _ = w.Write([]byte(`{"code":0}`))
		default:
			t.Errorf("unexpected request: %s", r.URL)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	t.Setenv("DSCLI_API_URL", server.URL)
	t.Setenv("DSCLI_TOKEN", "tok")

	stdout, err := executeRoot(t, "task-instance", "stop-running", "--project-code", "42",
		"--wait", "--poll-interval", "1s", "--quiet-period", "1ms", "--wait-timeout", "250ms")
	if err == nil {
		t.Fatalf("expected timeout: %s", stdout)
	}
	result := decodeStopRunningResult(t, stdout)
	if result.Verified || result.RunningTotal != 1 || len(result.Remaining) != 1 ||
		len(result.Targets) != 1 || result.Targets[0].Status != "accepted" {
		t.Fatalf("expected accepted request and running task after timeout: %s", stdout)
	}
}

func writeTaskTestPage(w http.ResponseWriter, total, totalPage int, rows string) {
	_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{
		"total": total, "totalPage": totalPage, "totalList": json.RawMessage(rows),
	}})
}

func decodeStopRunningResult(t *testing.T, stdout string) stopRunningResult {
	t.Helper()
	var result struct {
		Data stopRunningResult `json:"data"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("decode stop-running result: %v\n%s", err, stdout)
	}
	return result.Data
}
