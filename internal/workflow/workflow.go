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

func HostServices(cfg *config.Config, host string) []string {
	var services []string
	if contains(cfg.Roles.API, host) {
		services = append(services, "api-server")
	}
	if contains(cfg.Roles.Master, host) {
		services = append(services, "master-server")
	}
	if contains(cfg.Roles.Worker, host) {
		services = append(services, "worker-server")
	}
	if contains(cfg.Roles.Alert, host) {
		services = append(services, "alert-server")
	}
	return services
}

func HostServicesReverse(cfg *config.Config, host string) []string {
	services := HostServices(cfg, host)
	for i, j := 0, len(services)-1; i < j; i, j = i+1, j-1 {
		services[i], services[j] = services[j], services[i]
	}
	return services
}

func contains(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

func BootstrapCommands() []string {
	return []string{"preflight", "install-java", "install-zookeeper", "install-dolphinscheduler", "configure", "init-db", "start", "status"}
}
