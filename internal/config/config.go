package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type GlobalConfig struct {
	DefaultProtocol string `yaml:"default_protocol"`
	LogLevel        string `yaml:"log_level"`
	MCPPort         int    `yaml:"mcp_port"`
	Workers         int    `yaml:"workers"`
}

func BeamDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".beam")
}

func LogsDir() string {
	return filepath.Join(BeamDir(), "logs")
}

func KeysDir() string {
	return filepath.Join(BeamDir(), "keys")
}

func KnownHostsPath() string {
	return filepath.Join(BeamDir(), "known_hosts")
}

func defaults() *GlobalConfig {
	return &GlobalConfig{
		DefaultProtocol: "sftp",
		LogLevel:        "info",
		MCPPort:         7850,
		Workers:         3,
	}
}

func Load() (*GlobalConfig, error) {
	path := filepath.Join(BeamDir(), "beam.yaml")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return defaults(), nil
	}
	if err != nil {
		return nil, err
	}
	cfg := defaults()
	return cfg, yaml.Unmarshal(data, cfg)
}

func (c *GlobalConfig) Save() error {
	if err := os.MkdirAll(BeamDir(), 0o700); err != nil {
		return err
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(BeamDir(), "beam.yaml"), data, 0o600)
}
