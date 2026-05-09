package packages

import "fmt"

type Spec struct {
	Name      string
	Version   string
	URL       string
	Filename  string
	SHA512URL string
}

func DolphinSchedulerSpec(version string) (Spec, error) {
	if version != "3.4.1" {
		return Spec{}, fmt.Errorf("unsupported dolphinscheduler version %s", version)
	}
	filename := fmt.Sprintf("apache-dolphinscheduler-%s-bin.tar.gz", version)
	url := fmt.Sprintf("https://archive.apache.org/dist/dolphinscheduler/%s/%s", version, filename)
	return Spec{Name: "dolphinscheduler", Version: version, URL: url, Filename: filename, SHA512URL: url + ".sha512"}, nil
}

func ZooKeeperSpec(version string) Spec {
	filename := fmt.Sprintf("apache-zookeeper-%s-bin.tar.gz", version)
	url := fmt.Sprintf("https://archive.apache.org/dist/zookeeper/zookeeper-%s/%s", version, filename)
	return Spec{Name: "zookeeper", Version: version, URL: url, Filename: filename, SHA512URL: url + ".sha512"}
}

func MySQLDriverSpec(version string) Spec {
	filename := fmt.Sprintf("mysql-connector-j-%s.jar", version)
	url := fmt.Sprintf("https://repo1.maven.org/maven2/com/mysql/mysql-connector-j/%s/%s", version, filename)
	return Spec{Name: "mysql-driver", Version: version, URL: url, Filename: filename}
}
