package workflow

import "github.com/ds-cli/ds-cli/internal/config"

func StartServices(cfg *config.Config) []string {
	var services []string
	if cfg.Services.API {
		services = append(services, "api-server")
	}
	if cfg.Services.Master {
		services = append(services, "master-server")
	}
	if cfg.Services.Worker {
		services = append(services, "worker-server")
	}
	if cfg.Services.Alert {
		services = append(services, "alert-server")
	}
	return services
}

func StopServices(cfg *config.Config) []string {
	start := StartServices(cfg)
	for i, j := 0, len(start)-1; i < j; i, j = i+1, j-1 {
		start[i], start[j] = start[j], start[i]
	}
	return start
}

func BootstrapCommands() []string {
	return []string{"preflight", "install-java", "install-zookeeper", "install-dolphinscheduler", "configure", "init-db", "start", "status"}
}
