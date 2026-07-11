package web

import (
	"regexp"
	"sort"
	"strings"
)

var (
	tagRe    = regexp.MustCompile(`<[^>]+>`)
	tokenRe  = regexp.MustCompile(`[^a-zA-Z]+`)
	stopWords = map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "from": true, "is": true, "was": true, "are": true,
		"were": true, "be": true, "been": true, "being": true, "have": true, "has": true,
		"had": true, "do": true, "does": true, "did": true, "will": true, "would": true,
		"could": true, "should": true, "may": true, "might": true, "can": true,
		"not": true, "no": true, "so": true, "that": true, "this": true, "these": true,
		"those": true, "which": true, "who": true, "when": true, "where": true,
		"what": true, "how": true, "all": true, "also": true, "just": true, "only": true,
		"then": true, "than": true, "its": true, "their": true, "they": true, "them": true,
		"we": true, "he": true, "she": true, "it": true, "you": true, "as": true,
		"if": true, "about": true, "into": true, "more": true, "some": true, "any": true,
		"one": true, "two": true, "our": true, "your": true, "his": true, "her": true,
		"while": true, "after": true, "before": true, "over": true, "such": true,
		"very": true, "up": true, "out": true, "there": true, "here": true, "both": true,
		"each": true, "new": true, "other": true, "first": true, "last": true,
		"many": true, "most": true, "own": true, "same": true, "through": true,
	}
)

// SuggestTags extracts the top n keywords from raw HTML content.
func SuggestTags(html string, n int) []string {
	text := tagRe.ReplaceAllString(html, " ")
	tokens := tokenRe.Split(strings.ToLower(text), -1)

	freq := make(map[string]int)
	for _, w := range tokens {
		if len(w) < 4 || stopWords[w] {
			continue
		}
		freq[w]++
	}

	type kv struct {
		word  string
		count int
	}
	var ranked []kv
	for w, c := range freq {
		if c > 1 {
			ranked = append(ranked, kv{w, c})
		}
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].count != ranked[j].count {
			return ranked[i].count > ranked[j].count
		}
		return ranked[i].word < ranked[j].word
	})

	out := make([]string, 0, n)
	for i := 0; i < len(ranked) && i < n; i++ {
		out = append(out, ranked[i].word)
	}
	return out
}
