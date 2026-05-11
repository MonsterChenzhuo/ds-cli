package render

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/ds-cli/ds-cli/internal/config"
)

func MySQLJDBCURL(cfg *config.Config) string {
	params := url.Values{}
	params.Set("useUnicode", "true")
	params.Set("characterEncoding", "UTF-8")
	params.Set("useSSL", "false")
	params.Set("serverTimezone", cfg.MySQL.ServerTimezone)
	return fmt.Sprintf("jdbc:mysql://%s:%d/%s?%s", cfg.MySQL.Host, cfg.MySQL.Port, cfg.MySQL.Database, params.Encode())
}

func DolphinSchedulerEnv(cfg *config.Config) string {
	pathParts := []string{"$JAVA_HOME/bin"}
	pathParts = append(pathParts, cfg.Env.PathPrepend...)
	lines := []string{
		fmt.Sprintf("export JAVA_HOME=${JAVA_HOME:-%s}", RuntimeJavaHome(cfg)),
		"export DATABASE=${DATABASE:-mysql}",
		"export SPRING_PROFILES_ACTIVE=${DATABASE}",
		fmt.Sprintf("export SPRING_DATASOURCE_URL=%q", MySQLJDBCURL(cfg)),
		fmt.Sprintf("export SPRING_DATASOURCE_USERNAME=%q", cfg.MySQL.Username),
		fmt.Sprintf("export SPRING_DATASOURCE_PASSWORD=%q", cfg.MySQL.Password),
		"export SPRING_CACHE_TYPE=${SPRING_CACHE_TYPE:-none}",
		"export SPRING_JACKSON_TIME_ZONE=${SPRING_JACKSON_TIME_ZONE:-UTC}",
		"export REGISTRY_TYPE=${REGISTRY_TYPE:-zookeeper}",
		fmt.Sprintf("export REGISTRY_ZOOKEEPER_CONNECT_STRING=${REGISTRY_ZOOKEEPER_CONNECT_STRING:-%s}", ZooKeeperConnectString(cfg)),
	}
	if cfg.Env.PythonLauncher != "" {
		lines = append(lines, fmt.Sprintf("export PYTHON_LAUNCHER=%s", shellQuote(cfg.Env.PythonLauncher)))
	}
	if cfg.Env.HadoopUserName != "" {
		lines = append(lines, fmt.Sprintf("export HADOOP_USER_NAME=%s", shellQuote(cfg.Env.HadoopUserName)))
	}
	if cfg.Env.HadoopHome != "" {
		lines = append(lines, fmt.Sprintf("export HADOOP_HOME=%s", shellQuote(cfg.Env.HadoopHome)))
	}
	for _, name := range sortedExportNames(cfg.Env.Exports) {
		lines = append(lines, fmt.Sprintf("export %s=%s", name, shellQuote(cfg.Env.Exports[name])))
	}
	lines = append(lines, fmt.Sprintf("export PATH=%s:$PATH", strings.Join(pathParts, ":")))
	return strings.Join(lines, "\n") + "\n"
}

func RuntimeJavaHome(cfg *config.Config) string {
	if strings.TrimSpace(cfg.Env.JavaHome) != "" {
		return cfg.Env.JavaHome
	}
	return cfg.Cluster.JavaHome
}

func ZooKeeperConnectString(cfg *config.Config) string {
	if cfg.ZooKeeper.ExternalConnectString != "" {
		return cfg.ZooKeeper.ExternalConnectString
	}
	if !cfg.Distributed() {
		return fmt.Sprintf("localhost:%d", cfg.ZooKeeper.ClientPort)
	}
	var parts []string
	for _, name := range cfg.Roles.ZooKeeper {
		if h, ok := cfg.HostByName(name); ok {
			parts = append(parts, fmt.Sprintf("%s:%d", h.Address, cfg.ZooKeeper.ClientPort))
		}
	}
	return strings.Join(parts, ",")
}

func sortedExportNames(exports map[string]string) []string {
	names := make([]string, 0, len(exports))
	for name := range exports {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
