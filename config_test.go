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
kobo_path: "/media/user/KOBOeReader"
max_per_feed: 10
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
	if cfg.KoboPath != "/media/user/KOBOeReader" {
		t.Errorf("unexpected kobo_path: %s", cfg.KoboPath)
	}
	if cfg.MaxPerFeed != 10 {
		t.Errorf("expected max_per_feed 10, got %d", cfg.MaxPerFeed)
	}
}

func TestLoadConfig_NoFeeds(t *testing.T) {
	yaml := `
feeds: []
kobo_path: "/media/user/KOBOeReader"
`
	f := writeTempFile(t, yaml)
	_, err := LoadConfig(f)
	if err == nil {
		t.Fatal("expected error for empty feeds")
	}
}

func TestLoadConfig_DefaultMaxPerFeed(t *testing.T) {
	yaml := `
feeds:
  - url: "https://example.com/feed.xml"
    name: "Example"
`
	f := writeTempFile(t, yaml)
	cfg, err := LoadConfig(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.MaxPerFeed != 5 {
		t.Errorf("expected default max_per_feed 5, got %d", cfg.MaxPerFeed)
	}
}

func TestLoadConfig_DefaultCategoryIsNews(t *testing.T) {
	yaml := `
feeds:
  - url: "https://example.com/feed.xml"
    name: "Example"
`
	f := writeTempFile(t, yaml)
	cfg, err := LoadConfig(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Feeds[0].Category != "news" {
		t.Errorf("expected default category 'news', got %q", cfg.Feeds[0].Category)
	}
}

func TestLoadConfig_ExplicitCategory(t *testing.T) {
	yaml := `
feeds:
  - url: "https://arxiv.org/rss/cs.AI"
    name: "arXiv"
    category: "papers"
`
	f := writeTempFile(t, yaml)
	cfg, err := LoadConfig(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Feeds[0].Category != "papers" {
		t.Errorf("expected category 'papers', got %q", cfg.Feeds[0].Category)
	}
}

func TestLoadConfig_KoboPathOptional(t *testing.T) {
	yaml := `
feeds:
  - url: "https://example.com/feed.xml"
    name: "Example"
`
	f := writeTempFile(t, yaml)
	cfg, err := LoadConfig(f)
	if err != nil {
		t.Fatalf("kobo_path should be optional (auto-detect at runtime): %v", err)
	}
	if cfg.KoboPath != "" {
		t.Errorf("expected empty kobo_path, got %s", cfg.KoboPath)
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
