package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	readability "github.com/go-shiori/go-readability"
)

var httpClient = &http.Client{Timeout: 20 * time.Second}

// FetchContent fetches the article at articleURL and returns the main body HTML.
// For arXiv abstract URLs it tries the experimental HTML full-paper version first,
// falling back to the abstract page if the HTML version is unavailable.
func FetchContent(articleURL string) (string, error) {
	return fetchContent(articleURL, arxivAbstractToHTML(articleURL))
}

// fetchContent fetches altURL first (if non-empty); on failure falls back to articleURL.
// Keeping altURL as an explicit parameter makes the arXiv fallback logic testable
// without hitting real external servers.
func fetchContent(articleURL, altURL string) (string, error) {
	if altURL != "" {
		if content, err := fetchReadable(altURL); err == nil {
			return content, nil
		}
		// HTML version unavailable — fall through to the original URL.
	}
	return fetchReadable(articleURL)
}

// fetchReadable fetches u and extracts the main article body via go-readability.
func fetchReadable(u string) (string, error) {
	if u == "" {
		return "", fmt.Errorf("empty URL")
	}

	parsedURL, err := url.Parse(u)
	if err != nil {
		return "", fmt.Errorf("parse url %q: %w", u, err)
	}

	resp, err := httpClient.Get(u)
	if err != nil {
		return "", fmt.Errorf("fetch %q: %w", u, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch %q: HTTP %d", u, resp.StatusCode)
	}

	article, err := readability.FromReader(resp.Body, parsedURL)
	if err != nil {
		return "", fmt.Errorf("parse %q: %w", u, err)
	}
	if article.Content == "" {
		return "", fmt.Errorf("no content extracted from %q", u)
	}

	return article.Content, nil
}

// EnrichArticles fetches the full text for each article URL. Articles that
// cannot be fetched or parsed are dropped entirely.
func EnrichArticles(articles []Article) []Article {
	var out []Article
	for _, a := range articles {
		content, err := FetchContent(a.URL)
		if err != nil {
			log.Printf("skip %q: %v", a.Title, err)
			continue
		}
		a.Content = content
		out = append(out, a)
	}
	log.Printf("%d/%d articles fetched successfully", len(out), len(articles))
	return out
}
