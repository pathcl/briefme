package config_test

import (
	"os"
	"testing"

	"github.com/pathcl/briefme/internal/config"
)

func TestLoad_Valid(t *testing.T) {
	f := writeTempFile(t, `
feeds:
  - url: "https://example.com/feed.xml"
    name: "Example"
kobo_path: "/media/user/KOBOeReader"
max_per_feed: 10
`)
	cfg, err := config.Load(f)
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

func TestLoad_NoFeeds(t *testing.T) {
	f := writeTempFile(t, `feeds: []`)
	if _, err := config.Load(f); err == nil {
		t.Fatal("expected error for empty feeds")
	}
}

func TestLoad_DefaultMaxPerFeed(t *testing.T) {
	f := writeTempFile(t, `
feeds:
  - url: "https://example.com/feed.xml"
    name: "Example"
`)
	cfg, err := config.Load(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.MaxPerFeed != 5 {
		t.Errorf("expected default max_per_feed 5, got %d", cfg.MaxPerFeed)
	}
}

func TestLoad_DefaultCategoryIsNews(t *testing.T) {
	f := writeTempFile(t, `
feeds:
  - url: "https://example.com/feed.xml"
    name: "Example"
`)
	cfg, err := config.Load(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Feeds[0].Category != "news" {
		t.Errorf("expected default category 'news', got %q", cfg.Feeds[0].Category)
	}
}

func TestLoad_ExplicitCategory(t *testing.T) {
	f := writeTempFile(t, `
feeds:
  - url: "https://arxiv.org/rss/cs.AI"
    name: "arXiv"
    category: "papers"
`)
	cfg, err := config.Load(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Feeds[0].Category != "papers" {
		t.Errorf("expected category 'papers', got %q", cfg.Feeds[0].Category)
	}
}

func TestLoad_KoboPathOptional(t *testing.T) {
	f := writeTempFile(t, `
feeds:
  - url: "https://example.com/feed.xml"
    name: "Example"
`)
	cfg, err := config.Load(f)
	if err != nil {
		t.Fatalf("kobo_path should be optional: %v", err)
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
	f.WriteString(content)
	f.Close()
	return f.Name()
}
