package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestResourceCommandsUseSourceCompatibleRoutes(t *testing.T) {
	t.Setenv("DSCLI_CONFIG_DIR", t.TempDir())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("token") != "test-token" {
			t.Error("missing token")
		}
		if err := r.ParseForm(); err != nil {
			t.Error(err)
		}
		switch r.URL.Path {
		case "/dolphinscheduler/resources/base-dir":
			if r.Method != http.MethodGet || r.FormValue("type") != "FILE" {
				t.Error("invalid base-dir request")
			}
		case "/dolphinscheduler/resources/directory":
			if r.Method != http.MethodPost || r.FormValue("name") != "ods_success_monitor" || r.FormValue("currentDir") != "/resources" {
				t.Error("invalid directory request")
			}
		default:
			t.Errorf("unexpected route %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": "/resources"})
	}))
	defer server.Close()
	for _, args := range [][]string{{"resource", "base-dir"}, {"resource", "mkdir", "ods_success_monitor", "--current-dir", "/resources"}} {
		args = append(args, "--api-url", server.URL+"/dolphinscheduler", "--token", "test-token")
		if _, err := executeRoot(t, args...); err != nil {
			t.Fatal(err)
		}
	}
}

func TestResourceUploadPreservesExactFileAndFormFields(t *testing.T) {
	t.Setenv("DSCLI_CONFIG_DIR", t.TempDir())
	content := "database\ttable\nπαράδειγμα\tt\n"
	file := filepath.Join(t.TempDir(), "paths.tsv")
	if err := os.WriteFile(file, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/dolphinscheduler/resources" {
			t.Errorf("invalid route: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("token") != "test-token" {
			t.Error("missing token")
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		if r.FormValue("name") != "remote.tsv" || r.FormValue("currentDir") != "/resources/ods" || r.FormValue("type") != "FILE" {
			t.Error("invalid multipart fields")
		}
		upload, header, err := r.FormFile("file")
		if err != nil {
			t.Fatal(err)
		}
		defer upload.Close()
		body, err := io.ReadAll(upload)
		if err != nil {
			t.Fatal(err)
		}
		if header.Filename != "paths.tsv" || string(body) != content {
			t.Errorf("file changed: filename %q content %q", header.Filename, body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0})
	}))
	defer server.Close()
	_, err := executeRoot(t, "resource", "upload", file, "--name", "remote.tsv", "--current-dir", "/resources/ods", "--api-url", server.URL+"/dolphinscheduler", "--token", "test-token")
	if err != nil {
		t.Fatal(err)
	}
}
