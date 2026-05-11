package workflow

import (
	"fmt"
	"strings"

	"github.com/ds-cli/ds-cli/internal/config"
)

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

func APIWorkerServices(cfg *config.Config) []string {
	var services []string
	if cfg.Services.API {
		services = append(services, "api-server")
	}
	if cfg.Services.Worker {
		services = append(services, "worker-server")
	}
	return services
}

func HostAPIWorkerServices(cfg *config.Config, host string) []string {
	var services []string
	if contains(cfg.Roles.API, host) {
		services = append(services, "api-server")
	}
	if contains(cfg.Roles.Worker, host) {
		services = append(services, "worker-server")
	}
	return services
}

func SelectServices(available, requested []string) []string {
	var services []string
	for _, svc := range available {
		if contains(requested, svc) {
			services = append(services, svc)
		}
	}
	return services
}

func RestartTargets(cfg *config.Config, components []string) (bool, []string, error) {
	if len(components) == 0 {
		return false, nil, fmt.Errorf("at least one component is required")
	}
	var restartZooKeeper bool
	var services []string
	addService := func(service string) {
		if !contains(services, service) {
			services = append(services, service)
		}
	}
	for _, component := range components {
		switch normalizeComponent(component) {
		case "all":
			if cfg.UsesManagedZooKeeper() {
				restartZooKeeper = true
			}
			for _, svc := range StartServices(cfg) {
				addService(svc)
			}
		case "zookeeper":
			if !cfg.UsesManagedZooKeeper() {
				return false, nil, fmt.Errorf("zookeeper is external; ds-cli cannot restart it")
			}
			restartZooKeeper = true
		case "api-server":
			if !cfg.Services.API && !cfg.Distributed() {
				return false, nil, fmt.Errorf("api-server is disabled in services")
			}
			addService("api-server")
		case "master-server":
			if !cfg.Services.Master && !cfg.Distributed() {
				return false, nil, fmt.Errorf("master-server is disabled in services")
			}
			addService("master-server")
		case "worker-server":
			if !cfg.Services.Worker && !cfg.Distributed() {
				return false, nil, fmt.Errorf("worker-server is disabled in services")
			}
			addService("worker-server")
		case "alert-server":
			if !cfg.Services.Alert && !cfg.Distributed() {
				return false, nil, fmt.Errorf("alert-server is disabled in services")
			}
			addService("alert-server")
		default:
			return false, nil, fmt.Errorf("unsupported restart component %q; supported: api, master, worker, alert, zookeeper, all", component)
		}
	}
	if !restartZooKeeper && len(services) == 0 {
		return false, nil, fmt.Errorf("no restart targets resolved")
	}
	return restartZooKeeper, services, nil
}

func contains(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

func normalizeComponent(component string) string {
	switch strings.ToLower(strings.TrimSpace(component)) {
	case "all":
		return "all"
	case "zk", "zookeeper":
		return "zookeeper"
	case "api", "api-server", "api_server":
		return "api-server"
	case "master", "master-server", "master_server":
		return "master-server"
	case "worker", "worker-server", "worker_server":
		return "worker-server"
	case "alert", "alert-server", "alert_server":
		return "alert-server"
	default:
		return component
	}
}

func BootstrapCommands() []string {
	return []string{"preflight", "install-java", "install-zookeeper", "install-dolphinscheduler", "install-task-plugins", "configure", "init-db", "start", "status"}
}
