package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
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

func TestConfigInitWritesTemplate(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DSCLI_CONFIG_DIR", dir)

	out, err := executeRoot(t, "config", "init")
	if err != nil {
		t.Fatalf("config init returned error: %v", err)
	}
	var envelope struct {
		Command string `json:"command"`
		OK      bool   `json:"ok"`
		Data    struct {
			Path    string `json:"path"`
			Written bool   `json:"written"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(out), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, out)
	}
	if !envelope.OK || envelope.Command != "config.init" || !envelope.Data.Written {
		t.Fatalf("unexpected envelope: %+v", envelope)
	}
	body, err := os.ReadFile(filepath.Join(dir, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, want := range []string{
		"active_cluster: prod",
		"clusters:",
		"api_url: https://dolphinscheduler.example.com/dolphinscheduler",
		"token: <access-token>",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("config template missing %q:\n%s", want, text)
		}
	}
}

func TestConfigShowMasksCredentialsAndReportsSources(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DSCLI_CONFIG_DIR", dir)
	t.Setenv("DSCLI_TOKEN", "env-token")
	t.Setenv("DSCLI_API_URL", "https://env.example.com/dolphinscheduler")
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(`
active_cluster: prod
clusters:
  prod:
    api_url: https://prod.example.com/dolphinscheduler
    token: stored-token
    timeout: 45s
`), 0o600); err != nil {
		t.Fatal(err)
	}

	out, err := executeRoot(t, "config", "show")
	if err != nil {
		t.Fatalf("config show returned error: %v", err)
	}
	if strings.Contains(out, "stored-token") || strings.Contains(out, "env-token") {
		t.Fatalf("config show exposed a token:\n%s", out)
	}
	var envelope struct {
		Command string `json:"command"`
		OK      bool   `json:"ok"`
		Data    struct {
			SelectedCluster string `json:"selected_cluster"`
			APIURL          struct {
				Source string `json:"source"`
				Value  string `json:"value"`
			} `json:"api_url"`
			Auth struct {
				Source     string `json:"source"`
				Method     string `json:"method"`
				HasToken   bool   `json:"has_token"`
				HasSession bool   `json:"has_session"`
				HasUser    bool   `json:"has_user"`
				HasPass    bool   `json:"has_password"`
			} `json:"auth"`
			Timeout struct {
				Source string `json:"source"`
				Value  string `json:"value"`
			} `json:"timeout"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(out), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, out)
	}
	if envelope.Command != "config.show" || !envelope.OK {
		t.Fatalf("unexpected envelope: %+v", envelope)
	}
	if envelope.Data.SelectedCluster != "prod" {
		t.Fatalf("selected_cluster = %q", envelope.Data.SelectedCluster)
	}
	if envelope.Data.APIURL.Value != "https://env.example.com/dolphinscheduler" || envelope.Data.APIURL.Source != "env" {
		t.Fatalf("api_url field = %+v", envelope.Data.APIURL)
	}
	if envelope.Data.Auth.Method != "token" || envelope.Data.Auth.Source != "env" || !envelope.Data.Auth.HasToken {
		t.Fatalf("auth field = %+v", envelope.Data.Auth)
	}
	if envelope.Data.Timeout.Value != "45s" || envelope.Data.Timeout.Source != "file" {
		t.Fatalf("timeout field = %+v", envelope.Data.Timeout)
	}
}

func TestConfigShowWorksWithoutProfile(t *testing.T) {
	t.Setenv("DSCLI_CONFIG_DIR", t.TempDir())
	t.Setenv("DSCLI_API_URL", "")
	t.Setenv("DSCLI_TOKEN", "")
	t.Setenv("DSCLI_SESSION_ID", "")
	t.Setenv("DSCLI_USER", "")
	t.Setenv("DSCLI_PASSWORD", "")

	out, err := executeRoot(t, "config", "show")
	if err != nil {
		t.Fatalf("config show returned error without profile: %v", err)
	}
	var envelope struct {
		Command string `json:"command"`
		OK      bool   `json:"ok"`
		Data    struct {
			SelectedCluster string `json:"selected_cluster"`
			APIURL          struct {
				Source string `json:"source"`
				Value  string `json:"value"`
			} `json:"api_url"`
			Auth struct {
				Source string `json:"source"`
				Method string `json:"method"`
			} `json:"auth"`
			Timeout struct {
				Source string `json:"source"`
				Value  string `json:"value"`
			} `json:"timeout"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(out), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, out)
	}
	if envelope.Command != "config.show" || !envelope.OK {
		t.Fatalf("unexpected envelope: %+v", envelope)
	}
	if envelope.Data.SelectedCluster != "" || envelope.Data.APIURL.Value != "" {
		t.Fatalf("unexpected selected/api fields: %+v", envelope.Data)
	}
	if envelope.Data.Auth.Method != "none" || envelope.Data.Auth.Source != "default" {
		t.Fatalf("auth field = %+v", envelope.Data.Auth)
	}
	if envelope.Data.Timeout.Value != "30s" || envelope.Data.Timeout.Source != "default" {
		t.Fatalf("timeout field = %+v", envelope.Data.Timeout)
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

func TestRootDoesNotExposeDeploymentLifecycleCommands(t *testing.T) {
	for _, name := range []string{
		"preflight",
		"install",
		"configure",
		"init-db",
		"plugins",
		"start",
		"stop",
		"restart",
		"status",
		"systemd",
		"uninstall",
		"bootstrap",
	} {
		name := name
		t.Run(name, func(t *testing.T) {
			out, err := executeRoot(t, name)
			if err == nil {
				t.Fatalf("%s returned nil error; deployment lifecycle commands must be removed", name)
			}
			if out != "" {
				t.Fatalf("%s wrote stdout %q; unknown commands should not emit API envelopes", name, out)
			}
		})
	}

	out, err := executeRoot(t, "--help")
	if err != nil {
		t.Fatalf("--help returned error: %v", err)
	}
	for _, removed := range []string{"bootstrap", "install", "zookeeper", "systemd"} {
		if strings.Contains(out, removed) {
			t.Fatalf("root help still mentions removed deployment term %q:\n%s", removed, out)
		}
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

func TestWorkflowGetDetailHitsLegacyDefinitionEndpoint(t *testing.T) {
	var sawPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawPath = r.URL.Path
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"data": map[string]any{
				"workflowDefinition":       map[string]any{"code": 999, "name": "wf"},
				"workflowTaskRelationList": []any{},
				"taskDefinitionList":       []any{map[string]any{"code": 111, "name": "t"}},
			},
		})
	}))
	defer server.Close()
	t.Setenv("DSCLI_API_URL", server.URL+"/dolphinscheduler")
	t.Setenv("DSCLI_TOKEN", "tok")
	out, err := executeRoot(t, "workflow", "get-detail", "999", "--project-code", "42")
	if err != nil {
		t.Fatalf("workflow get-detail: %v", err)
	}
	if sawPath != "/dolphinscheduler/projects/42/workflow-definition/999" {
		t.Fatalf("path = %q", sawPath)
	}
	if !strings.Contains(out, `"command": "workflow.get-detail"`) || !strings.Contains(out, `"ok": true`) {
		t.Fatalf("unexpected stdout:\n%s", out)
	}
}

func TestWorkflowPatchTaskRunsOfflineUpdateOnlineChain(t *testing.T) {
	var calls []string
	var sawPutForm url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/dolphinscheduler/projects/42/workflow-definition/999":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"data": map[string]any{
					"workflowDefinition": map[string]any{
						"name": "wf", "description": "d",
						"releaseState":  "ONLINE",
						"globalParams":  "[]",
						"locations":     "[]",
						"executionType": "PARALLEL",
						"timeout":       float64(0),
					},
					"workflowTaskRelationList": []any{map[string]any{
						"preTaskCode":  float64(0),
						"postTaskCode": float64(111),
					}},
					"taskDefinitionList": []any{map[string]any{
						"code":     float64(111),
						"name":     "check_partition",
						"taskType": "SHELL",
						"taskParams": map[string]any{
							"rawScript": "old",
						},
					}},
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/dolphinscheduler/projects/42/workflow-definition/999/release":
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": true})
		case r.Method == http.MethodPut && r.URL.Path == "/dolphinscheduler/projects/42/workflow-definition/999":
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
			sawPutForm = r.Form
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"code": 999, "version": 7}})
		default:
			t.Fatalf("unexpected call: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("DSCLI_API_URL", server.URL+"/dolphinscheduler")
	t.Setenv("DSCLI_TOKEN", "tok")

	scriptPath := filepath.Join(t.TempDir(), "new.sh")
	if err := os.WriteFile(scriptPath, []byte("echo new\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := executeRoot(t, "workflow", "patch-task", "999",
		"--project-code", "42",
		"--task-code", "111",
		"--raw-script-file", scriptPath,
	)
	if err != nil {
		t.Fatalf("workflow patch-task: %v", err)
	}

	expected := []string{
		"GET /dolphinscheduler/projects/42/workflow-definition/999",
		"POST /dolphinscheduler/projects/42/workflow-definition/999/release",
		"PUT /dolphinscheduler/projects/42/workflow-definition/999",
		"POST /dolphinscheduler/projects/42/workflow-definition/999/release",
	}
	if len(calls) != 4 {
		t.Fatalf("expected 4 calls, got %d: %v", len(calls), calls)
	}
	for i, e := range expected {
		if calls[i] != e {
			t.Fatalf("call[%d] = %q, want %q", i, calls[i], e)
		}
	}
	if !strings.Contains(sawPutForm.Get("taskDefinitionJson"), `"rawScript":"echo new\n"`) {
		t.Fatalf("taskDefinitionJson missing new rawScript: %s", sawPutForm.Get("taskDefinitionJson"))
	}
	if !strings.Contains(out, `"command": "workflow.patch-task"`) || !strings.Contains(out, `"final_release_state": "ONLINE"`) {
		t.Fatalf("unexpected stdout:\n%s", out)
	}
}

func TestWorkflowStartPostsForm(t *testing.T) {
	var sawForm url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/dolphinscheduler/projects/42/executors/start-workflow-instance" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		sawForm = r.Form
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": []int{12345}})
	}))
	defer server.Close()
	t.Setenv("DSCLI_API_URL", server.URL+"/dolphinscheduler")
	t.Setenv("DSCLI_TOKEN", "tok")

	out, err := executeRoot(t, "workflow", "start", "999",
		"--project-code", "42",
		"--schedule-time", "2026-01-01 00:00:00",
		"--environment-code", "77",
	)
	if err != nil {
		t.Fatalf("workflow start: %v", err)
	}
	if sawForm.Get("workflowDefinitionCode") != "999" || sawForm.Get("scheduleTime") != "2026-01-01 00:00:00" {
		t.Fatalf("form = %v", sawForm)
	}
	if sawForm.Get("failureStrategy") != "CONTINUE" || sawForm.Get("warningType") != "NONE" || sawForm.Get("environmentCode") != "77" {
		t.Fatalf("form defaults = %v", sawForm)
	}
	if !strings.Contains(out, `"command": "workflow.start"`) || !strings.Contains(out, `"ok": true`) {
		t.Fatalf("unexpected stdout:\n%s", out)
	}
}

func TestWorkflowInstanceControlMapsResumeToRecoverSuspended(t *testing.T) {
	var sawForm url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/dolphinscheduler/projects/42/executors/execute" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		sawForm = r.Form
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": true})
	}))
	defer server.Close()
	t.Setenv("DSCLI_API_URL", server.URL+"/dolphinscheduler")
	t.Setenv("DSCLI_TOKEN", "tok")
	out, err := executeRoot(t, "workflow-instance", "control", "4350",
		"--project-code", "42",
		"--type", "RESUME",
	)
	if err != nil {
		t.Fatalf("workflow-instance control: %v", err)
	}
	if sawForm.Get("workflowInstanceId") != "4350" || sawForm.Get("executeType") != "RECOVER_SUSPENDED_PROCESS" {
		t.Fatalf("form = %v", sawForm)
	}
	if !strings.Contains(out, `"command": "workflow-instance.control"`) {
		t.Fatalf("unexpected stdout:\n%s", out)
	}
}

func TestTaskInstanceLogHitsDetailEndpoint(t *testing.T) {
	var sawQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/dolphinscheduler/log/detail" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		sawQuery = r.URL.RawQuery
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"message": "log line", "lineNum": 1}})
	}))
	defer server.Close()
	t.Setenv("DSCLI_API_URL", server.URL+"/dolphinscheduler")
	t.Setenv("DSCLI_TOKEN", "tok")
	out, err := executeRoot(t, "task-instance", "log", "11840", "--skip-line-num", "0", "--limit", "200")
	if err != nil {
		t.Fatalf("task-instance log: %v", err)
	}
	if !strings.Contains(sawQuery, "taskInstanceId=11840") || !strings.Contains(sawQuery, "limit=200") {
		t.Fatalf("query = %q", sawQuery)
	}
	if !strings.Contains(out, `"command": "task-instance.log"`) {
		t.Fatalf("unexpected stdout:\n%s", out)
	}
}

func TestTaskInstanceLogDownloadWritesFile(t *testing.T) {
	payload := []byte("RAW LOG BYTES\nLINE 2\n")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/dolphinscheduler/log/download-log" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(payload)
	}))
	defer server.Close()
	t.Setenv("DSCLI_API_URL", server.URL+"/dolphinscheduler")
	t.Setenv("DSCLI_TOKEN", "tok")
	outPath := filepath.Join(t.TempDir(), "out.log")
	out, err := executeRoot(t, "task-instance", "log-download", "42", "--output", outPath)
	if err != nil {
		t.Fatalf("task-instance log-download: %v", err)
	}
	disk, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(disk, payload) {
		t.Fatalf("file mismatch: %q vs %q", string(disk), string(payload))
	}
	if !strings.Contains(out, `"command": "task-instance.log-download"`) || !strings.Contains(out, `"bytes": 21`) {
		t.Fatalf("unexpected stdout:\n%s", out)
	}
}

func TestTaskDefUpdatePostsWithUpstream(t *testing.T) {
	var calls []string
	var sawForm url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/dolphinscheduler/projects/42/task-definition/111":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"data": map[string]any{
					"code": 111, "name": "t", "taskType": "SHELL",
					"taskParams": map[string]any{"rawScript": "old"},
				},
			})
		case r.Method == http.MethodPut && r.URL.Path == "/dolphinscheduler/projects/42/task-definition/111/with-upstream":
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
			sawForm = r.Form
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": 111})
		default:
			t.Fatalf("unexpected call: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("DSCLI_API_URL", server.URL+"/dolphinscheduler")
	t.Setenv("DSCLI_TOKEN", "tok")
	out, err := executeRoot(t, "task-def", "update", "111",
		"--project-code", "42",
		"--raw-script", "echo new",
		"--upstream-codes", "100,101",
	)
	if err != nil {
		t.Fatalf("task-def update: %v", err)
	}
	if len(calls) != 2 || calls[0] != "GET /dolphinscheduler/projects/42/task-definition/111" || calls[1] != "PUT /dolphinscheduler/projects/42/task-definition/111/with-upstream" {
		t.Fatalf("calls = %v", calls)
	}
	if !strings.Contains(sawForm.Get("taskDefinitionJsonObj"), `"rawScript":"echo new"`) {
		t.Fatalf("payload missing new rawScript: %s", sawForm.Get("taskDefinitionJsonObj"))
	}
	if sawForm.Get("upstreamCodes") != "100,101" {
		t.Fatalf("upstreamCodes = %q", sawForm.Get("upstreamCodes"))
	}
	if !strings.Contains(out, `"command": "task-def.update"`) {
		t.Fatalf("unexpected stdout:\n%s", out)
	}
}

func TestConfigClusterShowMasksTokenByDefault(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DSCLI_CONFIG_DIR", dir)
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(`
active_cluster: prod
clusters:
  prod:
    api_url: https://prod.example.com/dolphinscheduler
    token: stored-token
    timeout: 45s
`), 0o600); err != nil {
		t.Fatal(err)
	}
	out, err := executeRoot(t, "config", "cluster", "show", "prod")
	if err != nil {
		t.Fatalf("config cluster show: %v", err)
	}
	if strings.Contains(out, "stored-token") {
		t.Fatalf("default cluster show exposed token:\n%s", out)
	}
	if !strings.Contains(out, `"has_token": true`) {
		t.Fatalf("missing has_token bool:\n%s", out)
	}
	out2, err := executeRoot(t, "config", "cluster", "show", "prod", "--reveal-token")
	if err != nil {
		t.Fatalf("config cluster show --reveal-token: %v", err)
	}
	if !strings.Contains(out2, "stored-token") {
		t.Fatalf("--reveal-token did not expose token:\n%s", out2)
	}
	out3, err := executeRoot(t, "config", "cluster", "show", "prod", "--shell")
	if err != nil {
		t.Fatalf("config cluster show --shell: %v", err)
	}
	if !strings.Contains(out3, `export DSCLI_API_URL="https://prod.example.com/dolphinscheduler"`) ||
		!strings.Contains(out3, `export DSCLI_TOKEN="stored-token"`) {
		t.Fatalf("--shell output unexpected:\n%s", out3)
	}
}

func TestScheduleCreateRequiresEnvironmentCode(t *testing.T) {
	t.Setenv("DSCLI_API_URL", "https://example.com/dolphinscheduler")
	t.Setenv("DSCLI_TOKEN", "tok")
	_, err := executeRoot(t, "schedule", "create",
		"--workflow-code", "999",
		"--crontab", "0 0 3 * * ? *",
		"--start-time", "2026-01-01 00:00:00",
		"--end-time", "2099-01-01 00:00:00",
	)
	if err == nil {
		t.Fatal("schedule create without --environment-code should fail")
	}
	if !strings.Contains(err.Error(), "--environment-code is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}
