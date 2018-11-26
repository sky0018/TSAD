package manager

// Config .
type Config struct {
	// base config
	LogLevel    string `yaml:"LogLevel"`
	LogPath     string `yaml:"LogPath"`
	ManagerPort string `yaml:"ManagerPort"`
	WorkerPort  string `yaml:"WorkerPort"`

	// config for dal
	DSNRead  string `yaml:"DSNRead"`
	DSNWrite string `yaml:"DSNWrite"`

	// config for dist locker
	MysqlLocklDSN string `yaml:"MysqlLocklDSN"`
}
