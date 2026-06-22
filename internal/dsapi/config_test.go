package dsapi

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestResolveProfileUsesActiveClusterAndEnvAndFlagOverrides(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := []byte(`active_cluster: dev
clusters:
  dev:
    api_url: http://dev:12345/dolphinscheduler
    username: admin
    password: dev-pass
    timeout: 5s
  prod:
    api_url: http://prod:12345/dolphinscheduler
    token: prod-token
`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("DSCLI_PASSWORD", "env-pass")
	profile, err := ResolveProfile(path, APIOverrides{
		Cluster: "prod",
		Timeout: 9 * time.Second,
	})
	if err != nil {
		t.Fatalf("ResolveProfile returned error: %v", err)
	}

	if profile.Name != "prod" {
		t.Fatalf("profile.Name = %q, want prod", profile.Name)
	}
	if profile.APIURL != "http://prod:12345/dolphinscheduler" {
		t.Fatalf("profile.APIURL = %q", profile.APIURL)
	}
	if profile.Token != "prod-token" {
		t.Fatalf("profile.Token = %q", profile.Token)
	}
	if profile.Password != "env-pass" {
		t.Fatalf("profile.Password = %q", profile.Password)
	}
	if profile.Timeout != 9*time.Second {
		t.Fatalf("profile.Timeout = %s, want 9s", profile.Timeout)
	}
}

func TestResolveProfileRequiresAPIURL(t *testing.T) {
	t.Setenv("DSCLI_CONFIG_DIR", t.TempDir())
	t.Setenv("DSCLI_API_URL", "")
	t.Setenv("DSCLI_CLUSTER", "")
	_, err := ResolveProfile("", APIOverrides{})
	if err == nil {
		t.Fatal("ResolveProfile succeeded without api_url")
	}
}
