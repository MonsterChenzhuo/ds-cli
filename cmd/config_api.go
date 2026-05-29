package cmd

import (
	"fmt"
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
	cmd.AddCommand(newConfigClusterCmd())
	return cmd
}

func newConfigClusterCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "Manage named DolphinScheduler API clusters.",
	}
	cmd.AddCommand(newConfigClusterAddCmd())
	cmd.AddCommand(newConfigClusterListCmd())
	cmd.AddCommand(newConfigClusterActivateCmd())
	return cmd
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
