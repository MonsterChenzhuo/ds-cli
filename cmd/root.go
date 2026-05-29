package cmd

import "github.com/spf13/cobra"

var Version = "0.1.0-dev"

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "ds-cli",
		Short:         "AI-first Apache DolphinScheduler CLI for Codex and Claude Code.",
		Long:          "ds-cli is a non-interactive, single-binary CLI for AI agents such as Codex and Claude Code. It deploys Apache DolphinScheduler 3.4.1, manages pseudo-cluster or distributed lifecycle, and exposes REST API helpers for projects, workflows, tasks, schedules, alerts, and environments. Commands are designed for machine parsing: stdout is a JSON envelope and stderr is reserved for progress.",
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().String("config", "", "path to ds.yaml (default: $DSCLI_CONFIG, ./ds.yaml, ~/.ds-cli/ds.yaml)")
	root.PersistentFlags().Bool("no-color", false, "disable color in stderr progress output")

	root.AddCommand(newPreflightCmd())
	root.AddCommand(newConfigCmd())
	root.AddCommand(newInstallCmd())
	root.AddCommand(newPluginsCmd())
	root.AddCommand(newConfigureCmd())
	root.AddCommand(newInitDBCmd())
	root.AddCommand(newStartCmd())
	root.AddCommand(newStopCmd())
	root.AddCommand(newRestartCmd())
	root.AddCommand(newStatusCmd())
	root.AddCommand(newSystemdCmd())
	root.AddCommand(newUninstallCmd())
	root.AddCommand(newBootstrapCmd())
	root.AddCommand(newProjectCmd())
	root.AddCommand(newWorkflowCmd())
	root.AddCommand(newTaskCmd())
	root.AddCommand(newScheduleCmd())
	root.AddCommand(newAlertCmd())
	root.AddCommand(newEnvironmentCmd())
	return root
}
