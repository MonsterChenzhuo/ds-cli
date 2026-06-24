package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveGlobalParamsInlineValidJSON(t *testing.T) {
	got, err := resolveGlobalParams(`[{"prop":"a","value":"1"}]`, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != `[{"prop":"a","value":"1"}]` {
		t.Fatalf("got %q", got)
	}
}

func TestResolveGlobalParamsEmpty(t *testing.T) {
	got, err := resolveGlobalParams("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Fatalf("got %q, want empty", got)
	}
}

func TestResolveGlobalParamsInlineInvalidJSON(t *testing.T) {
	if _, err := resolveGlobalParams("not-json", ""); err == nil {
		t.Fatal("expected error for invalid inline JSON, got nil")
	}
}

func TestResolveGlobalParamsFromFilePreservesTimePlaceholder(t *testing.T) {
	// The $[yyyy-MM-dd-1] placeholder must survive verbatim; this is the whole
	// reason --global-params-file exists (the shell mangles it inline).
	dir := t.TempDir()
	file := filepath.Join(dir, "gp.json")
	content := `[{"prop":"biz_date","direct":"IN","type":"VARCHAR","value":"$[yyyy-MM-dd-1]"}]`
	if err := os.WriteFile(file, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	got, err := resolveGlobalParams("", file)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != content {
		t.Fatalf("got %q, want %q", got, content)
	}
}

func TestResolveGlobalParamsInlineAndFileMutuallyExclusive(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "gp.json")
	if err := os.WriteFile(file, []byte("[]"), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	if _, err := resolveGlobalParams("[]", file); err == nil {
		t.Fatal("expected mutual-exclusion error, got nil")
	}
}
