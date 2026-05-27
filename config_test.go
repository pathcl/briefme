package main

import (
	"os"
	"testing"
)

func TestLoadConfig_Valid(t *testing.T) {
	yaml := `
feeds:
  - url: "https://example.com/feed.xml"
    name: "Example"
kobo_email: "me@kobo.com"
smtp:
  host: "smtp.gmail.com"
  port: 587
  username: "user@gmail.com"
  password: "secret"
  from: "user@gmail.com"
max_articles: 10
`
	f := writeTempFile(t, yaml)
	cfg, err := LoadConfig(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Feeds) != 1 {
		t.Errorf("expected 1 feed, got %d", len(cfg.Feeds))
	}
	if cfg.Feeds[0].URL != "https://example.com/feed.xml" {
		t.Errorf("unexpected feed URL: %s", cfg.Feeds[0].URL)
	}
	if cfg.KoboEmail != "me@kobo.com" {
		t.Errorf("unexpected kobo_email: %s", cfg.KoboEmail)
	}
	if cfg.MaxArticles != 10 {
		t.Errorf("expected max_articles 10, got %d", cfg.MaxArticles)
	}
}

func TestLoadConfig_MissingKoboEmail(t *testing.T) {
	yaml := `
feeds:
  - url: "https://example.com/feed.xml"
    name: "Example"
smtp:
  host: "smtp.gmail.com"
  port: 587
  username: "user@gmail.com"
  password: "secret"
  from: "user@gmail.com"
`
	f := writeTempFile(t, yaml)
	_, err := LoadConfig(f)
	if err == nil {
		t.Fatal("expected error for missing kobo_email")
	}
}

func TestLoadConfig_NoFeeds(t *testing.T) {
	yaml := `
feeds: []
kobo_email: "me@kobo.com"
smtp:
  host: "smtp.gmail.com"
  port: 587
  username: "user@gmail.com"
  password: "secret"
  from: "user@gmail.com"
`
	f := writeTempFile(t, yaml)
	_, err := LoadConfig(f)
	if err == nil {
		t.Fatal("expected error for empty feeds")
	}
}

func TestLoadConfig_DefaultMaxArticles(t *testing.T) {
	yaml := `
feeds:
  - url: "https://example.com/feed.xml"
    name: "Example"
kobo_email: "me@kobo.com"
smtp:
  host: "smtp.gmail.com"
  port: 587
  username: "user@gmail.com"
  password: "secret"
  from: "user@gmail.com"
`
	f := writeTempFile(t, yaml)
	cfg, err := LoadConfig(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.MaxArticles != 20 {
		t.Errorf("expected default max_articles 20, got %d", cfg.MaxArticles)
	}
}

func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "briefme-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(f.Name()) })
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return f.Name()
}
