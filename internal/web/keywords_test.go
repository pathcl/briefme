package web_test

import (
	"testing"

	"github.com/pathcl/briefme/internal/web"
)

func TestSuggestTags_ExtractsFrequentWords(t *testing.T) {
	html := `<p>Go is a programming language. Go makes systems programming easy.
	Systems written in Go are fast. Fast systems need good programming.</p>`

	tags := web.SuggestTags(html, 5)
	if len(tags) == 0 {
		t.Fatal("expected at least one suggestion")
	}

	found := map[string]bool{}
	for _, tag := range tags {
		found[tag] = true
	}
	// "programming" appears 3 times, "systems" 3 times, "fast" 2 times
	if !found["programming"] && !found["systems"] {
		t.Errorf("expected high-frequency words in suggestions, got %v", tags)
	}
}

func TestSuggestTags_FiltersStopWords(t *testing.T) {
	html := `<p>the and or but is was are were have has had</p>`
	tags := web.SuggestTags(html, 5)
	for _, tag := range tags {
		if stopWordCheck(tag) {
			t.Errorf("stop word %q should not appear in suggestions", tag)
		}
	}
}

func TestSuggestTags_FiltersShortWords(t *testing.T) {
	html := `<p>AI ML go run big large systems systems systems</p>`
	tags := web.SuggestTags(html, 5)
	for _, tag := range tags {
		if len(tag) < 4 {
			t.Errorf("short word %q (len %d) should not appear in suggestions", tag, len(tag))
		}
	}
}

func TestSuggestTags_RespectsN(t *testing.T) {
	html := `<p>alpha alpha beta beta gamma gamma delta delta epsilon epsilon zeta zeta</p>`
	tags := web.SuggestTags(html, 3)
	if len(tags) > 3 {
		t.Errorf("expected at most 3 suggestions, got %d: %v", len(tags), tags)
	}
}

func stopWordCheck(w string) bool {
	stops := []string{"the", "and", "or", "but", "is", "was", "are", "were", "have", "has", "had"}
	for _, s := range stops {
		if w == s {
			return true
		}
	}
	return false
}
