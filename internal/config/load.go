package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func Default() Config {
	return Config{
		Cluster: Cluster{
			Name:       "dolphinscheduler-pseudo",
			InstallDir: "/opt/ds-cli",
			DataDir:    "/data/ds-cli",
			User:       "dolphinscheduler",
			JavaHome:   "/opt/ds-cli/java",
		},
		Versions: Versions{
			DolphinScheduler: "3.4.1",
			ZooKeeper:        "3.8.4",
			Java:             "11",
			MySQLDriver:      "8.0.33",
		},
		MySQL: MySQL{
			Host:           "127.0.0.1",
			Port:           3306,
			Database:       "dolphinscheduler",
			ServerTimezone: "Asia/Shanghai",
			CreateDatabase: false,
		},
		ZooKeeper: ZooKeeper{ClientPort: 2181},
		API:       API{Port: 12345},
		Services:  Services{API: true, Master: true, Worker: true, Alert: true},
	}
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := Default()
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	if err := Validate(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func Resolve(flag string) (string, string, error) {
	if flag != "" {
		return expand(flag), "--config", nil
	}
	if env := os.Getenv("DSCLI_CONFIG"); env != "" {
		return expand(env), "$DSCLI_CONFIG", nil
	}
	if _, err := os.Stat("ds.yaml"); err == nil {
		return "ds.yaml", "./ds.yaml", nil
	}
	home, _ := os.UserHomeDir()
	if home != "" {
		p := filepath.Join(home, ".ds-cli", "ds.yaml")
		if _, err := os.Stat(p); err == nil {
			return p, "~/.ds-cli/ds.yaml", nil
		}
	}
	return "", "", errors.New("config not found: pass --config, set DSCLI_CONFIG, create ./ds.yaml, or create ~/.ds-cli/ds.yaml")
}

func Validate(cfg *Config) error {
	if strings.TrimSpace(cfg.Cluster.InstallDir) == "" {
		return fmt.Errorf("cluster.install_dir is required")
	}
	if strings.TrimSpace(cfg.Cluster.DataDir) == "" {
		return fmt.Errorf("cluster.data_dir is required")
	}
	if strings.TrimSpace(cfg.Cluster.User) == "" {
		return fmt.Errorf("cluster.user is required")
	}
	if strings.TrimSpace(cfg.Cluster.JavaHome) == "" {
		return fmt.Errorf("cluster.java_home is required")
	}
	if cfg.Versions.DolphinScheduler != "3.4.1" {
		return fmt.Errorf("unsupported dolphinscheduler version %q: ds-cli currently targets 3.4.1", cfg.Versions.DolphinScheduler)
	}
	if strings.TrimSpace(cfg.Versions.ZooKeeper) == "" {
		return fmt.Errorf("versions.zookeeper is required")
	}
	if strings.TrimSpace(cfg.MySQL.Host) == "" {
		return fmt.Errorf("mysql.host is required")
	}
	if cfg.MySQL.Port <= 0 {
		return fmt.Errorf("mysql.port must be positive")
	}
	if strings.TrimSpace(cfg.MySQL.Database) == "" {
		return fmt.Errorf("mysql.database is required")
	}
	if strings.TrimSpace(cfg.MySQL.Username) == "" {
		return fmt.Errorf("mysql.username is required")
	}
	if cfg.MySQL.CreateDatabase && strings.TrimSpace(cfg.MySQL.AdminUsername) == "" {
		return fmt.Errorf("mysql.admin_username is required when mysql.create_database is true")
	}
	return nil
}

func expand(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		if home != "" {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}
