package cmd

import (
	"github.com/ds-cli/ds-cli/internal/output"
	"github.com/ds-cli/ds-cli/internal/workflow"
	"github.com/spf13/cobra"
)

func newPreflightCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "preflight",
		Short: "Check local prerequisites for pseudo-cluster deployment.",
		RunE: func(cmd *cobra.Command, args []string) error {
			rc, err := prepare(cmd, "preflight")
			if err != nil {
				return err
			}
			ctx, cancel := commandCtx()
			defer cancel()
			e := rc.envelope("preflight")
			rc.runStep(ctx, e, "preflight", workflow.PreflightScript(rc.Cfg))
			return finish(rc, e)
		},
	}
}

func newInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Install Java, ZooKeeper, DolphinScheduler, and MySQL JDBC driver.",
		RunE: func(cmd *cobra.Command, args []string) error {
			rc, err := prepare(cmd, "install")
			if err != nil {
				return err
			}
			ctx, cancel := commandCtx()
			defer cancel()
			e := rc.envelope("install")
			for _, step := range []struct{ name, script string }{
				{"install-java", workflow.InstallJavaScript(rc.Cfg)},
				{"install-zookeeper", workflow.InstallZooKeeperScript(rc.Cfg)},
				{"install-dolphinscheduler", workflow.InstallDolphinSchedulerScript(rc.Cfg)},
			} {
				if !rc.runStep(ctx, e, step.name, step.script) {
					break
				}
			}
			return finish(rc, e)
		},
	}
}

func newConfigureCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "configure",
		Short: "Render DolphinScheduler configuration for MySQL and ZooKeeper.",
		RunE: func(cmd *cobra.Command, args []string) error {
			rc, err := prepare(cmd, "configure")
			if err != nil {
				return err
			}
			ctx, cancel := commandCtx()
			defer cancel()
			e := rc.envelope("configure")
			rc.runStep(ctx, e, "configure-dolphinscheduler", workflow.ConfigureScript(rc.Cfg))
			return finish(rc, e)
		},
	}
}

func newInitDBCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init-db",
		Short: "Initialize DolphinScheduler metadata schema in the configured MySQL database.",
		RunE: func(cmd *cobra.Command, args []string) error {
			rc, err := prepare(cmd, "init-db")
			if err != nil {
				return err
			}
			ctx, cancel := commandCtx()
			defer cancel()
			e := rc.envelope("init-db")
			rc.runStep(ctx, e, "init-db", workflow.InitDBScript(rc.Cfg))
			return finish(rc, e)
		},
	}
}

func newStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start ZooKeeper and DolphinScheduler services.",
		RunE: func(cmd *cobra.Command, args []string) error {
			rc, err := prepare(cmd, "start")
			if err != nil {
				return err
			}
			ctx, cancel := commandCtx()
			defer cancel()
			e := rc.envelope("start")
			if rc.runStep(ctx, e, "start-zookeeper", workflow.StartZooKeeperScript(rc.Cfg)) {
				rc.runStep(ctx, e, "start-dolphinscheduler", workflow.ServiceScript(rc.Cfg, "start", workflow.StartServices(rc.Cfg)))
			}
			return finish(rc, e)
		},
	}
}

func newStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop DolphinScheduler services and ZooKeeper.",
		RunE: func(cmd *cobra.Command, args []string) error {
			rc, err := prepare(cmd, "stop")
			if err != nil {
				return err
			}
			ctx, cancel := commandCtx()
			defer cancel()
			e := rc.envelope("stop")
			rc.runStep(ctx, e, "stop-dolphinscheduler", workflow.ServiceScript(rc.Cfg, "stop", workflow.StopServices(rc.Cfg)))
			rc.runStep(ctx, e, "stop-zookeeper", workflow.StopZooKeeperScript(rc.Cfg))
			return finish(rc, e)
		},
	}
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check ZooKeeper and DolphinScheduler service status.",
		RunE: func(cmd *cobra.Command, args []string) error {
			rc, err := prepare(cmd, "status")
			if err != nil {
				return err
			}
			ctx, cancel := commandCtx()
			defer cancel()
			e := rc.envelope("status")
			rc.runStep(ctx, e, "status-zookeeper", workflow.StatusZooKeeperScript(rc.Cfg))
			rc.runStep(ctx, e, "status-dolphinscheduler", workflow.ServiceScript(rc.Cfg, "status", workflow.StartServices(rc.Cfg)))
			return finish(rc, e)
		},
	}
}

func newUninstallCmd() *cobra.Command {
	var purgeData bool
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove DolphinScheduler and ZooKeeper installation directories.",
		RunE: func(cmd *cobra.Command, args []string) error {
			rc, err := prepare(cmd, "uninstall")
			if err != nil {
				return err
			}
			ctx, cancel := commandCtx()
			defer cancel()
			e := rc.envelope("uninstall")
			rc.runStep(ctx, e, "uninstall", workflow.UninstallScript(rc.Cfg, purgeData))
			return finish(rc, e)
		},
	}
	cmd.Flags().BoolVar(&purgeData, "purge-data", false, "also remove cluster.data_dir")
	return cmd
}

func newBootstrapCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "bootstrap",
		Short: "Run preflight, install, configure, init-db, start, and status.",
		RunE: func(cmd *cobra.Command, args []string) error {
			rc, err := prepare(cmd, "bootstrap")
			if err != nil {
				return err
			}
			ctx, cancel := commandCtx()
			defer cancel()
			e := rc.envelope("bootstrap")
			steps := []struct{ name, script string }{
				{"preflight", workflow.PreflightScript(rc.Cfg)},
				{"install-java", workflow.InstallJavaScript(rc.Cfg)},
				{"install-zookeeper", workflow.InstallZooKeeperScript(rc.Cfg)},
				{"install-dolphinscheduler", workflow.InstallDolphinSchedulerScript(rc.Cfg)},
				{"configure", workflow.ConfigureScript(rc.Cfg)},
				{"init-db", workflow.InitDBScript(rc.Cfg)},
				{"start-zookeeper", workflow.StartZooKeeperScript(rc.Cfg)},
				{"start-dolphinscheduler", workflow.ServiceScript(rc.Cfg, "start", workflow.StartServices(rc.Cfg))},
				{"status-dolphinscheduler", workflow.ServiceScript(rc.Cfg, "status", workflow.StartServices(rc.Cfg))},
			}
			for _, step := range steps {
				if !rc.runStep(ctx, e, step.name, step.script) {
					e.WithError(output.EnvelopeError{Code: "STEP_FAILED", Message: step.name + " failed", Hint: "read ~/.ds-cli/runs/" + rc.Run.ID + "/" + step.name + ".stderr"})
					break
				}
			}
			return finish(rc, e)
		},
	}
}
