package dsapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

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
	var sawToken string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/dolphinscheduler/login" {
			t.Fatal("login should not be called when token is configured")
		}
		sawToken = r.Header.Get("token")
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
}
