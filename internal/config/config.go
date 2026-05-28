package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type FeedConfig struct {
	URL      string `yaml:"url"`
	Name     string `yaml:"name"`
	Category string `yaml:"category"`
}

type Config struct {
	Feeds      []FeedConfig `yaml:"feeds"`
	KoboPath   string       `yaml:"kobo_path"`
	MaxPerFeed int          `yaml:"max_per_feed"`
	DBPath     string       `yaml:"db_path"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if len(cfg.Feeds) == 0 {
		return nil, fmt.Errorf("at least one feed is required")
	}
	for i, f := range cfg.Feeds {
		if f.Category == "" {
			cfg.Feeds[i].Category = "news"
		}
	}
	if cfg.MaxPerFeed == 0 {
		cfg.MaxPerFeed = 5
	}
	if cfg.DBPath == "" {
		cfg.DBPath = "briefme.db"
	}

	return &cfg, nil
}
