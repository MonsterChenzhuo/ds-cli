package cmd

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTaskInstanceStopFindsStandaloneStream(t *testing.T) {
	var posts int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/projects/42/task-instances":
			if r.URL.Query().Get("taskExecuteType") != "STREAM" || r.URL.Query().Get("stateType") != "RUNNING_EXECUTION" {
				t.Errorf("unexpected task filters: %s", r.URL)
			}
			writeTaskTestPage(w, 1, 1, `[{"id":20,"state":"RUNNING_EXECUTION","taskExecuteType":"STREAM","workflowInstanceId":0}]`)
		case "/projects/42/task-instances/20/stop":
			if r.Method != http.MethodPost {
				t.Errorf("stop method: %s", r.Method)
			}
			posts++
			_, _ = w.Write([]byte(`{"code":0}`))
		default:
			t.Errorf("unexpected request: %s", r.URL)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	t.Setenv("DSCLI_API_URL", server.URL)
	t.Setenv("DSCLI_TOKEN", "tok")

	if stdout, err := executeRoot(t, "task-instance", "stop", "20", "--project-code", "42"); err != nil {
		t.Fatalf("stop standalone stream: %v\n%s", err, stdout)
	}
	if stdout, err := executeRoot(t, "task-instance", "stop", "21", "--project-code", "42"); err == nil || posts != 1 {
		t.Fatalf("unexpected unverified stop result, posts = %d, error = %v\n%s", posts, err, stdout)
	}
}
