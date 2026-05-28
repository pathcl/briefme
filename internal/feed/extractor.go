package feed

import (
	"html"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var brTagRe = regexp.MustCompile(`(?i)<br\s*/?>`)

// ExtractText parses rawHTML and returns clean XHTML paragraphs containing
// only text. Block-level elements become <p> tags; inline tags are stripped.
func ExtractText(rawHTML string) string {
	if strings.TrimSpace(rawHTML) == "" {
		return ""
	}

	rawHTML = brTagRe.ReplaceAllString(rawHTML, " ")

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(rawHTML))
	if err != nil {
		return "<p>" + html.EscapeString(strings.TrimSpace(rawHTML)) + "</p>"
	}

	var paras []string
	seen := make(map[string]bool)

	doc.Find("p, li, h2, h3, h4, h5, h6, blockquote").Each(func(_ int, s *goquery.Selection) {
		text := strings.Join(strings.Fields(s.Text()), " ")
		if text == "" || seen[text] {
			return
		}
		seen[text] = true
		paras = append(paras, "<p>"+html.EscapeString(text)+"</p>")
	})

	if len(paras) == 0 {
		text := strings.Join(strings.Fields(doc.Text()), " ")
		if text != "" {
			return "<p>" + html.EscapeString(text) + "</p>"
		}
		return ""
	}

	return strings.Join(paras, "\n")
}
