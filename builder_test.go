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
	out := tempEPUBPath(t)
	if err := BuildEPUB(twoArticles(), out, "Briefme News – 2026-05-28"); err != nil {
		t.Fatalf("BuildEPUB error: %v", err)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("EPUB file not created: %v", err)
	}
}

func TestBuildEPUB_TitleAppearsInOutput(t *testing.T) {
	out := tempEPUBPath(t)
	title := "Briefme Papers – 2026-05-28"
	if err := BuildEPUB(twoArticles(), out, title); err != nil {
		t.Fatalf("BuildEPUB error: %v", err)
	}
	opf := readEPUBFile(t, out, "EPUB/package.opf")
	if !strings.Contains(opf, title) {
		t.Errorf("OPF does not contain title %q", title)
	}
	index := readEPUBFile(t, out, "EPUB/xhtml/index.xhtml")
	if !strings.Contains(index, title) {
		t.Errorf("index page does not contain title %q", title)
	}
}

func TestBuildEPUB_ValidZipStructure(t *testing.T) {
	out := tempEPUBPath(t)
	if err := BuildEPUB(twoArticles(), out, "Test"); err != nil {
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
	for _, required := range []string{"mimetype", "META-INF/container.xml", "EPUB/package.opf"} {
		if !names[required] {
			t.Errorf("missing required EPUB entry: %s", required)
		}
	}
}

func TestBuildEPUB_HasIndexPage(t *testing.T) {
	articles := []Article{
		{Title: "Article One", Content: "<p>one</p>", URL: "https://example.com/1"},
		{Title: "Article Two", Content: "<p>two</p>", URL: "https://example.com/2"},
	}
	out := tempEPUBPath(t)
	if err := BuildEPUB(articles, out, "Test"); err != nil {
		t.Fatalf("BuildEPUB error: %v", err)
	}

	index := readEPUBFile(t, out, "EPUB/xhtml/index.xhtml")
	if !strings.Contains(index, "Article One") {
		t.Error("index missing first article title")
	}
	if !strings.Contains(index, "Article Two") {
		t.Error("index missing second article title")
	}
}

func TestBuildEPUB_IndexLinksToArticles(t *testing.T) {
	articles := []Article{
		{Title: "First",  Content: "<p>one</p>", URL: "https://example.com/1"},
		{Title: "Second", Content: "<p>two</p>", URL: "https://example.com/2"},
	}
	out := tempEPUBPath(t)
	if err := BuildEPUB(articles, out, "Test"); err != nil {
		t.Fatalf("BuildEPUB error: %v", err)
	}

	index := readEPUBFile(t, out, "EPUB/xhtml/index.xhtml")
	for i := range articles {
		link := fmt.Sprintf(`href="article-%03d.xhtml"`, i+1)
		if !strings.Contains(index, link) {
			t.Errorf("index missing <a href> link to article-%03d.xhtml", i+1)
		}
	}
}

func TestBuildEPUB_ArticlesHaveBackLink(t *testing.T) {
	out := tempEPUBPath(t)
	if err := BuildEPUB(twoArticles(), out, "Test"); err != nil {
		t.Fatalf("BuildEPUB error: %v", err)
	}

	article := readEPUBFile(t, out, "EPUB/xhtml/article-001.xhtml")
	if !strings.Contains(article, `href="index.xhtml"`) {
		t.Error("article missing <a href=\"index.xhtml\"> back link")
	}
}

func TestBuildEPUB_ArticleContentPresent(t *testing.T) {
	articles := []Article{
		{Title: "My Article", Content: "<p>The body of the article lives here.</p>", URL: "https://example.com/1"},
	}
	out := tempEPUBPath(t)
	if err := BuildEPUB(articles, out, "Test"); err != nil {
		t.Fatalf("BuildEPUB error: %v", err)
	}

	body := readEPUBFile(t, out, "EPUB/xhtml/article-001.xhtml")
	if !strings.Contains(body, "The body of the article lives here.") {
		t.Error("article body text missing from article file")
	}
}

func TestBuildEPUB_HTMLStrippedFromContent(t *testing.T) {
	articles := []Article{
		{
			Title:   "Tagged",
			Content: `<p>Clean text. <a href="https://evil.com"><img src="x"/></a></p>`,
			URL:     "https://example.com/1",
		},
	}
	out := tempEPUBPath(t)
	if err := BuildEPUB(articles, out, "Test"); err != nil {
		t.Fatalf("BuildEPUB error: %v", err)
	}

	body := readEPUBFile(t, out, "EPUB/xhtml/article-001.xhtml")
	if strings.Contains(body, "evil.com") {
		t.Error("raw link URL leaked into article content")
	}
	if strings.Contains(body, "<img") {
		t.Error("img tag leaked into article content")
	}
	if !strings.Contains(body, "Clean text.") {
		t.Error("readable text was lost during HTML stripping")
	}
}

func TestBuildEPUB_ArticleMetadata(t *testing.T) {
	articles := []Article{
		{
			Title:       "Metadata Test",
			Author:      "Jane Doe",
			FeedName:    "Test Feed",
			PublishedAt: time.Date(2026, 5, 28, 0, 0, 0, 0, time.UTC),
			Content:     "<p>body</p>",
			URL:         "https://example.com/1",
		},
	}
	out := tempEPUBPath(t)
	if err := BuildEPUB(articles, out, "Test"); err != nil {
		t.Fatalf("BuildEPUB error: %v", err)
	}

	body := readEPUBFile(t, out, "EPUB/xhtml/article-001.xhtml")
	for _, want := range []string{"Metadata Test", "Jane Doe", "Test Feed", "May 28, 2026"} {
		if !strings.Contains(body, want) {
			t.Errorf("article file missing metadata field %q", want)
		}
	}
}

func TestBuildEPUB_SpecialCharsEscaped(t *testing.T) {
	articles := []Article{
		{
			Title:   `<script>alert("xss")</script>`,
			Content: "<p>body</p>",
			URL:     "https://example.com/1",
		},
	}
	out := tempEPUBPath(t)
	if err := BuildEPUB(articles, out, "Test"); err != nil {
		t.Fatalf("BuildEPUB error: %v", err)
	}

	for _, path := range []string{"EPUB/xhtml/index.xhtml", "EPUB/xhtml/article-001.xhtml"} {
		body := readEPUBFile(t, out, path)
		if strings.Contains(body, "<script>") {
			t.Errorf("%s contains unescaped <script> tag", path)
		}
		if !strings.Contains(body, "&lt;script&gt;") {
			t.Errorf("%s missing HTML-escaped title", path)
		}
	}
}

func TestBuildEPUB_OPFManifestComplete(t *testing.T) {
	articles := []Article{
		{Title: "One",   Content: "<p>a</p>", URL: "https://example.com/1"},
		{Title: "Two",   Content: "<p>b</p>", URL: "https://example.com/2"},
		{Title: "Three", Content: "<p>c</p>", URL: "https://example.com/3"},
	}
	out := tempEPUBPath(t)
	if err := BuildEPUB(articles, out, "Test"); err != nil {
		t.Fatalf("BuildEPUB error: %v", err)
	}

	opf := readEPUBFile(t, out, "EPUB/package.opf")
	for i := range articles {
		entry := fmt.Sprintf("article-%03d.xhtml", i+1)
		if !strings.Contains(opf, entry) {
			t.Errorf("OPF manifest missing entry for %s", entry)
		}
	}
	if !strings.Contains(opf, "index.xhtml") {
		t.Error("OPF manifest missing index.xhtml")
	}
}

func TestBuildEPUB_SpineOrderIndexFirst(t *testing.T) {
	out := tempEPUBPath(t)
	if err := BuildEPUB(twoArticles(), out, "Test"); err != nil {
		t.Fatalf("BuildEPUB error: %v", err)
	}

	opf := readEPUBFile(t, out, "EPUB/package.opf")
	spineStart := strings.Index(opf, "<spine")
	idxPos := strings.Index(opf[spineStart:], "index.xhtml")
	artPos := strings.Index(opf[spineStart:], "article-001.xhtml")
	if idxPos == -1 || artPos == -1 {
		t.Fatal("spine missing index or article entry")
	}
	if idxPos > artPos {
		t.Error("index.xhtml must appear before article-001.xhtml in spine")
	}
}

func TestBuildEPUB_EmptyArticles(t *testing.T) {
	out := tempEPUBPath(t)
	if err := BuildEPUB([]Article{}, out, "Test"); err == nil {
		t.Fatal("expected error for empty article list")
	}
}

// twoArticles returns a reusable fixture with two minimal valid articles.
func twoArticles() []Article {
	return []Article{
		{Title: "Hello World",    Content: "<p>First.</p>",  URL: "https://example.com/1"},
		{Title: "Second Article", Content: "<p>Second.</p>", URL: "https://example.com/2"},
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
