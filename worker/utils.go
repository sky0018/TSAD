package worker

import (
	"time"
	"code.byted.org/microservice/tsad/worker/detector"
	"regexp"
)

// Config .
type Config struct {
	LogLevel   string `yaml:"LogLevel"`
	LogPath    string `yaml:"LogPath"`
	WorkerPort string `yaml:"WorkerPort"`

	MysqlDSN string `yaml:"MysqlDSN"`

	TSDBAPI     string        `yaml:"TSDBAPI"`
	TSDBRetry   int           `yaml:"TSDBRetry"`
	TSDBTimeout time.Duration `yaml:"TSDBTimeout"`

	AlertAddress string `yaml:"AlertAddress"`

	MaxTasks int `yaml:"MaxTasks"`

	WhiteSourceList []string `yaml:"WhiteSourceList"`
	BlackSourceList []string `yaml:"BlackSourceList"`
}

var (
	whiteMachines []*regexp.Regexp
	blackMachines []*regexp.Regexp
)

func initWhiteBlackList(c *Config) error {
	if len(c.WhiteSourceList) > 0 {
		whiteMachines = make([]*regexp.Regexp, 0, len(config.WhiteSourceList))
		for _, reg := range c.WhiteSourceList {
			m, err := regexp.Compile(reg)
			if err != nil {
				return err
			}
			whiteMachines = append(whiteMachines, m)
		}
	}

	if len(c.BlackSourceList) > 0 {
		blackMachines = make([]*regexp.Regexp, 0, len(config.BlackSourceList))
		for _, reg := range c.BlackSourceList {
			m, err := regexp.Compile(reg)
			if err != nil {
				return err
			}
			blackMachines = append(blackMachines, m)
		}
	}

	return nil
}

func inWhiteList(t detector.TaskMeta) bool {
	if len(whiteMachines) > 0 {
		for _, m := range whiteMachines {
			if m.MatchString(t.DataSource.Key) {
				return true
			}
			if m.MatchString(t.DataSource.Extra) {
				return true
			}
		}
	}
	return false
}

func inBlackList(t detector.TaskMeta) bool {
	if len(blackMachines) > 0 {
		for _, m := range blackMachines {
			if m.MatchString(t.DataSource.Key) {
				return true
			}
			if m.MatchString(t.DataSource.Extra) {
				return true
			}
		}
	}
	return false
}

func taskIsAllowed(t detector.TaskMeta) bool {
	if inWhiteList(t) {
		return true
	}
	if inBlackList(t) {
		return false
	}
	return true
}
