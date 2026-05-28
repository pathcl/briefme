package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestArxivAbstractToHTML(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{
			"https://arxiv.org/abs/2605.27567",
			"https://arxiv.org/html/2605.27567",
		},
		{
			"https://arxiv.org/abs/2605.27567v1",
			"https://arxiv.org/html/2605.27567v1",
		},
		{
			"http://arxiv.org/abs/2605.27567v2",
			"https://arxiv.org/html/2605.27567v2",
		},
		{
			// already an HTML URL — must not double-convert
			"https://arxiv.org/html/2605.27567v1",
			"",
		},
		{
			// non-arXiv URL
			"https://www.theguardian.com/tech/article",
			"",
		},
		{
			// empty
			"",
			"",
		},
	}
	for _, c := range cases {
		got := arxivAbstractToHTML(c.in)
		if got != c.want {
			t.Errorf("arxivAbstractToHTML(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFetchContent_PrefersArxivHTML(t *testing.T) {
	htmlSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(realArticleHTML)) // reuse fixture from scraper_test.go
	}))
	defer htmlSrv.Close()

	// Provide the HTML version URL directly; simulate what arxivAbstractToHTML returns.
	content, err := fetchContent("https://arxiv.org/abs/2605.27567", htmlSrv.URL)
	if err != nil {
		t.Fatalf("fetchContent error: %v", err)
	}
	if !strings.Contains(content, "bioluminescent") {
		t.Error("HTML version content not returned")
	}
}

func TestFetchContent_FallsBackToAbstractWhenHTMLMissing(t *testing.T) {
	// HTML version returns 404.
	htmlSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer htmlSrv.Close()

	// Abstract page returns real content.
	abstractSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(realArticleHTML))
	}))
	defer abstractSrv.Close()

	content, err := fetchContent(abstractSrv.URL, htmlSrv.URL)
	if err != nil {
		t.Fatalf("fetchContent error: %v", err)
	}
	if !strings.Contains(content, "bioluminescent") {
		t.Error("abstract fallback content not returned")
	}
}

func TestFetchContent_NoAltURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(realArticleHTML))
	}))
	defer srv.Close()

	// altURL = "" means no arXiv variant — should fetch original directly.
	content, err := fetchContent(srv.URL, "")
	if err != nil {
		t.Fatalf("fetchContent error: %v", err)
	}
	if !strings.Contains(content, "bioluminescent") {
		t.Error("direct fetch content not returned")
	}
}
