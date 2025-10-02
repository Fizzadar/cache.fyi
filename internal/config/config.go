package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type CachefyiConfig struct {
	Database           string        `yaml:"database"`
	AuthHeader         string        `yaml:"authHeader"`
	ListenAddr         string        `yaml:"listenAddr"`
	ProcessInterval    time.Duration `yaml:"processInterval"`
	LinkwardenURL      string        `yaml:"linkwardenAPIURL"`
	LinkwardenAPIToken string        `yaml:"linkwardenAPIToken"`
}

func NewCachefyiConfig(filename string) CachefyiConfig {
	data, err := os.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	var cfg CachefyiConfig
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		panic(err)
	}

	return cfg
}
