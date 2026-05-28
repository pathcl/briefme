package main

import (
	"fmt"
	"html"
	"os"
	"strings"
	"time"

	epub "github.com/bmaupin/go-epub"
)

const articleCSS = `body { font-family: Georgia, serif; font-size: 1em; line-height: 1.6; margin: 1em; }
h1 { font-size: 1.4em; margin-bottom: 0.2em; }
h2 { font-size: 1.1em; margin: 1.5em 0 0.3em; }
.meta { font-size: 0.85em; color: #555; margin-bottom: 1em; }
.back { font-size: 0.85em; margin-bottom: 1.5em; }
.toc { list-style: decimal; padding-left: 1.5em; }
.toc li { margin: 0.5em 0; }
.toc .feed { font-size: 0.8em; color: #777; margin-left: 0.4em; }
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

	// Pre-assign filenames so the index can link to them.
	filenames := make([]string, len(articles))
	for i := range articles {
		filenames[i] = fmt.Sprintf("article-%03d.xhtml", i+1)
	}

	indexHTML := buildIndexHTML(title, articles, filenames)
	if _, err := book.AddSection(indexHTML, "Contents", "index.xhtml", cssPath); err != nil {
		return fmt.Errorf("add index: %w", err)
	}

	for i, a := range articles {
		body := buildArticleHTML(a)
		if _, err := book.AddSection(body, html.EscapeString(a.Title), filenames[i], cssPath); err != nil {
			return fmt.Errorf("add section %q: %w", a.Title, err)
		}
	}

	if err := book.Write(outputPath); err != nil {
		return fmt.Errorf("write epub: %w", err)
	}
	return nil
}

func buildIndexHTML(title string, articles []Article, filenames []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "<h1>%s</h1>\n<ol class=\"toc\">\n", html.EscapeString(title))
	for i, a := range articles {
		feed := ""
		if a.FeedName != "" {
			feed = fmt.Sprintf(` <span class="feed">%s</span>`, html.EscapeString(a.FeedName))
		}
		fmt.Fprintf(&b, "  <li><a href=\"%s\">%s</a>%s</li>\n",
			filenames[i], html.EscapeString(a.Title), feed)
	}
	b.WriteString("</ol>")
	return b.String()
}

func buildArticleHTML(a Article) string {
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

	var b strings.Builder
	fmt.Fprintf(&b, `<p class="back"><a href="index.xhtml">← Contents</a></p>`)
	fmt.Fprintf(&b, "<h1>%s</h1>\n", html.EscapeString(a.Title))
	if meta != "" {
		fmt.Fprintf(&b, "<p class=\"meta\">%s</p>\n", meta)
	}
	fmt.Fprintf(&b, "<div>%s</div>", extractText(a.Content))
	return b.String()
}
