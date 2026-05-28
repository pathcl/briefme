package feed

import (
	"fmt"
	"time"

	"github.com/mmcdole/gofeed"
	"github.com/pathcl/briefme/internal/config"
	"github.com/pathcl/briefme/internal/model"
)

func FetchArticles(feeds []config.FeedConfig, max int) ([]model.Article, error) {
	parser := gofeed.NewParser()
	seen := make(map[string]struct{})
	var articles []model.Article

	for _, fc := range feeds {
		feed, err := parser.ParseURL(fc.URL)
		if err != nil {
			return nil, fmt.Errorf("fetch feed %q: %w", fc.URL, err)
		}

		feedCount := 0
		for _, item := range feed.Items {
			if feedCount >= max {
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

			feedCount++
			articles = append(articles, model.Article{
				Title:       item.Title,
				Author:      author,
				URL:         url,
				Content:     content,
				PublishedAt: published,
				FeedName:    fc.Name,
				Category:    fc.Category,
			})
		}
	}

	return articles, nil
}
