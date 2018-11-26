package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"code.byted.org/microservice/tsad/manager"
	"code.byted.org/microservice/tsad/worker"
	"gopkg.in/yaml.v2"
)

// Config .
type Config struct {
	DebugPort int            `yaml:"DebugPort"`
	Manager   manager.Config `yaml:"Manager"`
	Worker    worker.Config  `yaml:"Worker"`
}

func loadConfig() (*Config, error) {
	confenv := os.Getenv("CONF_ENV")
	confpath := "./conf/config.yml"
	if confenv != "" {
		confpath += "." + confenv
	}

	confbuf, err := ioutil.ReadFile(confpath)
	if err != nil {
		return nil, fmt.Errorf("open config file err: %v", err)
	}

	var config Config
	if err := yaml.Unmarshal(confbuf, &config); err != nil {
		return nil, fmt.Errorf("unmarshal yaml err: %v", err)
	}

	// print config
	if buf, err := json.MarshalIndent(config, "", "  "); err == nil {
		fmt.Println("config >>>>>>>>>>>>>>>>>>>>>>>>>>>>>")
		fmt.Println(string(buf))
		fmt.Println()
	} else {
		fmt.Println(config)
	}

	return &config, nil
}
