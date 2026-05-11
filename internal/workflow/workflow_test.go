package workflow

import (
	"reflect"
	"strings"
	"testing"

	"github.com/ds-cli/ds-cli/internal/config"
)

func TestStartServicesUsesPseudoClusterOrder(t *testing.T) {
	cfg := config.Default()
	got := StartServices(&cfg)
	want := []string{"api-server", "master-server", "worker-server", "alert-server"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("services = %#v, want %#v", got, want)
	}
}

func TestBootstrapCommands(t *testing.T) {
	got := BootstrapCommands()
	want := []string{"preflight", "install-java", "install-zookeeper", "install-dolphinscheduler", "configure", "init-db", "start", "status"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("commands = %#v, want %#v", got, want)
	}
}

func TestHostServicesUsesDistributedRoles(t *testing.T) {
	cfg := config.Default()
	cfg.Roles.API = []string{"ds1"}
	cfg.Roles.Master = []string{"ds1", "ds2"}
	cfg.Roles.Worker = []string{"ds2"}
	cfg.Roles.Alert = []string{"ds1"}

	got := HostServices(&cfg, "ds1")
	want := []string{"api-server", "master-server", "alert-server"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("services = %#v, want %#v", got, want)
	}
}

func TestAPIWorkerServices(t *testing.T) {
	cfg := config.Default()
	got := APIWorkerServices(&cfg)
	want := []string{"api-server", "worker-server"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("services = %#v, want %#v", got, want)
	}
}

func TestInstallTaskPluginsScriptDownloadsShellAndPython(t *testing.T) {
	cfg := config.Default()
	got := InstallTaskPluginsScript(&cfg)
	for _, want := range []string{
		"dolphinscheduler-task-shell-3.4.1.jar",
		"dolphinscheduler-task-python-3.4.1.jar",
		"plugins/task-plugins",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("script missing %q:\n%s", want, got)
		}
	}
}

func TestStatusServiceScriptChecksEachServiceMain(t *testing.T) {
	cfg := config.Default()
	got := StatusServiceScript(&cfg, []string{"api-server", "master-server", "worker-server", "alert-server"})
	for _, want := range []string{"ApiApplicationServer", "MasterServer", "WorkerServer", "AlertServer", "missing DolphinScheduler service"} {
		if !strings.Contains(got, want) {
			t.Fatalf("status script missing %q:\n%s", want, got)
		}
	}
}

func TestInstallSystemdScriptSetsRestartOnFailure(t *testing.T) {
	cfg := config.Default()
	got := InstallSystemdScript(&cfg, []string{"worker-server"})
	for _, want := range []string{"Restart=on-failure", "dolphinscheduler-worker-server.service", "systemctl enable"} {
		if !strings.Contains(got, want) {
			t.Fatalf("systemd script missing %q:\n%s", want, got)
		}
	}
}
