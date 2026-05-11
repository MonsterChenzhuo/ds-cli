package render

import (
	"strings"
	"testing"

	"github.com/ds-cli/ds-cli/internal/config"
)

func TestMySQLJDBCURLUsesUserProvidedDatabase(t *testing.T) {
	cfg := config.Default()
	cfg.MySQL.Host = "mysql.internal"
	cfg.MySQL.Port = 3307
	cfg.MySQL.Database = "ds_meta"
	cfg.MySQL.ServerTimezone = "Asia/Shanghai"

	got := MySQLJDBCURL(&cfg)
	wantPrefix := "jdbc:mysql://mysql.internal:3307/ds_meta?"
	if !strings.HasPrefix(got, wantPrefix) {
		t.Fatalf("url = %q, want prefix %q", got, wantPrefix)
	}
	for _, part := range []string{"characterEncoding=UTF-8", "serverTimezone=Asia%2FShanghai", "useSSL=false", "useUnicode=true"} {
		if !strings.Contains(got, part) {
			t.Fatalf("url %q missing %q", got, part)
		}
	}
}

func TestDolphinSchedulerEnvRendersMySQLAndZooKeeper(t *testing.T) {
	cfg := config.Default()
	cfg.MySQL.Username = "ds_user"
	cfg.MySQL.Password = "secret"

	env := DolphinSchedulerEnv(&cfg)
	for _, want := range []string{
		"export DATABASE=${DATABASE:-mysql}",
		`export SPRING_DATASOURCE_USERNAME="ds_user"`,
		`export SPRING_DATASOURCE_PASSWORD="secret"`,
		"export REGISTRY_TYPE=${REGISTRY_TYPE:-zookeeper}",
		"localhost:2181",
	} {
		if !strings.Contains(env, want) {
			t.Fatalf("env missing %q:\n%s", want, env)
		}
	}
}

func TestDolphinSchedulerEnvRendersUserEnvironment(t *testing.T) {
	cfg := config.Default()
	cfg.Env.JavaHome = "/data/hadoopclient/JDK/jdk1.8.0_272"
	cfg.Env.PythonLauncher = "/usr/bin/python3"
	cfg.Env.HadoopUserName = "airflow"
	cfg.Env.HadoopHome = "/data/hadoopclient/HDFS/hadoop"
	cfg.Env.PathPrepend = []string{"$HADOOP_HOME/bin", "$HADOOP_HOME/sbin"}
	cfg.Env.Exports = map[string]string{
		"SPARK_HOME": "/data/spark",
	}

	env := DolphinSchedulerEnv(&cfg)
	for _, want := range []string{
		"export JAVA_HOME=${JAVA_HOME:-/data/hadoopclient/JDK/jdk1.8.0_272}",
		"export PYTHON_LAUNCHER='/usr/bin/python3'",
		"export HADOOP_USER_NAME='airflow'",
		"export HADOOP_HOME='/data/hadoopclient/HDFS/hadoop'",
		"export SPARK_HOME='/data/spark'",
		"export PATH=$JAVA_HOME/bin:$HADOOP_HOME/bin:$HADOOP_HOME/sbin:$PATH",
	} {
		if !strings.Contains(env, want) {
			t.Fatalf("env missing %q:\n%s", want, env)
		}
	}
}

func TestZooKeeperConnectStringUsesExternalValue(t *testing.T) {
	cfg := config.Default()
	cfg.ZooKeeper.ExternalConnectString = "zk1:2181,zk2:2181,zk3:2181"

	got := ZooKeeperConnectString(&cfg)
	if got != "zk1:2181,zk2:2181,zk3:2181" {
		t.Fatalf("connect string = %q", got)
	}
}

func TestZooKeeperConnectStringUsesDistributedRoles(t *testing.T) {
	cfg := config.Default()
	cfg.Cluster.Mode = "distributed"
	cfg.Hosts = []config.Host{
		{Name: "ds1", Address: "10.0.0.1"},
		{Name: "ds2", Address: "10.0.0.2"},
		{Name: "ds3", Address: "10.0.0.3"},
	}
	cfg.Roles.ZooKeeper = []string{"ds1", "ds2", "ds3"}

	got := ZooKeeperConnectString(&cfg)
	want := "10.0.0.1:2181,10.0.0.2:2181,10.0.0.3:2181"
	if got != want {
		t.Fatalf("connect string = %q, want %q", got, want)
	}
}
