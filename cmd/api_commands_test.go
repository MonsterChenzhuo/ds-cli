package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ds-cli/ds-cli/internal/dsapi"
)

func executeRoot(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), err
}

func TestConfigClusterAddWritesProfile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DSCLI_CONFIG_DIR", dir)

	out, err := executeRoot(t, "config", "cluster", "add", "dev",
		"--api-url", "http://localhost:12345/dolphinscheduler",
		"--user", "admin",
		"--password", "secret",
		"--activate",
	)
	if err != nil {
		t.Fatalf("config cluster add returned error: %v", err)
	}

	var envelope struct {
		Command string `json:"command"`
		OK      bool   `json:"ok"`
		Data    struct {
			ActiveCluster string `json:"active_cluster"`
			Path          string `json:"path"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(out), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, out)
	}
	if !envelope.OK || envelope.Command != "config.cluster.add" {
		t.Fatalf("unexpected envelope: %+v", envelope)
	}
	if envelope.Data.ActiveCluster != "dev" {
		t.Fatalf("active cluster = %q", envelope.Data.ActiveCluster)
	}
	if envelope.Data.Path != filepath.Join(dir, "config.yaml") {
		t.Fatalf("path = %q", envelope.Data.Path)
	}

	file, err := dsapi.LoadConfigFile(envelope.Data.Path)
	if err != nil {
		t.Fatal(err)
	}
	if file.ActiveCluster != "dev" {
		t.Fatalf("file active_cluster = %q", file.ActiveCluster)
	}
	if file.Clusters["dev"].Password != "secret" {
		t.Fatalf("password was not written")
	}
}

func TestAPICommandConfigErrorWritesEnvelope(t *testing.T) {
	t.Setenv("DSCLI_CONFIG_DIR", t.TempDir())
	t.Setenv("DSCLI_API_URL", "")
	t.Setenv("DSCLI_TOKEN", "")
	t.Setenv("DSCLI_SESSION_ID", "")
	t.Setenv("DSCLI_USER", "")
	t.Setenv("DSCLI_PASSWORD", "")

	out, err := executeRoot(t, "project", "list")
	if err == nil {
		t.Fatal("project list returned nil error without API profile")
	}
	var envelope struct {
		Command string `json:"command"`
		OK      bool   `json:"ok"`
		Error   struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(out), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, out)
	}
	if envelope.Command != "project.list" || envelope.OK {
		t.Fatalf("unexpected envelope: %+v", envelope)
	}
	if envelope.Error.Code != "CONFIG_ERROR" || !strings.Contains(envelope.Error.Message, "api_url is required") {
		t.Fatalf("unexpected error envelope: %+v", envelope.Error)
	}
}

func TestProjectCreatePostsToDolphinSchedulerAPI(t *testing.T) {
	var sawToken string
	var sawBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/dolphinscheduler/v2/projects" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("method = %q", r.Method)
		}
		sawToken = r.Header.Get("token")
		if err := json.NewDecoder(r.Body).Decode(&sawBody); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"msg":  "success",
			"data": map[string]any{"projectName": "demo"},
		})
	}))
	defer server.Close()
	t.Setenv("DSCLI_API_URL", server.URL+"/dolphinscheduler")
	t.Setenv("DSCLI_TOKEN", "token-xyz")

	out, err := executeRoot(t, "project", "create", "demo", "--description", "created by test")
	if err != nil {
		t.Fatalf("project create returned error: %v", err)
	}
	if sawToken != "token-xyz" {
		t.Fatalf("token header = %q", sawToken)
	}
	if sawBody["projectName"] != "demo" || sawBody["description"] != "created by test" {
		t.Fatalf("unexpected body: %#v", sawBody)
	}
	if !strings.Contains(out, `"command": "project.create"`) || !strings.Contains(out, `"ok": true`) {
		t.Fatalf("unexpected stdout:\n%s", out)
	}
}

func TestTaskCreateGeneratesCodeAndCreatesSingleTaskWorkflow(t *testing.T) {
	var sawCreate bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dolphinscheduler/projects/123/task-definition/gen-task-codes":
			if r.URL.Query().Get("genNum") != "1" {
				t.Fatalf("genNum = %q", r.URL.Query().Get("genNum"))
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"data": map[string]any{"dataList": []int64{987}},
			})
		case "/dolphinscheduler/projects/123/workflow-definition":
			sawCreate = true
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
			if r.Form.Get("name") != "daily_shell" {
				t.Fatalf("workflow name = %q", r.Form.Get("name"))
			}
			if !strings.Contains(r.Form.Get("taskDefinitionJson"), `"rawScript":"echo hello"`) {
				t.Fatalf("taskDefinitionJson = %s", r.Form.Get("taskDefinitionJson"))
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"data": map[string]any{"code": 456},
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("DSCLI_API_URL", server.URL+"/dolphinscheduler")
	t.Setenv("DSCLI_TOKEN", "token-xyz")

	scriptPath := filepath.Join(t.TempDir(), "task.sh")
	if err := os.WriteFile(scriptPath, []byte("echo hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := executeRoot(t, "task", "create", "extract",
		"--project-code", "123",
		"--workflow-name", "daily_shell",
		"--script-file", scriptPath,
	)
	if err != nil {
		t.Fatalf("task create returned error: %v", err)
	}
	if !sawCreate {
		t.Fatal("workflow create endpoint was not called")
	}
	if !strings.Contains(out, `"command": "task.create"`) || !strings.Contains(out, `"ok": true`) {
		t.Fatalf("unexpected stdout:\n%s", out)
	}
}
