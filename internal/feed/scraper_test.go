package feed_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pathcl/briefme/internal/feed"
	"github.com/pathcl/briefme/internal/model"
)

// realArticleHTML is shared across scraper and arxiv tests.
const realArticleHTML = `<!DOCTYPE html>
<html>
<head><title>Scientists Discover New Species</title></head>
<body>
  <header><nav>Home | About | Contact</nav></header>
  <main>
    <article>
      <h1>Scientists Discover New Species in Amazon Rainforest</h1>
      <p class="byline">By Jane Smith · May 28, 2026</p>
      <p>Researchers from the University of São Paulo announced today the discovery
      of a previously unknown species of tree frog deep in the Amazon basin. The
      amphibian, measuring just two centimetres in length, was found during a
      routine survey near the Tapajós River.</p>
      <p>The species, provisionally named Dendropsophus tapajosensis, displays an
      unusual pattern of bioluminescent markings along its dorsal surface — a trait
      previously undocumented in this genus. Lead researcher Dr Ana Carvalho said
      the find underscores how much biodiversity remains undescribed even in
      heavily studied regions.</p>
      <p>The team spent three field seasons collecting specimens before publishing
      their findings in the journal Zootaxa. Genetic analysis confirmed the animal
      is distinct from all 115 known members of the Dendropsophus genus. Further
      surveys are planned to estimate population size and assess conservation status.</p>
    </article>
  </main>
  <aside>Advertisement: Buy our stuff!</aside>
  <footer>© 2026 Example News</footer>
</body>
</html>`

func articleServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(realArticleHTML))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestFetchContent_ExtractsArticleText(t *testing.T) {
	srv := articleServer(t)
	content, err := feed.FetchContent(srv.URL)
	if err != nil {
		t.Fatalf("FetchContent error: %v", err)
	}
	if !strings.Contains(content, "bioluminescent") {
		t.Error("article body text not found in extracted content")
	}
}

func TestFetchContent_ExcludesSidebarAndFooter(t *testing.T) {
	srv := articleServer(t)
	content, err := feed.FetchContent(srv.URL)
	if err != nil {
		t.Fatalf("FetchContent error: %v", err)
	}
	if strings.Contains(content, "Buy our stuff") {
		t.Error("sidebar/ad content leaked into extracted text")
	}
}

func TestFetchContent_BadURL(t *testing.T) {
	_, err := feed.FetchContent("http://127.0.0.1:0/nonexistent")
	if err == nil {
		t.Fatal("expected error for unreachable URL")
	}
}

func TestFetchContent_Non200Status(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()
	if _, err := feed.FetchContent(srv.URL); err == nil {
		t.Fatal("expected error for 403 response")
	}
}

func TestEnrichArticles_DropsFailedArticles(t *testing.T) {
	articles := []model.Article{{Title: "Bad", URL: "", Content: "rss garbage"}}
	result := feed.EnrichArticles(articles)
	if len(result) != 0 {
		t.Errorf("expected failed article to be dropped, got %d articles", len(result))
	}
}

func TestEnrichArticles_KeepsSuccessfulArticles(t *testing.T) {
	srv := articleServer(t)
	articles := []model.Article{{Title: "Good", URL: srv.URL, Content: "rss excerpt"}}
	result := feed.EnrichArticles(articles)
	if len(result) != 1 {
		t.Fatalf("expected 1 article, got %d", len(result))
	}
	if result[0].Content == "rss excerpt" {
		t.Error("content was not replaced with scraped version")
	}
}

func TestEnrichArticles_MixedResults(t *testing.T) {
	srv := articleServer(t)
	articles := []model.Article{
		{Title: "Good",      URL: srv.URL, Content: "rss excerpt"},
		{Title: "Paywalled", URL: "",       Content: "rss garbage"},
		{Title: "Good 2",    URL: srv.URL, Content: "rss excerpt 2"},
	}
	result := feed.EnrichArticles(articles)
	if len(result) != 2 {
		t.Errorf("expected 2 successful articles, got %d", len(result))
	}
	for _, a := range result {
		if a.Title == "Paywalled" {
			t.Error("failed article should not appear in result")
		}
	}
}
