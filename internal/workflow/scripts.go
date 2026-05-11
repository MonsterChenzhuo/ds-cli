package workflow

import (
	"fmt"
	"strings"

	"github.com/ds-cli/ds-cli/internal/config"
	"github.com/ds-cli/ds-cli/internal/packages"
	"github.com/ds-cli/ds-cli/internal/render"
)

func DSHome(cfg *config.Config) string {
	return fmt.Sprintf("%s/dolphinscheduler-%s", cfg.Cluster.InstallDir, cfg.Versions.DolphinScheduler)
}

func ZKHome(cfg *config.Config) string {
	return fmt.Sprintf("%s/zookeeper-%s", cfg.Cluster.InstallDir, cfg.Versions.ZooKeeper)
}

func PreflightScript(cfg *config.Config) string {
	checkMySQL := "true"
	if cfg.MySQL.CreateDatabase {
		checkMySQL = "command -v mysql >/dev/null"
	}
	return fmt.Sprintf(`set -e
uname -s >/dev/null
command -v bash >/dev/null
command -v curl >/dev/null
command -v tar >/dev/null
%s
test -n %q
test -n %q
`, checkMySQL, cfg.MySQL.Username, cfg.MySQL.Host)
}

func InstallJavaScript(cfg *config.Config) string {
	return fmt.Sprintf(`set -e
detect_java_home() {
  if command -v /usr/libexec/java_home >/dev/null 2>&1; then
    /usr/libexec/java_home -v 11 2>/dev/null && return 0
  fi
  if [ -d /usr/lib/jvm ]; then
    found="$(find /usr/lib/jvm -maxdepth 1 -type d \( -name 'java-11*' -o -name 'jdk-11*' \) | head -n 1)"
    if [ -n "$found" ] && [ -x "$found/bin/java" ]; then
      printf '%%s\n' "$found"
      return 0
    fi
  fi
  if command -v java >/dev/null 2>&1; then
    java_bin="$(command -v java)"
    if command -v readlink >/dev/null 2>&1; then
      resolved="$(readlink -f "$java_bin" 2>/dev/null || true)"
      if [ -n "$resolved" ]; then java_bin="$resolved"; fi
    fi
    candidate="$(cd "$(dirname "$java_bin")/.." && pwd)"
    if [ -x "$candidate/bin/java" ]; then
      printf '%%s\n' "$candidate"
      return 0
    fi
  fi
  return 1
}
if [ -x %q/bin/java ]; then exit 0; fi
if java_home="$(detect_java_home)"; then
  sudo mkdir -p "$(dirname %q)"
  sudo ln -sfn "$java_home" %q
  exit 0
fi
if command -v apt-get >/dev/null 2>&1; then
  sudo apt-get update
  sudo DEBIAN_FRONTEND=noninteractive apt-get install -y openjdk-11-jdk
elif command -v dnf >/dev/null 2>&1; then
  sudo dnf install -y java-11-openjdk-devel
elif command -v yum >/dev/null 2>&1; then
  sudo yum install -y java-11-openjdk-devel
elif command -v brew >/dev/null 2>&1; then
  brew install openjdk@11
else
  echo "no supported Java installer found; install JDK 11 or set cluster.java_home" >&2
  exit 1
fi
java_home="$(detect_java_home)"
sudo mkdir -p "$(dirname %q)"
sudo ln -sfn "$java_home" %q
test -x %q/bin/java
`, cfg.Cluster.JavaHome, cfg.Cluster.JavaHome, cfg.Cluster.JavaHome, cfg.Cluster.JavaHome, cfg.Cluster.JavaHome, cfg.Cluster.JavaHome)
}

func InstallZooKeeperScript(cfg *config.Config) string {
	spec := packages.ZooKeeperSpec(cfg.Versions.ZooKeeper)
	home := ZKHome(cfg)
	return fmt.Sprintf(`set -e
mkdir -p %q %q/packages %q/zookeeper/data %q/zookeeper/logs
if [ -x %q/bin/zkServer.sh ]; then exit 0; fi
curl -fL -o %q/packages/%s %q
tar -xzf %q/packages/%s -C %q
rm -rf %q
mv %q/apache-zookeeper-%s-bin %q
cat > %q/conf/zoo.cfg <<'EOF'
tickTime=2000
initLimit=10
syncLimit=5
dataDir=%s/zookeeper/data
dataLogDir=%s/zookeeper/logs
clientPort=%d
admin.enableServer=false
EOF
`, cfg.Cluster.InstallDir, cfg.Cluster.InstallDir, cfg.Cluster.DataDir, cfg.Cluster.DataDir, home,
		cfg.Cluster.InstallDir, spec.Filename, spec.URL,
		cfg.Cluster.InstallDir, spec.Filename, cfg.Cluster.InstallDir,
		home, cfg.Cluster.InstallDir, cfg.Versions.ZooKeeper, home,
		home, cfg.Cluster.DataDir, cfg.Cluster.DataDir, cfg.ZooKeeper.ClientPort)
}

func ConfigureZooKeeperScript(cfg *config.Config, host string) string {
	home := ZKHome(cfg)
	myID := 1
	for i, h := range cfg.Roles.ZooKeeper {
		if h == host {
			myID = i + 1
			break
		}
	}
	var servers strings.Builder
	for i, name := range cfg.Roles.ZooKeeper {
		h, ok := cfg.HostByName(name)
		if !ok {
			continue
		}
		servers.WriteString(fmt.Sprintf("server.%d=%s:2888:3888\n", i+1, h.Address))
	}
	return fmt.Sprintf(`set -e
mkdir -p %q/conf %q/zookeeper/data %q/zookeeper/logs
cat > %q/conf/zoo.cfg <<'EOF'
tickTime=2000
initLimit=10
syncLimit=5
dataDir=%s/zookeeper/data
dataLogDir=%s/zookeeper/logs
clientPort=%d
admin.enableServer=false
%sEOF
printf '%%s\n' %d > %q/zookeeper/data/myid
`, home, cfg.Cluster.DataDir, cfg.Cluster.DataDir, home,
		cfg.Cluster.DataDir, cfg.Cluster.DataDir, cfg.ZooKeeper.ClientPort, servers.String(), myID, cfg.Cluster.DataDir)
}

func InstallDolphinSchedulerScript(cfg *config.Config) string {
	dsSpec, _ := packages.DolphinSchedulerSpec(cfg.Versions.DolphinScheduler)
	driver := packages.MySQLDriverSpec(cfg.Versions.MySQLDriver)
	home := DSHome(cfg)
	return fmt.Sprintf(`set -e
mkdir -p %q %q/packages %q/dolphinscheduler
if [ -x %q/bin/dolphinscheduler-daemon.sh ]; then exit 0; fi
curl -fL -o %q/packages/%s %q
tar -xzf %q/packages/%s -C %q
rm -rf %q
mv %q/apache-dolphinscheduler-%s-bin %q
curl -fL -o %q/packages/%s %q
for d in api-server/libs alert-server/libs master-server/libs worker-server/libs tools/libs; do
  mkdir -p %q/$d
  cp %q/packages/%s %q/$d/
done
`, cfg.Cluster.InstallDir, cfg.Cluster.InstallDir, cfg.Cluster.DataDir,
		home, cfg.Cluster.InstallDir, dsSpec.Filename, dsSpec.URL,
		cfg.Cluster.InstallDir, dsSpec.Filename, cfg.Cluster.InstallDir,
		home, cfg.Cluster.InstallDir, cfg.Versions.DolphinScheduler, home,
		cfg.Cluster.InstallDir, driver.Filename, driver.URL,
		home, cfg.Cluster.InstallDir, driver.Filename, home)
}

func ConfigureScript(cfg *config.Config) string {
	return fmt.Sprintf(`set -e
test -d %q
cat > %q/bin/env/dolphinscheduler_env.sh <<'EOF'
%sEOF
`, DSHome(cfg), DSHome(cfg), render.DolphinSchedulerEnv(cfg))
}

func ConfigureDolphinSchedulerScript(cfg *config.Config) string {
	return ConfigureScript(cfg)
}

func InitDBScript(cfg *config.Config) string {
	create := "true"
	if cfg.MySQL.CreateDatabase {
		create = fmt.Sprintf(`mysql -h %q -P %d -u %q -p%q -e "CREATE DATABASE IF NOT EXISTS %s DEFAULT CHARACTER SET utf8 DEFAULT COLLATE utf8_general_ci;"`,
			cfg.MySQL.Host, cfg.MySQL.Port, cfg.MySQL.AdminUsername, cfg.MySQL.AdminPassword, cfg.MySQL.Database)
	}
	return fmt.Sprintf(`set -e
%s
cd %q
export JAVA_HOME=%q
bash tools/bin/upgrade-schema.sh
`, create, DSHome(cfg), cfg.Cluster.JavaHome)
}

func StartZooKeeperScript(cfg *config.Config) string {
	return fmt.Sprintf(`set -e
export JAVA_HOME=%q
if %q/bin/zkServer.sh status >/dev/null 2>&1; then exit 0; fi
%q/bin/zkServer.sh start
`, cfg.Cluster.JavaHome, ZKHome(cfg), ZKHome(cfg))
}

func StopZooKeeperScript(cfg *config.Config) string {
	return fmt.Sprintf(`set -e
export JAVA_HOME=%q
%q/bin/zkServer.sh stop || true
`, cfg.Cluster.JavaHome, ZKHome(cfg))
}

func StatusZooKeeperScript(cfg *config.Config) string {
	return fmt.Sprintf(`set -e
export JAVA_HOME=%q
%q/bin/zkServer.sh status
`, cfg.Cluster.JavaHome, ZKHome(cfg))
}

func ServiceScript(cfg *config.Config, action string, services []string) string {
	var b strings.Builder
	b.WriteString("set -e\n")
	b.WriteString(fmt.Sprintf("cd %q\n", DSHome(cfg)))
	b.WriteString(fmt.Sprintf("export JAVA_HOME=%q\n", cfg.Cluster.JavaHome))
	for _, svc := range services {
		b.WriteString(fmt.Sprintf("bash ./bin/dolphinscheduler-daemon.sh %s %s\n", action, svc))
	}
	return b.String()
}

func UninstallScript(cfg *config.Config, purgeData bool) string {
	script := fmt.Sprintf(`set -e
rm -rf %q %q
`, DSHome(cfg), ZKHome(cfg))
	if purgeData {
		script += fmt.Sprintf("rm -rf %q\n", cfg.Cluster.DataDir)
	}
	return script
}
