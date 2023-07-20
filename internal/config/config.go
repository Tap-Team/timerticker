package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int64  `yaml:"port"`
}

type Config struct {
	ServerConfig ServerConfig `yaml:"server"`
}

func New(serverConfig ServerConfig) *Config {
	return &Config{ServerConfig: serverConfig}
}

func FromFile(
	filepath string,
) *Config {
	b, err := os.ReadFile(filepath)
	if err != nil {
		log.Fatalf("error while read file, %s", err)
	}
	config := new(Config)
	err = yaml.Unmarshal(b, config)
	if err != nil {
		log.Fatalf("error while unmarshal file, %s", err)
	}
	return config
}
