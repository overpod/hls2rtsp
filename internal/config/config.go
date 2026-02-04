package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server  ServerConfig            `yaml:"server"`
	Auth    AuthConfig              `yaml:"auth"`
	Streams map[string]StreamConfig `yaml:"streams"`
	Metrics MetricsConfig           `yaml:"metrics"`
}

type ServerConfig struct {
	Port string `yaml:"port"`
}

type AuthConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type StreamConfig struct {
	URL string `yaml:"url"`
}

type MetricsConfig struct {
	Enabled  bool          `yaml:"enabled"`
	Interval time.Duration `yaml:"interval"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := &Config{
		Server: ServerConfig{
			Port: "8554",
		},
		Metrics: MetricsConfig{
			Enabled:  true,
			Interval: 30 * time.Second,
		},
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if len(cfg.Streams) == 0 {
		return nil, fmt.Errorf("no streams configured")
	}

	for name, s := range cfg.Streams {
		if s.URL == "" {
			return nil, fmt.Errorf("stream %q: url is required", name)
		}
	}

	return cfg, nil
}
