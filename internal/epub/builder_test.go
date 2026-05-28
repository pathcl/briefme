package epub_test

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pathcl/briefme/internal/epub"
	"github.com/pathcl/briefme/internal/model"
)

func TestBuild_CreatesFile(t *testing.T) {
	out := tempEPUBPath(t)
	if err := epub.Build(twoArticles(), out, "Briefme News – 2026-05-28"); err != nil {
		t.Fatalf("Build error: %v", err)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("EPUB file not created: %v", err)
	}
}

func TestBuild_TitleAppearsInOutput(t *testing.T) {
	out := tempEPUBPath(t)
	title := "Briefme Papers – 2026-05-28"
	if err := epub.Build(twoArticles(), out, title); err != nil {
		t.Fatalf("Build error: %v", err)
	}
	if !strings.Contains(readEPUBFile(t, out, "EPUB/package.opf"), title) {
		t.Errorf("OPF does not contain title %q", title)
	}
	if !strings.Contains(readEPUBFile(t, out, "EPUB/xhtml/index.xhtml"), title) {
		t.Errorf("index page does not contain title %q", title)
	}
}

func TestBuild_ValidZipStructure(t *testing.T) {
	out := tempEPUBPath(t)
	if err := epub.Build(twoArticles(), out, "Test"); err != nil {
		t.Fatalf("Build error: %v", err)
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

func TestBuild_HasIndexPage(t *testing.T) {
	articles := []model.Article{
		{Title: "Article One", Content: "<p>one</p>", URL: "https://example.com/1"},
		{Title: "Article Two", Content: "<p>two</p>", URL: "https://example.com/2"},
	}
	out := tempEPUBPath(t)
	if err := epub.Build(articles, out, "Test"); err != nil {
		t.Fatalf("Build error: %v", err)
	}
	index := readEPUBFile(t, out, "EPUB/xhtml/index.xhtml")
	if !strings.Contains(index, "Article One") || !strings.Contains(index, "Article Two") {
		t.Error("index missing article titles")
	}
}

func TestBuild_IndexLinksToArticles(t *testing.T) {
	articles := []model.Article{
		{Title: "First",  Content: "<p>one</p>", URL: "https://example.com/1"},
		{Title: "Second", Content: "<p>two</p>", URL: "https://example.com/2"},
	}
	out := tempEPUBPath(t)
	if err := epub.Build(articles, out, "Test"); err != nil {
		t.Fatalf("Build error: %v", err)
	}
	index := readEPUBFile(t, out, "EPUB/xhtml/index.xhtml")
	for i := range articles {
		if !strings.Contains(index, fmt.Sprintf(`href="article-%03d.xhtml"`, i+1)) {
			t.Errorf("index missing href link to article-%03d.xhtml", i+1)
		}
	}
}

func TestBuild_ArticlesHaveBackLink(t *testing.T) {
	out := tempEPUBPath(t)
	if err := epub.Build(twoArticles(), out, "Test"); err != nil {
		t.Fatalf("Build error: %v", err)
	}
	if !strings.Contains(readEPUBFile(t, out, "EPUB/xhtml/article-001.xhtml"), `href="index.xhtml"`) {
		t.Error("article missing back link to index.xhtml")
	}
}

func TestBuild_ArticleContentPresent(t *testing.T) {
	articles := []model.Article{
		{Title: "My Article", Content: "<p>The body of the article lives here.</p>", URL: "https://example.com/1"},
	}
	out := tempEPUBPath(t)
	if err := epub.Build(articles, out, "Test"); err != nil {
		t.Fatalf("Build error: %v", err)
	}
	if !strings.Contains(readEPUBFile(t, out, "EPUB/xhtml/article-001.xhtml"), "The body of the article lives here.") {
		t.Error("article body text missing")
	}
}

func TestBuild_HTMLStrippedFromContent(t *testing.T) {
	articles := []model.Article{
		{Title: "Tagged", Content: `<p>Clean text. <a href="https://evil.com"><img src="x"/></a></p>`, URL: "https://example.com/1"},
	}
	out := tempEPUBPath(t)
	if err := epub.Build(articles, out, "Test"); err != nil {
		t.Fatalf("Build error: %v", err)
	}
	body := readEPUBFile(t, out, "EPUB/xhtml/article-001.xhtml")
	if strings.Contains(body, "evil.com") || strings.Contains(body, "<img") {
		t.Error("raw HTML leaked into article content")
	}
	if !strings.Contains(body, "Clean text.") {
		t.Error("readable text was lost")
	}
}

func TestBuild_ArticleMetadata(t *testing.T) {
	articles := []model.Article{{
		Title:       "Metadata Test",
		Author:      "Jane Doe",
		FeedName:    "Test Feed",
		PublishedAt: time.Date(2026, 5, 28, 0, 0, 0, 0, time.UTC),
		Content:     "<p>body</p>",
		URL:         "https://example.com/1",
	}}
	out := tempEPUBPath(t)
	if err := epub.Build(articles, out, "Test"); err != nil {
		t.Fatalf("Build error: %v", err)
	}
	body := readEPUBFile(t, out, "EPUB/xhtml/article-001.xhtml")
	for _, want := range []string{"Metadata Test", "Jane Doe", "Test Feed", "May 28, 2026"} {
		if !strings.Contains(body, want) {
			t.Errorf("article missing metadata field %q", want)
		}
	}
}

func TestBuild_SpecialCharsEscaped(t *testing.T) {
	articles := []model.Article{{
		Title:   `<script>alert("xss")</script>`,
		Content: "<p>body</p>",
		URL:     "https://example.com/1",
	}}
	out := tempEPUBPath(t)
	if err := epub.Build(articles, out, "Test"); err != nil {
		t.Fatalf("Build error: %v", err)
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

func TestBuild_OPFManifestComplete(t *testing.T) {
	articles := []model.Article{
		{Title: "One",   Content: "<p>a</p>", URL: "https://example.com/1"},
		{Title: "Two",   Content: "<p>b</p>", URL: "https://example.com/2"},
		{Title: "Three", Content: "<p>c</p>", URL: "https://example.com/3"},
	}
	out := tempEPUBPath(t)
	if err := epub.Build(articles, out, "Test"); err != nil {
		t.Fatalf("Build error: %v", err)
	}
	opf := readEPUBFile(t, out, "EPUB/package.opf")
	for i := range articles {
		if !strings.Contains(opf, fmt.Sprintf("article-%03d.xhtml", i+1)) {
			t.Errorf("OPF missing article-%03d.xhtml", i+1)
		}
	}
	if !strings.Contains(opf, "index.xhtml") {
		t.Error("OPF missing index.xhtml")
	}
}

func TestBuild_SpineOrderIndexFirst(t *testing.T) {
	out := tempEPUBPath(t)
	if err := epub.Build(twoArticles(), out, "Test"); err != nil {
		t.Fatalf("Build error: %v", err)
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

func TestBuild_EmptyArticles(t *testing.T) {
	if err := epub.Build([]model.Article{}, tempEPUBPath(t), "Test"); err == nil {
		t.Fatal("expected error for empty article list")
	}
}

func twoArticles() []model.Article {
	return []model.Article{
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
