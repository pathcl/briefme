package web_test

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pathcl/briefme/internal/config"
	"github.com/pathcl/briefme/internal/model"
	"github.com/pathcl/briefme/internal/store"
	"github.com/pathcl/briefme/internal/web"
)

func openTestStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func testConfig() *config.Config {
	return &config.Config{
		Feeds: []config.FeedConfig{
			{URL: "http://x", Name: "Feed A", Category: "news"},
			{URL: "http://y", Name: "Feed B", Category: "papers"},
		},
	}
}

func noopIngest(_ *config.Config, _ *store.Store) {}

func TestRootRedirectsToToday(t *testing.T) {
	s := openTestStore(t)
	srv := web.New(s, testConfig(), "127.0.0.1", "0", noopIngest)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	today := "/" + time.Now().Format("2006-01-02")
	if loc != today {
		t.Errorf("expected redirect to %q, got %q", today, loc)
	}
}

func TestDatePage_InvalidDate(t *testing.T) {
	s := openTestStore(t)
	srv := web.New(s, testConfig(), "127.0.0.1", "0", noopIngest)

	req := httptest.NewRequest(http.MethodGet, "/not-a-date", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestDatePage_ShowsArticles(t *testing.T) {
	s := openTestStore(t)
	articles := []model.Article{
		{Title: "Hello World", URL: "https://example.com/1", Content: "<p>Content here.</p>", Category: "news", FeedName: "Feed A"},
	}
	if err := s.MarkSeen(articles); err != nil {
		t.Fatalf("MarkSeen: %v", err)
	}

	srv := web.New(s, testConfig(), "127.0.0.1", "0", noopIngest)

	date := time.Now().Format("2006-01-02")
	req := httptest.NewRequest(http.MethodGet, "/"+date, nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Hello World") {
		t.Error("article title not found in response")
	}
	if !strings.Contains(body, "Content here.") {
		t.Error("article content not found in response")
	}
}

func TestDatePage_EmptyDate(t *testing.T) {
	s := openTestStore(t)
	srv := web.New(s, testConfig(), "127.0.0.1", "0", noopIngest)

	req := httptest.NewRequest(http.MethodGet, "/1970-01-01", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "No articles") {
		t.Error("expected empty-state message")
	}
}

func TestDatePage_NavShowsDates(t *testing.T) {
	s := openTestStore(t)
	articles := []model.Article{
		{Title: "A", URL: "https://example.com/1", Content: "<p>x</p>", Category: "news"},
	}
	if err := s.MarkSeen(articles); err != nil {
		t.Fatalf("MarkSeen: %v", err)
	}

	srv := web.New(s, testConfig(), "127.0.0.1", "0", noopIngest)

	date := time.Now().Format("2006-01-02")
	req := httptest.NewRequest(http.MethodGet, "/"+date, nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, date) {
		t.Errorf("nav should contain today's date %q", date)
	}
}
