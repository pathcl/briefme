package main

import (
	"archive/zip"
	"os"
	"strings"
	"testing"
	"time"
)

func TestBuildEPUB_CreatesFile(t *testing.T) {
	articles := []Article{
		{
			Title:       "Hello World",
			Author:      "Alice",
			URL:         "https://example.com/1",
			Content:     "<p>First article content</p>",
			PublishedAt: time.Date(2025, 5, 26, 10, 0, 0, 0, time.UTC),
			FeedName:    "Test Feed",
		},
		{
			Title:   "Second Article",
			Content: "<p>Second article content</p>",
			URL:     "https://example.com/2",
		},
	}

	out := tempEPUBPath(t)
	if err := BuildEPUB(articles, out); err != nil {
		t.Fatalf("BuildEPUB error: %v", err)
	}

	if _, err := os.Stat(out); err != nil {
		t.Fatalf("EPUB file not created: %v", err)
	}
}

func TestBuildEPUB_ValidZipStructure(t *testing.T) {
	articles := []Article{
		{Title: "Test", Content: "<p>content</p>", URL: "https://example.com/1"},
	}
	out := tempEPUBPath(t)
	if err := BuildEPUB(articles, out); err != nil {
		t.Fatalf("BuildEPUB error: %v", err)
	}

	zr, err := zip.OpenReader(out)
	if err != nil {
		t.Fatalf("not a valid zip/epub: %v", err)
	}
	defer zr.Close()

	names := make(map[string]bool)
	for _, f := range zr.File {
		names[f.Name] = true
	}

	required := []string{"mimetype", "META-INF/container.xml"}
	for _, r := range required {
		if !names[r] {
			t.Errorf("missing required EPUB file: %s", r)
		}
	}

	// Must have at least one HTML content file
	hasHTML := false
	for name := range names {
		if strings.HasSuffix(name, ".xhtml") || strings.HasSuffix(name, ".html") {
			hasHTML = true
			break
		}
	}
	if !hasHTML {
		t.Error("EPUB contains no HTML content files")
	}
}

func TestBuildEPUB_EmptyArticles(t *testing.T) {
	out := tempEPUBPath(t)
	err := BuildEPUB([]Article{}, out)
	if err == nil {
		t.Fatal("expected error for empty article list")
	}
}

func tempEPUBPath(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp("", "briefme-*.epub")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })
	return f.Name()
}
