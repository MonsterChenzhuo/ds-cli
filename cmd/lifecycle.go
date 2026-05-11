package cmd

import (
	"github.com/ds-cli/ds-cli/internal/output"
	"github.com/ds-cli/ds-cli/internal/workflow"
	"github.com/spf13/cobra"
)

func newPreflightCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "preflight",
		Short: "Check prerequisites for pseudo-cluster or distributed deployment.",
		RunE: func(cmd *cobra.Command, args []string) error {
			rc, err := prepare(cmd, "preflight")
			if err != nil {
				return err
			}
			ctx, cancel := commandCtx()
			defer cancel()
			e := rc.envelope("preflight")
			if rc.Cfg.Distributed() {
				rc.runRemoteSameStep(ctx, e, "preflight", rc.Cfg.AllRoleHosts(), workflow.PreflightScript(rc.Cfg))
				return finish(rc, e)
			}
			rc.runStep(ctx, e, "preflight", workflow.PreflightScript(rc.Cfg))
			return finish(rc, e)
		},
	}
}

func newInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Install Java, managed ZooKeeper, DolphinScheduler, and MySQL JDBC driver.",
		RunE: func(cmd *cobra.Command, args []string) error {
			rc, err := prepare(cmd, "install")
			if err != nil {
				return err
			}
			ctx, cancel := commandCtx()
			defer cancel()
			e := rc.envelope("install")
			if rc.Cfg.Distributed() {
				hosts := rc.Cfg.AllRoleHosts()
				if !rc.runRemoteSameStep(ctx, e, "install-java", hosts, workflow.InstallJavaScript(rc.Cfg)) {
					return finish(rc, e)
				}
				if rc.Cfg.UsesManagedZooKeeper() &&
					!rc.runRemoteSameStep(ctx, e, "install-zookeeper", rc.Cfg.Roles.ZooKeeper, workflow.InstallZooKeeperScript(rc.Cfg)) {
					return finish(rc, e)
				}
				rc.runRemoteSameStep(ctx, e, "install-dolphinscheduler", rc.Cfg.ServiceHosts(), workflow.InstallDolphinSchedulerScript(rc.Cfg))
				return finish(rc, e)
			}
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
		Short: "Render DolphinScheduler and managed ZooKeeper configuration.",
		RunE: func(cmd *cobra.Command, args []string) error {
			rc, err := prepare(cmd, "configure")
			if err != nil {
				return err
			}
			ctx, cancel := commandCtx()
			defer cancel()
			e := rc.envelope("configure")
			if rc.Cfg.Distributed() {
				if rc.Cfg.UsesManagedZooKeeper() &&
					!rc.runRemoteStep(ctx, e, "configure-zookeeper", rc.Cfg.Roles.ZooKeeper, func(host string) string {
						return workflow.ConfigureZooKeeperScript(rc.Cfg, host)
					}) {
					return finish(rc, e)
				}
				rc.runRemoteSameStep(ctx, e, "configure-dolphinscheduler", rc.Cfg.ServiceHosts(), workflow.ConfigureDolphinSchedulerScript(rc.Cfg))
				return finish(rc, e)
			}
			rc.runStep(ctx, e, "configure-dolphinscheduler", workflow.ConfigureScript(rc.Cfg))
			return finish(rc, e)
		},
	}
}

func newPluginsCmd() *cobra.Command {
	var restart bool
	cmd := &cobra.Command{
		Use:   "plugins",
		Short: "Install configured DolphinScheduler task plugins and optionally restart api/worker.",
		RunE: func(cmd *cobra.Command, args []string) error {
			rc, err := prepare(cmd, "plugins")
			if err != nil {
				return err
			}
			ctx, cancel := commandCtx()
			defer cancel()
			e := rc.envelope("plugins")
			if rc.Cfg.Distributed() {
				hosts := rc.Cfg.ServiceHosts()
				if !rc.runRemoteSameStep(ctx, e, "install-task-plugins", hosts, workflow.InstallTaskPluginsScript(rc.Cfg)) {
					return finish(rc, e)
				}
				if restart {
					for _, host := range hosts {
						services := workflow.HostAPIWorkerServices(rc.Cfg, host)
						if len(services) > 0 {
							rc.runRemoteSameStep(ctx, e, "restart-api-worker", []string{host}, workflow.RestartServiceScript(rc.Cfg, services))
						}
					}
				}
				return finish(rc, e)
			}
			if rc.runStep(ctx, e, "install-task-plugins", workflow.InstallTaskPluginsScript(rc.Cfg)) && restart {
				rc.runStep(ctx, e, "restart-api-worker", workflow.RestartServiceScript(rc.Cfg, workflow.APIWorkerServices(rc.Cfg)))
			}
			return finish(rc, e)
		},
	}
	cmd.Flags().BoolVar(&restart, "restart", true, "restart api-server and worker-server after installing plugins")
	return cmd
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
			if rc.Cfg.Distributed() {
				host := rc.Cfg.Roles.API[0]
				rc.runRemoteSameStep(ctx, e, "init-db", []string{host}, workflow.InitDBScript(rc.Cfg))
				return finish(rc, e)
			}
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
			if rc.Cfg.Distributed() {
				if rc.Cfg.UsesManagedZooKeeper() &&
					!rc.runRemoteSameStep(ctx, e, "start-zookeeper", rc.Cfg.Roles.ZooKeeper, workflow.StartZooKeeperScript(rc.Cfg)) {
					return finish(rc, e)
				}
				for _, host := range rc.Cfg.ServiceHosts() {
					rc.runRemoteSameStep(ctx, e, "start-dolphinscheduler", []string{host}, workflow.ServiceScript(rc.Cfg, "start", workflow.HostServices(rc.Cfg, host)))
				}
				return finish(rc, e)
			}
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
			if rc.Cfg.Distributed() {
				for _, host := range rc.Cfg.ServiceHosts() {
					rc.runRemoteSameStep(ctx, e, "stop-dolphinscheduler", []string{host}, workflow.ServiceScript(rc.Cfg, "stop", workflow.HostServicesReverse(rc.Cfg, host)))
				}
				if rc.Cfg.UsesManagedZooKeeper() {
					rc.runRemoteSameStep(ctx, e, "stop-zookeeper", rc.Cfg.Roles.ZooKeeper, workflow.StopZooKeeperScript(rc.Cfg))
				}
				return finish(rc, e)
			}
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
			if rc.Cfg.Distributed() {
				if rc.Cfg.UsesManagedZooKeeper() {
					rc.runRemoteSameStep(ctx, e, "status-zookeeper", rc.Cfg.Roles.ZooKeeper, workflow.StatusZooKeeperScript(rc.Cfg))
				}
				for _, host := range rc.Cfg.ServiceHosts() {
					rc.runRemoteSameStep(ctx, e, "status-dolphinscheduler", []string{host}, workflow.StatusServiceScript(rc.Cfg, workflow.HostServices(rc.Cfg, host)))
				}
				return finish(rc, e)
			}
			rc.runStep(ctx, e, "status-zookeeper", workflow.StatusZooKeeperScript(rc.Cfg))
			rc.runStep(ctx, e, "status-dolphinscheduler", workflow.StatusServiceScript(rc.Cfg, workflow.StartServices(rc.Cfg)))
			return finish(rc, e)
		},
	}
}

func newSystemdCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "systemd",
		Short: "Install systemd units with Restart=on-failure for DolphinScheduler services.",
		RunE: func(cmd *cobra.Command, args []string) error {
			rc, err := prepare(cmd, "systemd")
			if err != nil {
				return err
			}
			ctx, cancel := commandCtx()
			defer cancel()
			e := rc.envelope("systemd")
			if rc.Cfg.Distributed() {
				for _, host := range rc.Cfg.ServiceHosts() {
					services := workflow.HostServices(rc.Cfg, host)
					if len(services) > 0 {
						rc.runRemoteSameStep(ctx, e, "install-systemd", []string{host}, workflow.InstallSystemdScript(rc.Cfg, services))
					}
				}
				return finish(rc, e)
			}
			rc.runStep(ctx, e, "install-systemd", workflow.InstallSystemdScript(rc.Cfg, workflow.StartServices(rc.Cfg)))
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
			if rc.Cfg.Distributed() {
				if rc.Cfg.UsesManagedZooKeeper() {
					rc.runRemoteSameStep(ctx, e, "uninstall", rc.Cfg.AllRoleHosts(), workflow.UninstallScript(rc.Cfg, purgeData))
				} else {
					rc.runRemoteSameStep(ctx, e, "uninstall", rc.Cfg.ServiceHosts(), workflow.UninstallScript(rc.Cfg, purgeData))
				}
				return finish(rc, e)
			}
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
			if rc.Cfg.Distributed() {
				if !rc.runRemoteSameStep(ctx, e, "preflight", rc.Cfg.AllRoleHosts(), workflow.PreflightScript(rc.Cfg)) {
					return finish(rc, e)
				}
				if !rc.runRemoteSameStep(ctx, e, "install-java", rc.Cfg.AllRoleHosts(), workflow.InstallJavaScript(rc.Cfg)) {
					return finish(rc, e)
				}
				if rc.Cfg.UsesManagedZooKeeper() {
					if !rc.runRemoteSameStep(ctx, e, "install-zookeeper", rc.Cfg.Roles.ZooKeeper, workflow.InstallZooKeeperScript(rc.Cfg)) {
						return finish(rc, e)
					}
					if !rc.runRemoteStep(ctx, e, "configure-zookeeper", rc.Cfg.Roles.ZooKeeper, func(host string) string {
						return workflow.ConfigureZooKeeperScript(rc.Cfg, host)
					}) {
						return finish(rc, e)
					}
				}
				if !rc.runRemoteSameStep(ctx, e, "install-dolphinscheduler", rc.Cfg.ServiceHosts(), workflow.InstallDolphinSchedulerScript(rc.Cfg)) {
					return finish(rc, e)
				}
				if !rc.runRemoteSameStep(ctx, e, "install-task-plugins", rc.Cfg.ServiceHosts(), workflow.InstallTaskPluginsScript(rc.Cfg)) {
					return finish(rc, e)
				}
				if !rc.runRemoteSameStep(ctx, e, "configure-dolphinscheduler", rc.Cfg.ServiceHosts(), workflow.ConfigureDolphinSchedulerScript(rc.Cfg)) {
					return finish(rc, e)
				}
				if !rc.runRemoteSameStep(ctx, e, "init-db", []string{rc.Cfg.Roles.API[0]}, workflow.InitDBScript(rc.Cfg)) {
					return finish(rc, e)
				}
				if rc.Cfg.UsesManagedZooKeeper() &&
					!rc.runRemoteSameStep(ctx, e, "start-zookeeper", rc.Cfg.Roles.ZooKeeper, workflow.StartZooKeeperScript(rc.Cfg)) {
					return finish(rc, e)
				}
				for _, host := range rc.Cfg.ServiceHosts() {
					if !rc.runRemoteSameStep(ctx, e, "start-dolphinscheduler", []string{host}, workflow.ServiceScript(rc.Cfg, "start", workflow.HostServices(rc.Cfg, host))) {
						return finish(rc, e)
					}
				}
				return finish(rc, e)
			}
			steps := []struct{ name, script string }{
				{"preflight", workflow.PreflightScript(rc.Cfg)},
				{"install-java", workflow.InstallJavaScript(rc.Cfg)},
				{"install-zookeeper", workflow.InstallZooKeeperScript(rc.Cfg)},
				{"install-dolphinscheduler", workflow.InstallDolphinSchedulerScript(rc.Cfg)},
				{"install-task-plugins", workflow.InstallTaskPluginsScript(rc.Cfg)},
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
