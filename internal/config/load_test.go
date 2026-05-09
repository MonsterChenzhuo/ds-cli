package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAppliesDefaultsAndUserMySQL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ds.yaml")
	err := os.WriteFile(path, []byte(`
mysql:
  host: mysql.internal
  username: ds_user
  password: ds_pass
`), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Versions.DolphinScheduler != "3.4.1" {
		t.Fatalf("DolphinScheduler version = %q", cfg.Versions.DolphinScheduler)
	}
	if cfg.MySQL.Host != "mysql.internal" || cfg.MySQL.Port != 3306 {
		t.Fatalf("mysql defaults not merged: %+v", cfg.MySQL)
	}
	if !cfg.Services.API || !cfg.Services.Master || !cfg.Services.Worker || !cfg.Services.Alert {
		t.Fatalf("expected all pseudo-cluster services enabled by default: %+v", cfg.Services)
	}
}

func TestValidateRequiresMySQLUser(t *testing.T) {
	cfg := Default()
	cfg.MySQL.Username = ""

	err := Validate(&cfg)
	if err == nil || err.Error() != "mysql.username is required" {
		t.Fatalf("Validate error = %v", err)
	}
}
