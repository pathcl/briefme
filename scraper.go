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

// FetchContent fetches the URL, runs it through the readability parser,
// and returns the main article HTML stripped of boilerplate.
func FetchContent(articleURL string) (string, error) {
	if articleURL == "" {
		return "", fmt.Errorf("empty URL")
	}

	parsedURL, err := url.Parse(articleURL)
	if err != nil {
		return "", fmt.Errorf("parse url %q: %w", articleURL, err)
	}

	resp, err := httpClient.Get(articleURL)
	if err != nil {
		return "", fmt.Errorf("fetch %q: %w", articleURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch %q: HTTP %d", articleURL, resp.StatusCode)
	}

	article, err := readability.FromReader(resp.Body, parsedURL)
	if err != nil {
		return "", fmt.Errorf("parse %q: %w", articleURL, err)
	}
	if article.Content == "" {
		return "", fmt.Errorf("no content extracted from %q", articleURL)
	}

	return article.Content, nil
}

// EnrichArticles fetches the full text for each article URL. Articles that
// cannot be fetched or parsed are dropped entirely — partial RSS metadata
// is not a useful substitute for real content.
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
