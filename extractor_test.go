package main

import (
	"strings"
	"testing"
)

func TestExtractText_ParagraphTags(t *testing.T) {
	in := `<p>First paragraph.</p><p>Second paragraph.</p>`
	out := extractText(in)
	if !strings.Contains(out, "First paragraph.") {
		t.Error("missing first paragraph")
	}
	if !strings.Contains(out, "Second paragraph.") {
		t.Error("missing second paragraph")
	}
	if strings.Contains(out, "<a") || strings.Contains(out, "<div") {
		t.Error("output contains unexpected HTML tags")
	}
}

func TestExtractText_StripsLinksAndFormatting(t *testing.T) {
	in := `<p>Read <a href="https://example.com">this article</a> for <strong>more</strong> info.</p>`
	out := extractText(in)
	if strings.Contains(out, "<a") || strings.Contains(out, "<strong") {
		t.Error("output contains HTML tags that should have been stripped")
	}
	if !strings.Contains(out, "this article") || !strings.Contains(out, "more") {
		t.Error("link/formatted text was lost")
	}
}

func TestExtractText_BreakTagBecomesSpace(t *testing.T) {
	in := `<p>line one<br/>line two</p>`
	out := extractText(in)
	if strings.Contains(out, "oneline") {
		t.Error("<br> caused adjacent words to merge without space")
	}
}

func TestExtractText_FallbackToBodyText(t *testing.T) {
	// No block elements — plain text or only inline tags
	in := `Just a plain sentence with <em>emphasis</em>.`
	out := extractText(in)
	if !strings.Contains(out, "Just a plain sentence") {
		t.Error("plain text was lost in fallback")
	}
	if !strings.Contains(out, "emphasis") {
		t.Error("inline text was lost in fallback")
	}
}

func TestExtractText_ListItems(t *testing.T) {
	in := `<ul><li>First item</li><li>Second item</li></ul>`
	out := extractText(in)
	if !strings.Contains(out, "First item") || !strings.Contains(out, "Second item") {
		t.Error("list items were lost")
	}
}

func TestExtractText_Empty(t *testing.T) {
	out := extractText("")
	if out != "" {
		t.Errorf("expected empty output, got %q", out)
	}
}

func TestExtractText_StripsImages(t *testing.T) {
	in := `<p>Text before.</p><img src="photo.jpg" alt="a photo"/><p>Text after.</p>`
	out := extractText(in)
	if strings.Contains(out, "<img") || strings.Contains(out, "photo.jpg") {
		t.Error("image tag or src leaked into output")
	}
	if !strings.Contains(out, "Text before.") || !strings.Contains(out, "Text after.") {
		t.Error("surrounding text was lost")
	}
}
