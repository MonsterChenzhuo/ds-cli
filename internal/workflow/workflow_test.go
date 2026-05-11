package workflow

import (
	"reflect"
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
