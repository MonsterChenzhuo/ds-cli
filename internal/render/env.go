package render

import (
	"fmt"
	"net/url"
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
	lines := []string{
		fmt.Sprintf("export JAVA_HOME=${JAVA_HOME:-%s}", cfg.Cluster.JavaHome),
		"export DATABASE=${DATABASE:-mysql}",
		"export SPRING_PROFILES_ACTIVE=${DATABASE}",
		fmt.Sprintf("export SPRING_DATASOURCE_URL=%q", MySQLJDBCURL(cfg)),
		fmt.Sprintf("export SPRING_DATASOURCE_USERNAME=%q", cfg.MySQL.Username),
		fmt.Sprintf("export SPRING_DATASOURCE_PASSWORD=%q", cfg.MySQL.Password),
		"export SPRING_CACHE_TYPE=${SPRING_CACHE_TYPE:-none}",
		"export SPRING_JACKSON_TIME_ZONE=${SPRING_JACKSON_TIME_ZONE:-UTC}",
		"export REGISTRY_TYPE=${REGISTRY_TYPE:-zookeeper}",
		fmt.Sprintf("export REGISTRY_ZOOKEEPER_CONNECT_STRING=${REGISTRY_ZOOKEEPER_CONNECT_STRING:-localhost:%d}", cfg.ZooKeeper.ClientPort),
		"export PATH=$JAVA_HOME/bin:$PATH",
	}
	return strings.Join(lines, "\n") + "\n"
}
