package config

type Cluster struct {
	Name       string `yaml:"name"`
	InstallDir string `yaml:"install_dir"`
	DataDir    string `yaml:"data_dir"`
	User       string `yaml:"user"`
	JavaHome   string `yaml:"java_home"`
	Mode       string `yaml:"mode"`
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
	ClientPort            int    `yaml:"client_port"`
	ExternalConnectString string `yaml:"external_connect_string"`
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

type Plugins struct {
	Task []string `yaml:"task"`
}

type SSH struct {
	Port        int    `yaml:"port"`
	User        string `yaml:"user"`
	PrivateKey  string `yaml:"private_key"`
	Parallelism int    `yaml:"parallelism"`
}

type Host struct {
	Name    string `yaml:"name"`
	Address string `yaml:"address"`
}

type Roles struct {
	ZooKeeper []string `yaml:"zookeeper"`
	API       []string `yaml:"api_server"`
	Master    []string `yaml:"master_server"`
	Worker    []string `yaml:"worker_server"`
	Alert     []string `yaml:"alert_server"`
}

type Config struct {
	Cluster   Cluster   `yaml:"cluster"`
	Versions  Versions  `yaml:"versions"`
	MySQL     MySQL     `yaml:"mysql"`
	ZooKeeper ZooKeeper `yaml:"zookeeper"`
	API       API       `yaml:"api"`
	Services  Services  `yaml:"services"`
	Plugins   Plugins   `yaml:"plugins"`
	SSH       SSH       `yaml:"ssh"`
	Hosts     []Host    `yaml:"hosts"`
	Roles     Roles     `yaml:"roles"`
}

func (c *Config) Distributed() bool {
	return c.Cluster.Mode == "distributed" || len(c.Hosts) > 0
}

func (c *Config) UsesManagedZooKeeper() bool {
	return c.ZooKeeper.ExternalConnectString == ""
}

func (c *Config) HostByName(name string) (Host, bool) {
	for _, h := range c.Hosts {
		if h.Name == name {
			return h, true
		}
	}
	return Host{}, false
}

func (c *Config) AllRoleHosts() []string {
	seen := map[string]struct{}{}
	add := func(xs []string) {
		for _, x := range xs {
			seen[x] = struct{}{}
		}
	}
	add(c.Roles.ZooKeeper)
	add(c.Roles.API)
	add(c.Roles.Master)
	add(c.Roles.Worker)
	add(c.Roles.Alert)
	out := make([]string, 0, len(seen))
	for _, h := range c.Hosts {
		if _, ok := seen[h.Name]; ok {
			out = append(out, h.Name)
		}
	}
	return out
}

func (c *Config) ServiceHosts() []string {
	seen := map[string]struct{}{}
	add := func(xs []string) {
		for _, x := range xs {
			seen[x] = struct{}{}
		}
	}
	add(c.Roles.API)
	add(c.Roles.Master)
	add(c.Roles.Worker)
	add(c.Roles.Alert)
	out := make([]string, 0, len(seen))
	for _, h := range c.Hosts {
		if _, ok := seen[h.Name]; ok {
			out = append(out, h.Name)
		}
	}
	return out
}
