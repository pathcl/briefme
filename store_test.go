package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := OpenStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// --- article deduplication ---

func TestStore_FilterNew_AllNew(t *testing.T) {
	s := openTestStore(t)
	articles := []Article{
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

func TestStore_FilterNew_AllSeen(t *testing.T) {
	s := openTestStore(t)
	articles := []Article{{Title: "A", URL: "https://example.com/1"}}
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

func TestStore_FilterNew_Mixed(t *testing.T) {
	s := openTestStore(t)
	if err := s.MarkSeen([]Article{{Title: "Old", URL: "https://example.com/old"}}); err != nil {
		t.Fatalf("MarkSeen: %v", err)
	}
	all := []Article{
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

func TestStore_MarkSeen_Idempotent(t *testing.T) {
	s := openTestStore(t)
	articles := []Article{{Title: "A", URL: "https://example.com/1"}}
	if err := s.MarkSeen(articles); err != nil {
		t.Fatalf("first MarkSeen: %v", err)
	}
	if err := s.MarkSeen(articles); err != nil {
		t.Fatalf("second MarkSeen should not error on duplicate: %v", err)
	}
}

func TestStore_RecordsMetadata(t *testing.T) {
	s := openTestStore(t)
	pub := time.Date(2026, 5, 28, 10, 0, 0, 0, time.UTC)
	articles := []Article{
		{Title: "Test Article", URL: "https://example.com/1", FeedName: "Test Feed", PublishedAt: pub},
	}
	if err := s.MarkSeen(articles); err != nil {
		t.Fatalf("MarkSeen: %v", err)
	}
	got, err := s.FilterNew(articles)
	if err != nil {
		t.Fatalf("FilterNew: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected article to be marked as seen")
	}
}

// --- EPUB checksum deduplication ---

func TestStore_LookupEPUB_NotFound(t *testing.T) {
	s := openTestStore(t)
	filename, found, err := s.LookupEPUB("abc123")
	if err != nil {
		t.Fatalf("LookupEPUB: %v", err)
	}
	if found {
		t.Error("unknown checksum should not be found")
	}
	if filename != "" {
		t.Errorf("expected empty filename, got %q", filename)
	}
}

func TestStore_LookupEPUB_ReturnsFilename(t *testing.T) {
	s := openTestStore(t)
	if err := s.RecordEPUB("abc123", "briefme-2026-05-28.epub"); err != nil {
		t.Fatalf("RecordEPUB: %v", err)
	}
	filename, found, err := s.LookupEPUB("abc123")
	if err != nil {
		t.Fatalf("LookupEPUB: %v", err)
	}
	if !found {
		t.Error("checksum should be found after RecordEPUB")
	}
	if filename != "briefme-2026-05-28.epub" {
		t.Errorf("expected filename briefme-2026-05-28.epub, got %q", filename)
	}
}

func TestStore_RecordEPUB_Idempotent(t *testing.T) {
	s := openTestStore(t)
	if err := s.RecordEPUB("abc123", "briefme.epub"); err != nil {
		t.Fatalf("first RecordEPUB: %v", err)
	}
	if err := s.RecordEPUB("abc123", "briefme.epub"); err != nil {
		t.Fatalf("second RecordEPUB should not error: %v", err)
	}
}

func TestStore_LookupEPUB_FileDeletedShouldReDeliver(t *testing.T) {
	// Simulates the scenario: DB has a record but the file was deleted.
	// LookupEPUB itself just returns the filename; the caller (main) is
	// responsible for the os.Stat check. This test verifies LookupEPUB
	// faithfully returns whatever filename was recorded, even if the file
	// no longer exists — so the caller can make the right decision.
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
	// The file does not exist — caller must check os.Stat independently.
	if _, statErr := os.Stat(filename); statErr == nil {
		t.Skip("file unexpectedly exists; skipping deletion scenario")
	}
	// Confirm the filename round-trips correctly so the caller can Stat it.
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

	sum1, err := checksumFile(f.Name())
	if err != nil {
		t.Fatalf("checksumFile: %v", err)
	}
	sum2, err := checksumFile(f.Name())
	if err != nil {
		t.Fatalf("checksumFile second call: %v", err)
	}
	if sum1 != sum2 {
		t.Error("checksum is not deterministic")
	}
	if len(sum1) != 64 {
		t.Errorf("expected 64-char hex SHA-256, got %d chars", len(sum1))
	}
}

func TestChecksumFile_DifferentContents(t *testing.T) {
	write := func(content string) string {
		f, err := os.CreateTemp("", "briefme-chk-*")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(f.Name())
		f.WriteString(content)
		f.Close()
		sum, err := checksumFile(f.Name())
		if err != nil {
			t.Fatal(err)
		}
		return sum
	}
	if write("aaa") == write("bbb") {
		t.Error("different files produced the same checksum")
	}
}
