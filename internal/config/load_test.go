package config

import (
	"os"
	"path/filepath"
	"strings"
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

func TestLoadDistributedWithManagedZooKeeper(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ds.yaml")
	err := os.WriteFile(path, []byte(`
cluster:
  mode: distributed
mysql:
  host: mysql.internal
  username: ds_user
ssh:
  user: deploy
  private_key: ~/.ssh/id_rsa
hosts:
  - { name: ds1, address: 10.0.0.1 }
  - { name: ds2, address: 10.0.0.2 }
  - { name: ds3, address: 10.0.0.3 }
roles:
  zookeeper: [ds1, ds2, ds3]
  api_server: [ds1]
  master_server: [ds1, ds2]
  worker_server: [ds2, ds3]
  alert_server: [ds1]
`), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Distributed() {
		t.Fatal("expected distributed config")
	}
	if !cfg.UsesManagedZooKeeper() {
		t.Fatal("expected managed zookeeper")
	}
	if got := cfg.AllRoleHosts(); len(got) != 3 {
		t.Fatalf("AllRoleHosts len = %d, hosts = %#v", len(got), got)
	}
}

func TestLoadDistributedWithExternalZooKeeper(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ds.yaml")
	err := os.WriteFile(path, []byte(`
cluster:
  mode: distributed
mysql:
  host: mysql.internal
  username: ds_user
zookeeper:
  external_connect_string: zk1:2181,zk2:2181,zk3:2181
ssh:
  user: deploy
  private_key: ~/.ssh/id_rsa
hosts:
  - { name: ds1, address: 10.0.0.1 }
roles:
  api_server: [ds1]
  master_server: [ds1]
  worker_server: [ds1]
  alert_server: [ds1]
`), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.UsesManagedZooKeeper() {
		t.Fatal("expected external zookeeper")
	}
}

func TestValidateDistributedRequiresOddManagedZooKeeper(t *testing.T) {
	cfg := Default()
	cfg.Cluster.Mode = "distributed"
	cfg.MySQL.Username = "ds_user"
	cfg.SSH.User = "deploy"
	cfg.SSH.PrivateKey = "~/.ssh/id_rsa"
	cfg.Hosts = []Host{{Name: "ds1", Address: "10.0.0.1"}, {Name: "ds2", Address: "10.0.0.2"}}
	cfg.Roles = Roles{
		ZooKeeper: []string{"ds1", "ds2"},
		API:       []string{"ds1"},
		Master:    []string{"ds1"},
		Worker:    []string{"ds2"},
		Alert:     []string{"ds1"},
	}

	err := Validate(&cfg)
	if err == nil || !strings.Contains(err.Error(), "roles.zookeeper must have an odd number") {
		t.Fatalf("Validate error = %v", err)
	}
}
