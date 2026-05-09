package cmd

import "github.com/spf13/cobra"

var Version = "0.1.0-dev"

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "ds-cli",
		Short:         "ds-cli deploys Apache DolphinScheduler pseudo-clusters locally.",
		Long:          "ds-cli is a single-binary CLI that installs Java, ZooKeeper, and Apache DolphinScheduler 3.4.1 on the local machine, configures MySQL metadata storage, and manages the pseudo-cluster lifecycle.",
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().String("config", "", "path to ds.yaml (default: $DSCLI_CONFIG, ./ds.yaml, ~/.ds-cli/ds.yaml)")
	root.PersistentFlags().Bool("no-color", false, "disable color in stderr progress output")

	root.AddCommand(newPreflightCmd())
	root.AddCommand(newInstallCmd())
	root.AddCommand(newConfigureCmd())
	root.AddCommand(newInitDBCmd())
	root.AddCommand(newStartCmd())
	root.AddCommand(newStopCmd())
	root.AddCommand(newStatusCmd())
	root.AddCommand(newUninstallCmd())
	root.AddCommand(newBootstrapCmd())
	return root
}
