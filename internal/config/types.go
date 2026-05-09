package config

type Cluster struct {
	Name       string `yaml:"name"`
	InstallDir string `yaml:"install_dir"`
	DataDir    string `yaml:"data_dir"`
	User       string `yaml:"user"`
	JavaHome   string `yaml:"java_home"`
}

type Versions struct {
	DolphinScheduler string `yaml:"dolphinscheduler"`
	ZooKeeper        string `yaml:"zookeeper"`
	Java             string `yaml:"java"`
	MySQLDriver      string `yaml:"mysql_driver"`
}

type MySQL struct {
	Host           string `yaml:"host"`
	Port           int    `yaml:"port"`
	Database       string `yaml:"database"`
	Username       string `yaml:"username"`
	Password       string `yaml:"password"`
	AdminUsername  string `yaml:"admin_username"`
	AdminPassword  string `yaml:"admin_password"`
	ServerTimezone string `yaml:"server_timezone"`
	CreateDatabase bool   `yaml:"create_database"`
}

type ZooKeeper struct {
	ClientPort int `yaml:"client_port"`
}

type API struct {
	Port int `yaml:"port"`
}

type Services struct {
	API    bool `yaml:"api"`
	Master bool `yaml:"master"`
	Worker bool `yaml:"worker"`
	Alert  bool `yaml:"alert"`
}

type Config struct {
	Cluster   Cluster   `yaml:"cluster"`
	Versions  Versions  `yaml:"versions"`
	MySQL     MySQL     `yaml:"mysql"`
	ZooKeeper ZooKeeper `yaml:"zookeeper"`
	API       API       `yaml:"api"`
	Services  Services  `yaml:"services"`
}
