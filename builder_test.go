package main

import (
	"archive/zip"
	"fmt"
	"io"
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

func TestBuildEPUB_HasIndexPage(t *testing.T) {
	articles := []Article{
		{Title: "Article One", Content: "<p>one</p>", URL: "https://example.com/1"},
		{Title: "Article Two", Content: "<p>two</p>", URL: "https://example.com/2"},
	}
	out := tempEPUBPath(t)
	if err := BuildEPUB(articles, out); err != nil {
		t.Fatalf("BuildEPUB error: %v", err)
	}

	content := readEPUBFile(t, out, "EPUB/xhtml/index.xhtml")
	if !strings.Contains(content, "Article One") {
		t.Error("index page missing first article title")
	}
	if !strings.Contains(content, "Article Two") {
		t.Error("index page missing second article title")
	}
}

func TestBuildEPUB_IndexLinksToArticles(t *testing.T) {
	articles := []Article{
		{Title: "First", Content: "<p>one</p>", URL: "https://example.com/1"},
		{Title: "Second", Content: "<p>two</p>", URL: "https://example.com/2"},
	}
	out := tempEPUBPath(t)
	if err := BuildEPUB(articles, out); err != nil {
		t.Fatalf("BuildEPUB error: %v", err)
	}

	index := readEPUBFile(t, out, "EPUB/xhtml/index.xhtml")
	for i := range articles {
		link := fmt.Sprintf("article-%03d.xhtml", i+1)
		if !strings.Contains(index, link) {
			t.Errorf("index missing link to %s", link)
		}
	}
}

func TestBuildEPUB_ArticlesHaveBackLink(t *testing.T) {
	articles := []Article{
		{Title: "Only Article", Content: "<p>content</p>", URL: "https://example.com/1"},
	}
	out := tempEPUBPath(t)
	if err := BuildEPUB(articles, out); err != nil {
		t.Fatalf("BuildEPUB error: %v", err)
	}

	article := readEPUBFile(t, out, "EPUB/xhtml/article-001.xhtml")
	if !strings.Contains(article, "index.xhtml") {
		t.Error("article missing back link to index.xhtml")
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

func readEPUBFile(t *testing.T, epubPath, internalPath string) string {
	t.Helper()
	zr, err := zip.OpenReader(epubPath)
	if err != nil {
		t.Fatalf("open epub: %v", err)
	}
	defer zr.Close()
	for _, f := range zr.File {
		if f.Name == internalPath {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("open %s: %v", internalPath, err)
			}
			defer rc.Close()
			data, err := io.ReadAll(rc)
			if err != nil {
				t.Fatalf("read %s: %v", internalPath, err)
			}
			return string(data)
		}
	}
	t.Fatalf("file %q not found in EPUB", internalPath)
	return ""
}
