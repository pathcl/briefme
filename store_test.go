package main

import (
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
	articles := []Article{
		{Title: "A", URL: "https://example.com/1"},
	}
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
	seen := []Article{{Title: "Old", URL: "https://example.com/old"}}
	if err := s.MarkSeen(seen); err != nil {
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

	// Article should now be filtered out
	got, err := s.FilterNew(articles)
	if err != nil {
		t.Fatalf("FilterNew: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected article to be marked as seen")
	}
}
