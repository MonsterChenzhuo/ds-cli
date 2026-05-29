package dsapi

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const defaultTimeout = 30 * time.Second

type Profile struct {
	Name      string        `json:"name,omitempty" yaml:"-"`
	APIURL    string        `json:"api_url,omitempty" yaml:"api_url,omitempty"`
	Username  string        `json:"username,omitempty" yaml:"username,omitempty"`
	Password  string        `json:"-" yaml:"password,omitempty"`
	Token     string        `json:"-" yaml:"token,omitempty"`
	SessionID string        `json:"-" yaml:"session_id,omitempty"`
	Timeout   time.Duration `json:"timeout_ms,omitempty" yaml:"-"`
}

type APIOverrides struct {
	Cluster   string
	APIURL    string
	Username  string
	Password  string
	Token     string
	SessionID string
	Timeout   time.Duration
}

type ConfigFile struct {
	ActiveCluster string             `json:"active_cluster,omitempty" yaml:"active_cluster,omitempty"`
	Clusters      map[string]Profile `json:"clusters,omitempty" yaml:"clusters,omitempty"`
}

type rawConfigFile struct {
	ActiveCluster string                `yaml:"active_cluster"`
	Clusters      map[string]rawProfile `yaml:"clusters"`
}

type rawProfile struct {
	APIURL    string `yaml:"api_url"`
	Username  string `yaml:"username"`
	Password  string `yaml:"password"`
	Token     string `yaml:"token"`
	SessionID string `yaml:"session_id"`
	Timeout   string `yaml:"timeout"`
}

func DefaultConfigPath() (string, error) {
	if dir := os.Getenv("DSCLI_CONFIG_DIR"); dir != "" {
		return filepath.Join(dir, "config.yaml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "ds-cli", "config.yaml"), nil
}

func LoadConfigFile(path string) (ConfigFile, error) {
	if path == "" {
		var err error
		path, err = DefaultConfigPath()
		if err != nil {
			return ConfigFile{}, err
		}
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ConfigFile{}, nil
		}
		return ConfigFile{}, err
	}
	var raw rawConfigFile
	if err := yaml.Unmarshal(b, &raw); err != nil {
		return ConfigFile{}, err
	}
	out := ConfigFile{
		ActiveCluster: strings.TrimSpace(raw.ActiveCluster),
		Clusters:      make(map[string]Profile, len(raw.Clusters)),
	}
	for name, p := range raw.Clusters {
		timeout := time.Duration(0)
		if strings.TrimSpace(p.Timeout) != "" {
			d, err := time.ParseDuration(strings.TrimSpace(p.Timeout))
			if err != nil {
				return ConfigFile{}, err
			}
			timeout = d
		}
		out.Clusters[name] = Profile{
			Name:      name,
			APIURL:    strings.TrimSpace(p.APIURL),
			Username:  strings.TrimSpace(p.Username),
			Password:  p.Password,
			Token:     strings.TrimSpace(p.Token),
			SessionID: strings.TrimSpace(p.SessionID),
			Timeout:   timeout,
		}
	}
	return out, nil
}

func SaveConfigFile(path string, file ConfigFile) error {
	if path == "" {
		var err error
		path, err = DefaultConfigPath()
		if err != nil {
			return err
		}
	}
	raw := rawConfigFile{
		ActiveCluster: file.ActiveCluster,
		Clusters:      map[string]rawProfile{},
	}
	for name, p := range file.Clusters {
		rp := rawProfile{
			APIURL:    p.APIURL,
			Username:  p.Username,
			Password:  p.Password,
			Token:     p.Token,
			SessionID: p.SessionID,
		}
		if p.Timeout > 0 {
			rp.Timeout = p.Timeout.String()
		}
		raw.Clusters[name] = rp
	}
	b, err := yaml.Marshal(raw)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}

func ResolveProfile(path string, overrides APIOverrides) (Profile, error) {
	file, err := LoadConfigFile(path)
	if err != nil {
		return Profile{}, err
	}
	cluster := strings.TrimSpace(overrides.Cluster)
	if cluster == "" {
		cluster = strings.TrimSpace(os.Getenv("DSCLI_CLUSTER"))
	}
	if cluster == "" {
		cluster = file.ActiveCluster
	}

	var profile Profile
	if cluster != "" {
		p, ok := file.Clusters[cluster]
		if !ok {
			return Profile{}, errors.New("ds cluster " + cluster + " not found in config.clusters")
		}
		profile = p
		profile.Name = cluster
	}

	applyEnv(&profile)
	applyOverrides(&profile, overrides)
	if profile.Timeout == 0 {
		profile.Timeout = defaultTimeout
	}
	if strings.TrimSpace(profile.APIURL) == "" {
		return Profile{}, errors.New("api_url is required: configure ds-cli config cluster add <name> --api-url, set DSCLI_API_URL, or pass --api-url")
	}
	profile.APIURL = NormalizeBaseURL(profile.APIURL)
	return profile, nil
}

func applyEnv(profile *Profile) {
	if v := os.Getenv("DSCLI_API_URL"); v != "" {
		profile.APIURL = v
	}
	if v := os.Getenv("DSCLI_USER"); v != "" {
		profile.Username = v
	}
	if v := os.Getenv("DSCLI_PASSWORD"); v != "" {
		profile.Password = v
	}
	if v := os.Getenv("DSCLI_TOKEN"); v != "" {
		profile.Token = v
	}
	if v := os.Getenv("DSCLI_SESSION_ID"); v != "" {
		profile.SessionID = v
	}
	if v := os.Getenv("DSCLI_API_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			profile.Timeout = d
		}
	}
}

func applyOverrides(profile *Profile, overrides APIOverrides) {
	if overrides.APIURL != "" {
		profile.APIURL = overrides.APIURL
	}
	if overrides.Username != "" {
		profile.Username = overrides.Username
	}
	if overrides.Password != "" {
		profile.Password = overrides.Password
	}
	if overrides.Token != "" {
		profile.Token = overrides.Token
	}
	if overrides.SessionID != "" {
		profile.SessionID = overrides.SessionID
	}
	if overrides.Timeout > 0 {
		profile.Timeout = overrides.Timeout
	}
}
