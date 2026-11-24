package dsapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNormalizeBaseURLWithProjectLink(t *testing.T) {
	const base = "https://example.com/dolphinscheduler"
	for _, input := range []string{
		base,
		base + "/ui/",
		base + "/ui/#/projects/42/task-instance",
		base + "/ui/projects/42/workflow-definition?projectName=demo",
	} {
		if got := NormalizeBaseURL(input); got != base {
			t.Errorf("NormalizeBaseURL(%q) = %q, want %q", input, got, base)
		}
	}
}

func TestClientLogsInWithPasswordAndSendsSessionHeader(t *testing.T) {
	var sawSession string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dolphinscheduler/login":
			if got := r.FormValue("userName"); got != "admin" {
				t.Fatalf("userName = %q", got)
			}
			if got := r.FormValue("userPassword"); got != "secret" {
				t.Fatalf("userPassword = %q", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"msg":  "login success",
				"data": map[string]any{"sessionId": "session-123"},
			})
		case "/dolphinscheduler/projects":
			sawSession = r.Header.Get("sessionId")
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": []any{}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient(Profile{
		APIURL:   server.URL + "/dolphinscheduler",
		Username: "admin",
		Password: "secret",
	})
	if _, err := client.Form(context.Background(), http.MethodGet, "/projects", nil); err != nil {
		t.Fatalf("Form returned error: %v", err)
	}
	if sawSession != "session-123" {
		t.Fatalf("sessionId header = %q, want session-123", sawSession)
	}
}

func TestClientPrefersTokenWithoutLogin(t *testing.T) {
	var sawToken, sawUserAgent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/dolphinscheduler/login" {
			t.Fatal("login should not be called when token is configured")
		}
		sawToken = r.Header.Get("token")
		sawUserAgent = r.Header.Get("User-Agent")
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"ok": true}})
	}))
	defer server.Close()

	client := NewClient(Profile{
		APIURL: server.URL + "/dolphinscheduler",
		Token:  "token-abc",
	})
	if _, err := client.JSON(context.Background(), http.MethodPost, "/v2/projects", map[string]any{"projectName": "demo"}); err != nil {
		t.Fatalf("JSON returned error: %v", err)
	}
	if sawToken != "token-abc" {
		t.Fatalf("token header = %q, want token-abc", sawToken)
	}
	if sawUserAgent != userAgent {
		t.Fatalf("user-agent = %q, want %q", sawUserAgent, userAgent)
	}
}

func TestClientCloudflareResponses(t *testing.T) {
	for _, tc := range []struct {
		name      string
		status    int
		challenge bool
		raw       bool
		ray       string
		body      string
		want      string
	}{
		{"API challenge", http.StatusForbidden, true, false, "example-SIN", "<html>Challenge</html>", `Cloudflare challenge (HTTP 403, Ray ID "example-SIN")`},
		{"download challenge", http.StatusOK, true, true, "", "<html>Challenge</html>", "Cloudflare challenge (HTTP 200)"},
		{"proxied rejection", http.StatusForbidden, false, false, "example-SIN", `{"msg":"permission denied"}`, `Ray ID "example-SIN"): {"msg":"permission denied"}`},
		{"proxied throttling", http.StatusTooManyRequests, false, true, "example-SIN", "<html>Rate limited</html>", `HTTP 429 via Cloudflare (Ray ID "example-SIN")`},
		{"proxied download", http.StatusOK, false, true, "example-SIN", "task log", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.challenge {
					w.Header().Set("CF-Mitigated", "challenge")
				}
				if tc.ray != "" {
					w.Header().Set("CF-Ray", tc.ray)
				}
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			client := NewClient(Profile{APIURL: server.URL, Token: "tok"})
			var resp *Response
			var err error
			if tc.raw {
				resp, err = client.RawGet(context.Background(), "/log/download-log", nil)
			} else {
				resp, err = client.Form(context.Background(), http.MethodGet, "/projects", nil)
			}
			if tc.want == "" {
				if err != nil || resp == nil || string(resp.Body) != tc.body {
					t.Fatalf("download response = %v, error = %v", resp, err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) || resp == nil || resp.HTTPStatus != tc.status {
				t.Fatalf("response = %v, error = %v, want status %d and %q", resp, err, tc.status, tc.want)
			}
		})
	}
}
