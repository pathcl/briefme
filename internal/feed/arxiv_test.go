package feed_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pathcl/briefme/internal/feed"
)

func TestArxivAbstractToHTML(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"https://arxiv.org/abs/2605.27567", "https://arxiv.org/html/2605.27567"},
		{"https://arxiv.org/abs/2605.27567v1", "https://arxiv.org/html/2605.27567v1"},
		{"http://arxiv.org/abs/2605.27567v2", "https://arxiv.org/html/2605.27567v2"},
		{"https://arxiv.org/html/2605.27567v1", ""},
		{"https://www.theguardian.com/tech/article", ""},
		{"", ""},
	}
	for _, c := range cases {
		got := feed.ArxivAbstractToHTML(c.in)
		if got != c.want {
			t.Errorf("ArxivAbstractToHTML(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFetchContent_PrefersArxivHTML(t *testing.T) {
	htmlSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(realArticleHTML))
	}))
	defer htmlSrv.Close()

	content, err := feed.FetchContentWithAlt("https://arxiv.org/abs/2605.27567", htmlSrv.URL)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !strings.Contains(content, "bioluminescent") {
		t.Error("HTML version content not returned")
	}
}

func TestFetchContent_FallsBackToAbstractWhenHTMLMissing(t *testing.T) {
	htmlSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer htmlSrv.Close()

	abstractSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(realArticleHTML))
	}))
	defer abstractSrv.Close()

	content, err := feed.FetchContentWithAlt(abstractSrv.URL, htmlSrv.URL)
	if err != nil {
		t.Fatalf("error: %v", err)
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

	content, err := feed.FetchContentWithAlt(srv.URL, "")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !strings.Contains(content, "bioluminescent") {
		t.Error("direct fetch content not returned")
	}
}
