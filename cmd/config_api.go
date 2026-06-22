package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ds-cli/ds-cli/internal/dsapi"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage ds-cli API cluster profiles.",
	}
	cmd.AddCommand(newConfigInitCmd())
	cmd.AddCommand(newConfigShowCmd())
	cmd.AddCommand(newConfigClusterCmd())
	return cmd
}

func newConfigInitCmd() *cobra.Command {
	var force bool
	c := &cobra.Command{
		Use:   "init",
		Short: "Create a ds-cli config template.",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := dsapi.DefaultConfigPath()
			if err != nil {
				return err
			}
			if !force {
				if _, err := os.Stat(path); err == nil {
					return fmt.Errorf("config file already exists: %s; pass --force to overwrite", path)
				} else if !os.IsNotExist(err) {
					return err
				}
			}
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return err
			}
			body := strings.Join([]string{
				"# ds-cli API profiles. Keep this file private; it may contain tokens.",
				"active_cluster: prod",
				"clusters:",
				"  prod:",
				"    api_url: https://dolphinscheduler.example.com/dolphinscheduler",
				"    token: <access-token>",
				"    timeout: 30s",
				"",
				"  # staging:",
				"  #   api_url: https://staging-ds.example.com/dolphinscheduler",
				"  #   token: <access-token>",
				"  #   timeout: 30s",
				"",
			}, "\n")
			if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
				return err
			}
			return writeDataEnvelope(cmd, "config.init", map[string]any{
				"path":    path,
				"written": true,
			})
		},
	}
	c.Flags().BoolVar(&force, "force", false, "Overwrite an existing config file")
	return c
}

type sourceField struct {
	Source string `json:"source"`
	Value  any    `json:"value"`
}

type authField struct {
	Source     string `json:"source"`
	Method     string `json:"method"`
	HasToken   bool   `json:"has_token"`
	HasSession bool   `json:"has_session"`
	HasUser    bool   `json:"has_user"`
	HasPass    bool   `json:"has_password"`
}

func newConfigShowCmd() *cobra.Command {
	var flags apiFlags
	c := &cobra.Command{
		Use:   "show",
		Short: "Print the effective API profile without exposing secrets.",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := dsapi.DefaultConfigPath()
			if err != nil {
				return err
			}
			file, err := dsapi.LoadConfigFile(path)
			if err != nil {
				return err
			}
			profile, err := effectiveProfileForShow(file, flags)
			if err != nil {
				return err
			}
			return writeDataEnvelope(cmd, "config.show", effectiveConfigData(file, profile, flags, path))
		},
	}
	addAPIFlags(c, &flags)
	return c
}

func effectiveProfileForShow(file dsapi.ConfigFile, flags apiFlags) (dsapi.Profile, error) {
	cluster := strings.TrimSpace(flags.Cluster)
	if cluster == "" {
		cluster = strings.TrimSpace(os.Getenv("DSCLI_CLUSTER"))
	}
	if cluster == "" {
		cluster = strings.TrimSpace(file.ActiveCluster)
	}

	var profile dsapi.Profile
	if cluster != "" {
		p, ok := file.Clusters[cluster]
		if !ok {
			return dsapi.Profile{}, fmt.Errorf("ds cluster %s not found in config.clusters", cluster)
		}
		profile = p
		profile.Name = cluster
	}

	applyShowEnv(&profile)
	applyShowFlags(&profile, flags)
	if profile.Timeout == 0 {
		profile.Timeout = 30 * time.Second
	}
	if strings.TrimSpace(profile.APIURL) != "" {
		profile.APIURL = dsapi.NormalizeBaseURL(profile.APIURL)
	}
	return profile, nil
}

func applyShowEnv(profile *dsapi.Profile) {
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

func applyShowFlags(profile *dsapi.Profile, flags apiFlags) {
	if flags.APIURL != "" {
		profile.APIURL = flags.APIURL
	}
	if flags.Username != "" {
		profile.Username = flags.Username
	}
	if flags.Password != "" {
		profile.Password = flags.Password
	}
	if flags.Token != "" {
		profile.Token = flags.Token
	}
	if flags.SessionID != "" {
		profile.SessionID = flags.SessionID
	}
	if flags.Timeout > 0 {
		profile.Timeout = flags.Timeout
	}
}

func effectiveConfigData(file dsapi.ConfigFile, profile dsapi.Profile, flags apiFlags, path string) map[string]any {
	clusterSource := sourceForCluster(flags.Cluster, file.ActiveCluster)
	apiSource := sourceForValue(flags.APIURL, "DSCLI_API_URL", func() bool {
		return profile.Name != "" && file.Clusters[profile.Name].APIURL != ""
	})
	timeoutSource := sourceForTimeout(flags.Timeout, func() bool {
		return profile.Name != "" && file.Clusters[profile.Name].Timeout > 0
	})
	return map[string]any{
		"path":             path,
		"active_cluster":   sourceField{Source: sourceForActive(file.ActiveCluster), Value: file.ActiveCluster},
		"selected_cluster": profile.Name,
		"cluster":          sourceField{Source: clusterSource, Value: profile.Name},
		"api_url":          sourceField{Source: apiSource, Value: profile.APIURL},
		"auth":             authSummary(file, profile, flags),
		"timeout":          sourceField{Source: timeoutSource, Value: profile.Timeout.String()},
	}
}

func sourceForActive(active string) string {
	if active == "" {
		return "default"
	}
	return "file"
}

func sourceForCluster(flagValue, active string) string {
	if strings.TrimSpace(flagValue) != "" {
		return "flag"
	}
	if os.Getenv("DSCLI_CLUSTER") != "" {
		return "env"
	}
	if strings.TrimSpace(active) != "" {
		return "file"
	}
	return "default"
}

func sourceForValue(flagValue, envName string, hasFileValue func() bool) string {
	if strings.TrimSpace(flagValue) != "" {
		return "flag"
	}
	if os.Getenv(envName) != "" {
		return "env"
	}
	if hasFileValue() {
		return "file"
	}
	return "default"
}

func sourceForTimeout(flagValue time.Duration, hasFileValue func() bool) string {
	if flagValue > 0 {
		return "flag"
	}
	if os.Getenv("DSCLI_API_TIMEOUT") != "" {
		return "env"
	}
	if hasFileValue() {
		return "file"
	}
	return "default"
}

func authSummary(file dsapi.ConfigFile, profile dsapi.Profile, flags apiFlags) authField {
	method := "none"
	source := "default"
	switch {
	case profile.Token != "":
		method = "token"
		source = authSource(flags.Token, "DSCLI_TOKEN", profile.Name, file, func(p dsapi.Profile) bool { return p.Token != "" })
	case profile.SessionID != "":
		method = "session_id"
		source = authSource(flags.SessionID, "DSCLI_SESSION_ID", profile.Name, file, func(p dsapi.Profile) bool { return p.SessionID != "" })
	case profile.Username != "" && profile.Password != "":
		method = "password"
		source = authSource(flags.Password, "DSCLI_PASSWORD", profile.Name, file, func(p dsapi.Profile) bool { return p.Password != "" && p.Username != "" })
	}
	return authField{
		Source:     source,
		Method:     method,
		HasToken:   profile.Token != "",
		HasSession: profile.SessionID != "",
		HasUser:    profile.Username != "",
		HasPass:    profile.Password != "",
	}
}

func authSource(flagValue, envName, cluster string, file dsapi.ConfigFile, hasFileValue func(dsapi.Profile) bool) string {
	if strings.TrimSpace(flagValue) != "" {
		return "flag"
	}
	if os.Getenv(envName) != "" {
		return "env"
	}
	if cluster != "" {
		if p, ok := file.Clusters[cluster]; ok && hasFileValue(p) {
			return "file"
		}
	}
	return "default"
}

func newConfigClusterCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "Manage named DolphinScheduler API clusters.",
	}
	cmd.AddCommand(newConfigClusterAddCmd())
	cmd.AddCommand(newConfigClusterListCmd())
	cmd.AddCommand(newConfigClusterActivateCmd())
	cmd.AddCommand(newConfigClusterShowCmd())
	return cmd
}

func newConfigClusterShowCmd() *cobra.Command {
	var revealToken, shell bool
	c := &cobra.Command{
		Use:   "show <name>",
		Short: "Show a single named cluster profile. With --reveal-token returns secrets; with --shell prints export lines.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := strings.TrimSpace(args[0])
			if name == "" {
				return fmt.Errorf("cluster name cannot be empty")
			}
			path, err := dsapi.DefaultConfigPath()
			if err != nil {
				return err
			}
			file, err := dsapi.LoadConfigFile(path)
			if err != nil {
				return err
			}
			p, ok := file.Clusters[name]
			if !ok {
				return fmt.Errorf("cluster %q not found", name)
			}
			if shell {
				if p.APIURL != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "export DSCLI_API_URL=%q\n", p.APIURL)
				}
				if p.Token != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "export DSCLI_TOKEN=%q\n", p.Token)
				}
				if p.SessionID != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "export DSCLI_SESSION_ID=%q\n", p.SessionID)
				}
				if p.Username != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "export DSCLI_USER=%q\n", p.Username)
				}
				if p.Password != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "export DSCLI_PASSWORD=%q\n", p.Password)
				}
				if p.Timeout > 0 {
					fmt.Fprintf(cmd.OutOrStdout(), "export DSCLI_API_TIMEOUT=%q\n", p.Timeout.String())
				}
				return nil
			}
			data := map[string]any{
				"path":         path,
				"name":         name,
				"active":       name == file.ActiveCluster,
				"api_url":      p.APIURL,
				"username":     p.Username,
				"has_token":    p.Token != "",
				"has_session":  p.SessionID != "",
				"has_password": p.Password != "",
				"timeout":      p.Timeout.String(),
			}
			if revealToken {
				data["token"] = p.Token
				data["session_id"] = p.SessionID
				data["password"] = p.Password
			}
			return writeDataEnvelope(cmd, "config.cluster.show", data)
		},
	}
	c.Flags().BoolVar(&revealToken, "reveal-token", false, "Include raw token/session/password in the envelope output")
	c.Flags().BoolVar(&shell, "shell", false, "Print shell `export DSCLI_*=...` lines instead of an envelope")
	return c
}

func newConfigClusterAddCmd() *cobra.Command {
	var apiURL, username, password, token, sessionID string
	var timeout time.Duration
	var activate bool
	c := &cobra.Command{
		Use:   "add <name>",
		Short: "Add or update a named DolphinScheduler API cluster.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := strings.TrimSpace(args[0])
			if name == "" {
				return fmt.Errorf("cluster name cannot be empty")
			}
			if strings.TrimSpace(apiURL) == "" {
				return fmt.Errorf("--api-url is required")
			}
			path, err := dsapi.DefaultConfigPath()
			if err != nil {
				return err
			}
			file, err := dsapi.LoadConfigFile(path)
			if err != nil {
				return err
			}
			if file.Clusters == nil {
				file.Clusters = map[string]dsapi.Profile{}
			}
			file.Clusters[name] = dsapi.Profile{
				Name:      name,
				APIURL:    dsapi.NormalizeBaseURL(apiURL),
				Username:  username,
				Password:  password,
				Token:     token,
				SessionID: sessionID,
				Timeout:   timeout,
			}
			if activate || file.ActiveCluster == "" {
				file.ActiveCluster = name
			}
			if err := dsapi.SaveConfigFile(path, file); err != nil {
				return err
			}
			return writeDataEnvelope(cmd, "config.cluster.add", map[string]any{
				"path":           path,
				"name":           name,
				"active_cluster": file.ActiveCluster,
			})
		},
	}
	c.Flags().StringVar(&apiURL, "api-url", "", "DolphinScheduler API base URL")
	c.Flags().StringVar(&username, "user", "", "DolphinScheduler username")
	c.Flags().StringVar(&password, "password", "", "DolphinScheduler password")
	c.Flags().StringVar(&token, "token", "", "DolphinScheduler access token")
	c.Flags().StringVar(&sessionID, "session-id", "", "DolphinScheduler sessionId")
	c.Flags().DurationVar(&timeout, "timeout", 0, "API timeout, e.g. 30s")
	c.Flags().BoolVar(&activate, "activate", false, "Set this cluster as active_cluster")
	return c
}

func newConfigClusterListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List DolphinScheduler API cluster profiles.",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := dsapi.DefaultConfigPath()
			if err != nil {
				return err
			}
			file, err := dsapi.LoadConfigFile(path)
			if err != nil {
				return err
			}
			names := make([]string, 0, len(file.Clusters))
			for name := range file.Clusters {
				names = append(names, name)
			}
			sort.Strings(names)
			rows := make([]map[string]any, 0, len(names))
			for _, name := range names {
				p := file.Clusters[name]
				rows = append(rows, map[string]any{
					"name":        name,
					"active":      name == file.ActiveCluster,
					"api_url":     p.APIURL,
					"username":    p.Username,
					"has_token":   p.Token != "",
					"has_session": p.SessionID != "",
					"timeout":     p.Timeout.String(),
				})
			}
			return writeDataEnvelope(cmd, "config.cluster.list", map[string]any{
				"path":           path,
				"active_cluster": file.ActiveCluster,
				"clusters":       rows,
			})
		},
	}
}

func newConfigClusterActivateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "activate <name>",
		Short: "Set active DolphinScheduler API cluster profile.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := strings.TrimSpace(args[0])
			path, err := dsapi.DefaultConfigPath()
			if err != nil {
				return err
			}
			file, err := dsapi.LoadConfigFile(path)
			if err != nil {
				return err
			}
			if _, ok := file.Clusters[name]; !ok {
				return fmt.Errorf("cluster %q not found", name)
			}
			file.ActiveCluster = name
			if err := dsapi.SaveConfigFile(path, file); err != nil {
				return err
			}
			return writeDataEnvelope(cmd, "config.cluster.activate", map[string]any{
				"path":           path,
				"active_cluster": name,
			})
		},
	}
}
