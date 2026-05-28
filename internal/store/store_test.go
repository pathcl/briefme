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

func TestMarkSeen_RecordsMetadata(t *testing.T) {
	s := openTestStore(t)
	pub := time.Date(2026, 5, 28, 10, 0, 0, 0, time.UTC)
	articles := []model.Article{
		{Title: "Test", URL: "https://example.com/1", FeedName: "Test Feed", PublishedAt: pub},
	}
	if err := s.MarkSeen(articles); err != nil {
		t.Fatalf("MarkSeen: %v", err)
	}
	got, err := s.FilterNew(articles)
	if err != nil {
		t.Fatalf("FilterNew: %v", err)
	}
	if len(got) != 0 {
		t.Error("expected article to be marked as seen")
	}
}

func TestLookupEPUB_NotFound(t *testing.T) {
	s := openTestStore(t)
	filename, found, err := s.LookupEPUB("abc123")
	if err != nil {
		t.Fatalf("LookupEPUB: %v", err)
	}
	if found || filename != "" {
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
	sum2, err := store.ChecksumFile(f.Name())
	if err != nil {
		t.Fatalf("ChecksumFile second call: %v", err)
	}
	if sum1 != sum2 || len(sum1) != 64 {
		t.Errorf("checksum not deterministic or wrong length: %q", sum1)
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
