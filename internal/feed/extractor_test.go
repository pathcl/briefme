package feed_test

import (
	"strings"
	"testing"

	"github.com/pathcl/briefme/internal/feed"
)

func TestExtractText_ParagraphTags(t *testing.T) {
	out := feed.ExtractText(`<p>First paragraph.</p><p>Second paragraph.</p>`)
	if !strings.Contains(out, "First paragraph.") || !strings.Contains(out, "Second paragraph.") {
		t.Error("paragraph text missing")
	}
	if strings.Contains(out, "<a") || strings.Contains(out, "<div") {
		t.Error("unexpected HTML tags in output")
	}
}

func TestExtractText_StripsLinksAndFormatting(t *testing.T) {
	out := feed.ExtractText(`<p>Read <a href="https://example.com">this article</a> for <strong>more</strong> info.</p>`)
	if strings.Contains(out, "<a") || strings.Contains(out, "<strong") {
		t.Error("HTML tags not stripped")
	}
	if !strings.Contains(out, "this article") || !strings.Contains(out, "more") {
		t.Error("text content lost")
	}
}

func TestExtractText_BreakTagBecomesSpace(t *testing.T) {
	out := feed.ExtractText(`<p>line one<br/>line two</p>`)
	if strings.Contains(out, "oneline") {
		t.Error("<br> caused words to merge without space")
	}
}

func TestExtractText_FallbackToBodyText(t *testing.T) {
	out := feed.ExtractText(`Just a plain sentence with <em>emphasis</em>.`)
	if !strings.Contains(out, "Just a plain sentence") || !strings.Contains(out, "emphasis") {
		t.Error("plain text lost in fallback")
	}
}

func TestExtractText_ListItems(t *testing.T) {
	out := feed.ExtractText(`<ul><li>First item</li><li>Second item</li></ul>`)
	if !strings.Contains(out, "First item") || !strings.Contains(out, "Second item") {
		t.Error("list items lost")
	}
}

func TestExtractText_Empty(t *testing.T) {
	if out := feed.ExtractText(""); out != "" {
		t.Errorf("expected empty output, got %q", out)
	}
}

func TestExtractText_StripsImages(t *testing.T) {
	out := feed.ExtractText(`<p>Text before.</p><img src="photo.jpg" alt="a photo"/><p>Text after.</p>`)
	if strings.Contains(out, "<img") || strings.Contains(out, "photo.jpg") {
		t.Error("image tag or src leaked into output")
	}
	if !strings.Contains(out, "Text before.") || !strings.Contains(out, "Text after.") {
		t.Error("surrounding text lost")
	}
}
