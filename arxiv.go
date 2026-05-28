package main

import "regexp"

var arxivAbstractRe = regexp.MustCompile(`^https?://arxiv\.org/abs/([\w.]+(?:v\d+)?)`)

// arxivAbstractToHTML converts an arXiv abstract URL to its HTML full-paper URL.
// Returns empty string if the URL is not a recognised arXiv abstract URL,
// or if it already points to the HTML version.
func arxivAbstractToHTML(u string) string {
	m := arxivAbstractRe.FindStringSubmatch(u)
	if m == nil {
		return ""
	}
	return "https://arxiv.org/html/" + m[1]
}
