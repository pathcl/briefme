package main

import (
	"fmt"
	"html"
	"os"
	"time"

	epub "github.com/bmaupin/go-epub"
)

const articleCSS = `body { font-family: Georgia, serif; font-size: 1em; line-height: 1.6; margin: 1em; }
h1 { font-size: 1.4em; margin-bottom: 0.2em; }
.meta { font-size: 0.85em; color: #555; margin-bottom: 1em; }
`

func BuildEPUB(articles []Article, outputPath string) error {
	if len(articles) == 0 {
		return fmt.Errorf("no articles to package")
	}

	cssFile, err := os.CreateTemp("", "briefme-*.css")
	if err != nil {
		return fmt.Errorf("create temp css: %w", err)
	}
	defer os.Remove(cssFile.Name())
	if _, err := cssFile.WriteString(articleCSS); err != nil {
		cssFile.Close()
		return fmt.Errorf("write temp css: %w", err)
	}
	cssFile.Close()

	date := time.Now().Format("2006-01-02")
	title := fmt.Sprintf("Briefme – %s", date)

	book := epub.NewEpub(title)
	book.SetAuthor("briefme")

	cssPath, err := book.AddCSS(cssFile.Name(), "article.css")
	if err != nil {
		return fmt.Errorf("add css: %w", err)
	}

	for _, a := range articles {
		body := buildArticleHTML(a, cssPath)
		sectionTitle := html.EscapeString(a.Title)
		if _, err := book.AddSection(body, sectionTitle, "", cssPath); err != nil {
			return fmt.Errorf("add section %q: %w", a.Title, err)
		}
	}

	if err := book.Write(outputPath); err != nil {
		return fmt.Errorf("write epub: %w", err)
	}
	return nil
}

func buildArticleHTML(a Article, _ string) string {
	meta := ""
	if !a.PublishedAt.IsZero() {
		meta += html.EscapeString(a.PublishedAt.Format("January 2, 2006"))
	}
	if a.Author != "" {
		if meta != "" {
			meta += " · "
		}
		meta += html.EscapeString(a.Author)
	}
	if a.FeedName != "" {
		if meta != "" {
			meta += " · "
		}
		meta += html.EscapeString(a.FeedName)
	}

	metaHTML := ""
	if meta != "" {
		metaHTML = fmt.Sprintf(`<p class="meta">%s</p>`, meta)
	}

	return fmt.Sprintf(`<h1>%s</h1>%s<div>%s</div>`,
		html.EscapeString(a.Title), metaHTML, a.Content)
}
