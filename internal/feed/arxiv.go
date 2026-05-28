package feed

import "regexp"

var arxivAbstractRe = regexp.MustCompile(`^https?://arxiv\.org/abs/([\w.]+(?:v\d+)?)`)

// ArxivAbstractToHTML converts an arXiv abstract URL to its HTML full-paper URL.
// Returns empty string if the URL is not a recognised arXiv abstract URL.
func ArxivAbstractToHTML(u string) string {
	m := arxivAbstractRe.FindStringSubmatch(u)
	if m == nil {
		return ""
	}
	return "https://arxiv.org/html/" + m[1]
}
