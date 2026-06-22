package cmd

import "github.com/spf13/cobra"

var Version = "0.1.0-dev"

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "ds-cli",
		Short:         "Operate Apache DolphinScheduler through its REST API.",
		Long:          "ds-cli is a non-interactive, single-binary CLI for AI agents and operators that manage existing Apache DolphinScheduler clusters through the REST API. API commands emit one structured JSON envelope on stdout so callers can parse projects, workflows, tasks, schedules, alerts, and environments deterministically.",
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(newConfigCmd())
	root.AddCommand(newProjectCmd())
	root.AddCommand(newWorkflowCmd())
	root.AddCommand(newWorkflowInstanceCmd())
	root.AddCommand(newTaskCmd())
	root.AddCommand(newTaskInstanceCmd())
	root.AddCommand(newTaskDefCmd())
	root.AddCommand(newScheduleCmd())
	root.AddCommand(newAlertCmd())
	root.AddCommand(newEnvironmentCmd())
	return root
}
