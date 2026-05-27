package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type FeedConfig struct {
	URL  string `yaml:"url"`
	Name string `yaml:"name"`
}

type SMTPConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	From     string `yaml:"from"`
}

type Config struct {
	Feeds       []FeedConfig `yaml:"feeds"`
	KoboEmail   string       `yaml:"kobo_email"`
	SMTP        SMTPConfig   `yaml:"smtp"`
	MaxArticles int          `yaml:"max_articles"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if cfg.KoboEmail == "" {
		return nil, fmt.Errorf("kobo_email is required")
	}
	if len(cfg.Feeds) == 0 {
		return nil, fmt.Errorf("at least one feed is required")
	}
	if cfg.SMTP.Host == "" {
		return nil, fmt.Errorf("smtp.host is required")
	}

	if cfg.MaxArticles == 0 {
		cfg.MaxArticles = 20
	}

	return &cfg, nil
}
