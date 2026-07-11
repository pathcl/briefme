package store_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pathcl/briefme/internal/model"
	"github.com/pathcl/briefme/internal/store"
)

func openTestStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// --- article deduplication ---

func TestFilterNew_AllNew(t *testing.T) {
	s := openTestStore(t)
	articles := []model.Article{
		{Title: "A", URL: "https://example.com/1"},
		{Title: "B", URL: "https://example.com/2"},
	}
	got, err := s.FilterNew(articles)
	if err != nil {
		t.Fatalf("FilterNew: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 new articles, got %d", len(got))
	}
}

func TestFilterNew_AllSeen(t *testing.T) {
	s := openTestStore(t)
	articles := []model.Article{{Title: "A", URL: "https://example.com/1"}}
	if err := s.MarkSeen(articles); err != nil {
		t.Fatalf("MarkSeen: %v", err)
	}
	got, err := s.FilterNew(articles)
	if err != nil {
		t.Fatalf("FilterNew: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 new articles, got %d", len(got))
	}
}

func TestFilterNew_Mixed(t *testing.T) {
	s := openTestStore(t)
	if err := s.MarkSeen([]model.Article{{Title: "Old", URL: "https://example.com/old"}}); err != nil {
		t.Fatalf("MarkSeen: %v", err)
	}
	all := []model.Article{
		{Title: "Old",  URL: "https://example.com/old"},
		{Title: "New1", URL: "https://example.com/new1"},
		{Title: "New2", URL: "https://example.com/new2"},
	}
	got, err := s.FilterNew(all)
	if err != nil {
		t.Fatalf("FilterNew: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 new articles, got %d", len(got))
	}
	for _, a := range got {
		if a.URL == "https://example.com/old" {
			t.Error("seen article should have been filtered out")
		}
	}
}

func TestMarkSeen_Idempotent(t *testing.T) {
	s := openTestStore(t)
	articles := []model.Article{{Title: "A", URL: "https://example.com/1"}}
	if err := s.MarkSeen(articles); err != nil {
		t.Fatalf("first MarkSeen: %v", err)
	}
	if err := s.MarkSeen(articles); err != nil {
		t.Fatalf("second MarkSeen should not error: %v", err)
	}
}

// --- daily accumulation ---

func TestMarkSeen_StoresContent(t *testing.T) {
	s := openTestStore(t)
	articles := []model.Article{{
		Title:    "Article with content",
		URL:      "https://example.com/1",
		Content:  "<p>The full article text.</p>",
		Category: "news",
		FeedName: "Test Feed",
	}}
	if err := s.MarkSeen(articles); err != nil {
		t.Fatalf("MarkSeen: %v", err)
	}

	date := time.Now().Format("2006-01-02")
	got, err := s.GetArticlesByDate("news", date)
	if err != nil {
		t.Fatalf("GetArticlesByDate: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 article, got %d", len(got))
	}
	if got[0].Content != "<p>The full article text.</p>" {
		t.Errorf("content not stored/retrieved: %q", got[0].Content)
	}
	if got[0].Title != "Article with content" {
		t.Errorf("title not retrieved: %q", got[0].Title)
	}
}

func TestGetArticlesByDate_AccumulatesAcrossRuns(t *testing.T) {
	s := openTestStore(t)
	date := time.Now().Format("2006-01-02")

	// Simulate first run — 2 news articles
	run1 := []model.Article{
		{Title: "Morning A", URL: "https://example.com/1", Content: "<p>A</p>", Category: "news"},
		{Title: "Morning B", URL: "https://example.com/2", Content: "<p>B</p>", Category: "news"},
	}
	if err := s.MarkSeen(run1); err != nil {
		t.Fatalf("MarkSeen run1: %v", err)
	}

	// Simulate second run — 1 more news article
	run2 := []model.Article{
		{Title: "Noon C", URL: "https://example.com/3", Content: "<p>C</p>", Category: "news"},
	}
	if err := s.MarkSeen(run2); err != nil {
		t.Fatalf("MarkSeen run2: %v", err)
	}

	got, err := s.GetArticlesByDate("news", date)
	if err != nil {
		t.Fatalf("GetArticlesByDate: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("expected 3 accumulated articles, got %d", len(got))
	}
}

func TestGetArticlesByDate_SeparatesCategories(t *testing.T) {
	s := openTestStore(t)
	date := time.Now().Format("2006-01-02")

	articles := []model.Article{
		{Title: "News 1",  URL: "https://example.com/n1", Content: "<p>n</p>", Category: "news"},
		{Title: "Paper 1", URL: "https://example.com/p1", Content: "<p>p</p>", Category: "papers"},
		{Title: "News 2",  URL: "https://example.com/n2", Content: "<p>n</p>", Category: "news"},
	}
	if err := s.MarkSeen(articles); err != nil {
		t.Fatalf("MarkSeen: %v", err)
	}

	news, err := s.GetArticlesByDate("news", date)
	if err != nil {
		t.Fatalf("GetArticlesByDate news: %v", err)
	}
	if len(news) != 2 {
		t.Errorf("expected 2 news articles, got %d", len(news))
	}

	papers, err := s.GetArticlesByDate("papers", date)
	if err != nil {
		t.Fatalf("GetArticlesByDate papers: %v", err)
	}
	if len(papers) != 1 {
		t.Errorf("expected 1 paper, got %d", len(papers))
	}
}

func TestGetArticlesByDate_EmptyForWrongDate(t *testing.T) {
	s := openTestStore(t)
	articles := []model.Article{
		{Title: "Today", URL: "https://example.com/1", Content: "<p>x</p>", Category: "news"},
	}
	if err := s.MarkSeen(articles); err != nil {
		t.Fatalf("MarkSeen: %v", err)
	}

	got, err := s.GetArticlesByDate("news", "1970-01-01")
	if err != nil {
		t.Fatalf("GetArticlesByDate: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 articles for wrong date, got %d", len(got))
	}
}

// --- calendar: dates in month ---

func TestGetDatesInMonth_ReturnsMatchingDates(t *testing.T) {
	s := openTestStore(t)
	articles := []model.Article{
		{Title: "A", URL: "https://example.com/1", Category: "news"},
		{Title: "B", URL: "https://example.com/2", Category: "papers"},
	}
	if err := s.MarkSeen(articles); err != nil {
		t.Fatalf("MarkSeen: %v", err)
	}
	month := time.Now().Format("2006-01")
	got, err := s.GetDatesInMonth(month)
	if err != nil {
		t.Fatalf("GetDatesInMonth: %v", err)
	}
	today := time.Now().Format("2006-01-02")
	if !got[today] {
		t.Errorf("expected today %q to be in result", today)
	}
}

func TestGetDatesInMonth_EmptyForOtherMonth(t *testing.T) {
	s := openTestStore(t)
	articles := []model.Article{
		{Title: "A", URL: "https://example.com/1", Category: "news"},
	}
	if err := s.MarkSeen(articles); err != nil {
		t.Fatalf("MarkSeen: %v", err)
	}
	got, err := s.GetDatesInMonth("1970-01")
	if err != nil {
		t.Fatalf("GetDatesInMonth: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty map for 1970-01, got %v", got)
	}
}

// --- tagging ---

func TestAddTag_AndRetrieve(t *testing.T) {
	s := openTestStore(t)
	arts := []model.Article{{Title: "A", URL: "https://example.com/1", Category: "news"}}
	if err := s.MarkSeen(arts); err != nil {
		t.Fatalf("MarkSeen: %v", err)
	}
	if err := s.AddTag("https://example.com/1", "golang"); err != nil {
		t.Fatalf("AddTag: %v", err)
	}
	tags, err := s.GetTagsForArticle("https://example.com/1")
	if err != nil {
		t.Fatalf("GetTagsForArticle: %v", err)
	}
	if len(tags) != 1 || tags[0] != "golang" {
		t.Errorf("expected [golang], got %v", tags)
	}
}

func TestAddTag_Idempotent(t *testing.T) {
	s := openTestStore(t)
	arts := []model.Article{{Title: "A", URL: "https://example.com/1", Category: "news"}}
	s.MarkSeen(arts)
	s.AddTag("https://example.com/1", "golang")
	if err := s.AddTag("https://example.com/1", "golang"); err != nil {
		t.Fatalf("duplicate AddTag should not error: %v", err)
	}
	tags, _ := s.GetTagsForArticle("https://example.com/1")
	if len(tags) != 1 {
		t.Errorf("expected 1 tag, got %d", len(tags))
	}
}

func TestRemoveTag(t *testing.T) {
	s := openTestStore(t)
	arts := []model.Article{{Title: "A", URL: "https://example.com/1", Category: "news"}}
	s.MarkSeen(arts)
	s.AddTag("https://example.com/1", "golang")
	s.AddTag("https://example.com/1", "ai")
	if err := s.RemoveTag("https://example.com/1", "golang"); err != nil {
		t.Fatalf("RemoveTag: %v", err)
	}
	tags, _ := s.GetTagsForArticle("https://example.com/1")
	if len(tags) != 1 || tags[0] != "ai" {
		t.Errorf("expected [ai] after removal, got %v", tags)
	}
}

func TestGetArticlesByTag(t *testing.T) {
	s := openTestStore(t)
	arts := []model.Article{
		{Title: "Go post",  URL: "https://example.com/1", Category: "news", Content: "<p>x</p>"},
		{Title: "AI paper", URL: "https://example.com/2", Category: "papers", Content: "<p>y</p>"},
		{Title: "Other",    URL: "https://example.com/3", Category: "news", Content: "<p>z</p>"},
	}
	s.MarkSeen(arts)
	s.AddTag("https://example.com/1", "golang")
	s.AddTag("https://example.com/2", "golang")

	got, err := s.GetArticlesByTag("golang")
	if err != nil {
		t.Fatalf("GetArticlesByTag: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 articles, got %d", len(got))
	}
}

func TestGetAllTags(t *testing.T) {
	s := openTestStore(t)
	arts := []model.Article{
		{Title: "A", URL: "https://example.com/1", Category: "news"},
		{Title: "B", URL: "https://example.com/2", Category: "news"},
	}
	s.MarkSeen(arts)
	s.AddTag("https://example.com/1", "golang")
	s.AddTag("https://example.com/1", "ai")
	s.AddTag("https://example.com/2", "golang")

	tags, err := s.GetAllTags()
	if err != nil {
		t.Fatalf("GetAllTags: %v", err)
	}
	if len(tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(tags))
	}
	if tags[0].Tag != "golang" || tags[0].Count != 2 {
		t.Errorf("expected golang×2 first, got %+v", tags[0])
	}
}

// --- date listing ---

func TestGetDates_ReturnsDistinctDatesNewestFirst(t *testing.T) {
	s := openTestStore(t)

	articles := []model.Article{
		{Title: "A", URL: "https://example.com/1", Category: "news"},
		{Title: "B", URL: "https://example.com/2", Category: "papers"},
	}
	if err := s.MarkSeen(articles); err != nil {
		t.Fatalf("MarkSeen: %v", err)
	}

	dates, err := s.GetDates()
	if err != nil {
		t.Fatalf("GetDates: %v", err)
	}
	if len(dates) != 1 {
		t.Fatalf("expected 1 date, got %d: %v", len(dates), dates)
	}
	today := time.Now().Format("2006-01-02")
	if dates[0] != today {
		t.Errorf("expected %q, got %q", today, dates[0])
	}
}

func TestGetDates_EmptyWhenNoArticles(t *testing.T) {
	s := openTestStore(t)
	dates, err := s.GetDates()
	if err != nil {
		t.Fatalf("GetDates: %v", err)
	}
	if len(dates) != 0 {
		t.Errorf("expected 0 dates, got %d", len(dates))
	}
}

// --- EPUB checksum ---

func TestLookupEPUB_NotFound(t *testing.T) {
	s := openTestStore(t)
	_, found, err := s.LookupEPUB("abc123")
	if err != nil {
		t.Fatalf("LookupEPUB: %v", err)
	}
	if found {
		t.Error("unknown checksum should not be found")
	}
}

func TestLookupEPUB_ReturnsFilename(t *testing.T) {
	s := openTestStore(t)
	if err := s.RecordEPUB("abc123", "briefme-2026-05-28.epub"); err != nil {
		t.Fatalf("RecordEPUB: %v", err)
	}
	filename, found, err := s.LookupEPUB("abc123")
	if err != nil {
		t.Fatalf("LookupEPUB: %v", err)
	}
	if !found || filename != "briefme-2026-05-28.epub" {
		t.Errorf("expected found with filename, got found=%v filename=%q", found, filename)
	}
}

func TestRecordEPUB_Idempotent(t *testing.T) {
	s := openTestStore(t)
	if err := s.RecordEPUB("abc123", "briefme.epub"); err != nil {
		t.Fatalf("first RecordEPUB: %v", err)
	}
	if err := s.RecordEPUB("abc123", "briefme.epub"); err != nil {
		t.Fatalf("second RecordEPUB should not error: %v", err)
	}
}

func TestLookupEPUB_FileDeletedShouldReDeliver(t *testing.T) {
	s := openTestStore(t)
	deleted := "/tmp/briefme-deleted.epub"
	if err := s.RecordEPUB("deadbeef", deleted); err != nil {
		t.Fatalf("RecordEPUB: %v", err)
	}
	filename, found, err := s.LookupEPUB("deadbeef")
	if err != nil {
		t.Fatalf("LookupEPUB: %v", err)
	}
	if !found {
		t.Fatal("expected record to be found")
	}
	if _, statErr := os.Stat(filename); statErr == nil {
		t.Skip("file unexpectedly exists")
	}
	if filename != deleted {
		t.Errorf("expected %q, got %q", deleted, filename)
	}
}

func TestChecksumFile_Deterministic(t *testing.T) {
	f, err := os.CreateTemp("", "briefme-chk-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.WriteString("hello briefme")
	f.Close()

	sum1, err := store.ChecksumFile(f.Name())
	if err != nil {
		t.Fatalf("ChecksumFile: %v", err)
	}
	sum2, _ := store.ChecksumFile(f.Name())
	if sum1 != sum2 || len(sum1) != 64 {
		t.Errorf("checksum not deterministic or wrong length: %q", sum1)
	}
}

// --- purge invalid articles ---

func TestPurgeInvalidArticles_RemovesPDFContent(t *testing.T) {
	s := openTestStore(t)
	arts := []model.Article{
		{Title: "Good", URL: "https://example.com/article", Category: "news", Content: "<p>hello</p>"},
		{Title: "PDF binary", URL: "https://example.com/paper.pdf", Category: "news", Content: "%PDF-1.7 binary garbage"},
		{Title: "PDF url", URL: "https://cdn.example.com/doc.pdf", Category: "papers", Content: "%PDF-1.4 more garbage"},
	}
	if err := s.MarkSeen(arts); err != nil {
		t.Fatalf("MarkSeen: %v", err)
	}
	n, err := s.PurgeInvalidArticles()
	if err != nil {
		t.Fatalf("PurgeInvalidArticles: %v", err)
	}
	if n != 2 {
		t.Errorf("expected 2 purged, got %d", n)
	}
	remaining, _ := s.FilterNew([]model.Article{{URL: "https://example.com/article"}})
	if len(remaining) != 0 {
		t.Error("good article should still be in DB")
	}
}

func TestPurgeInvalidArticles_IdempotentOnCleanDB(t *testing.T) {
	s := openTestStore(t)
	arts := []model.Article{
		{Title: "Good", URL: "https://example.com/1", Category: "news", Content: "<p>fine</p>"},
	}
	s.MarkSeen(arts)
	n, err := s.PurgeInvalidArticles()
	if err != nil {
		t.Fatalf("PurgeInvalidArticles: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 purged on clean DB, got %d", n)
	}
}

func TestChecksumFile_DifferentContents(t *testing.T) {
	write := func(content string) string {
		f, _ := os.CreateTemp("", "briefme-chk-*")
		defer os.Remove(f.Name())
		f.WriteString(content)
		f.Close()
		sum, _ := store.ChecksumFile(f.Name())
		return sum
	}
	if write("aaa") == write("bbb") {
		t.Error("different files produced same checksum")
	}
}
