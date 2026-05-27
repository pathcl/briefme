package main

import (
	"fmt"
	"time"

	"github.com/mmcdole/gofeed"
)

type Article struct {
	Title       string
	Author      string
	URL         string
	Content     string
	PublishedAt time.Time
	FeedName    string
}

func FetchArticles(feeds []FeedConfig, max int) ([]Article, error) {
	parser := gofeed.NewParser()
	seen := make(map[string]struct{})
	var articles []Article

	for _, fc := range feeds {
		feed, err := parser.ParseURL(fc.URL)
		if err != nil {
			return nil, fmt.Errorf("fetch feed %q: %w", fc.URL, err)
		}

		for _, item := range feed.Items {
			if len(articles) >= max {
				break
			}
			url := item.Link
			if _, dup := seen[url]; dup {
				continue
			}
			seen[url] = struct{}{}

			content := item.Content
			if content == "" {
				content = item.Description
			}

			author := ""
			if item.Author != nil {
				author = item.Author.Name
			}

			var published time.Time
			if item.PublishedParsed != nil {
				published = *item.PublishedParsed
			}

			articles = append(articles, Article{
				Title:       item.Title,
				Author:      author,
				URL:         url,
				Content:     content,
				PublishedAt: published,
				FeedName:    fc.Name,
			})
		}
	}

	return articles, nil
}
