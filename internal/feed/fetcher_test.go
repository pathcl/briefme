package feed_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pathcl/briefme/internal/config"
	"github.com/pathcl/briefme/internal/feed"
)

const sampleRSS = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
    <link>https://example.com</link>
    <item>
      <title>Article One</title>
      <link>https://example.com/article-1</link>
      <author>Alice</author>
      <description>Short description one</description>
      <content:encoded xmlns:content="http://purl.org/rss/1.0/modules/content/"><![CDATA[<p>Full content one</p>]]></content:encoded>
      <pubDate>Mon, 26 May 2025 10:00:00 +0000</pubDate>
    </item>
    <item>
      <title>Article Two</title>
      <link>https://example.com/article-2</link>
      <description>Short description two</description>
      <pubDate>Sun, 25 May 2025 09:00:00 +0000</pubDate>
    </item>
  </channel>
</rss>`

func rssServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.Write([]byte(sampleRSS))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestFetchArticles_Basic(t *testing.T) {
	srv := rssServer(t)
	articles, err := feed.FetchArticles([]config.FeedConfig{{URL: srv.URL, Name: "Test"}}, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(articles) != 2 {
		t.Fatalf("expected 2 articles, got %d", len(articles))
	}
	a := articles[0]
	if a.Title != "Article One" {
		t.Errorf("unexpected title: %s", a.Title)
	}
	if a.Content != "<p>Full content one</p>" {
		t.Errorf("expected full content, got: %s", a.Content)
	}
	if a.URL != "https://example.com/article-1" {
		t.Errorf("unexpected URL: %s", a.URL)
	}
}

func TestFetchArticles_FallsBackToDescription(t *testing.T) {
	srv := rssServer(t)
	articles, err := feed.FetchArticles([]config.FeedConfig{{URL: srv.URL, Name: "Test"}}, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if articles[1].Content != "Short description two" {
		t.Errorf("expected description fallback, got: %s", articles[1].Content)
	}
}

func TestFetchArticles_PerFeedLimit(t *testing.T) {
	srv := rssServer(t)
	feeds := []config.FeedConfig{
		{URL: srv.URL + "?a", Name: "Feed A"},
		{URL: srv.URL + "?b", Name: "Feed B"},
	}
	articles, err := feed.FetchArticles(feeds, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(articles) != 2 {
		t.Errorf("expected 1 per feed (2 total), got %d", len(articles))
	}
}

func TestFetchArticles_DeduplicatesByURL(t *testing.T) {
	srv := rssServer(t)
	feeds := []config.FeedConfig{
		{URL: srv.URL, Name: "Feed A"},
		{URL: srv.URL, Name: "Feed B"},
	}
	articles, err := feed.FetchArticles(feeds, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(articles) != 2 {
		t.Errorf("expected 2 deduplicated articles, got %d", len(articles))
	}
}

func TestFetchArticles_CategoryPassedToArticles(t *testing.T) {
	srv := rssServer(t)
	articles, err := feed.FetchArticles([]config.FeedConfig{{URL: srv.URL, Name: "arXiv", Category: "papers"}}, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, a := range articles {
		if a.Category != "papers" {
			t.Errorf("expected category 'papers', got %q", a.Category)
		}
	}
}

func TestFetchArticles_BadURL(t *testing.T) {
	_, err := feed.FetchArticles([]config.FeedConfig{{URL: "http://127.0.0.1:0/bad", Name: "Bad"}}, 10)
	if err == nil {
		t.Fatal("expected error for bad URL")
	}
}
