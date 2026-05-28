package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func OpenStore(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS articles (
		url         TEXT PRIMARY KEY,
		title       TEXT NOT NULL,
		feed        TEXT NOT NULL DEFAULT '',
		published   DATETIME,
		recorded_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

// FilterNew returns only articles whose URL is not already in the store.
func (s *Store) FilterNew(articles []Article) ([]Article, error) {
	var out []Article
	for _, a := range articles {
		var n int
		err := s.db.QueryRow("SELECT COUNT(*) FROM articles WHERE url = ?", a.URL).Scan(&n)
		if err != nil {
			return nil, fmt.Errorf("query %q: %w", a.URL, err)
		}
		if n == 0 {
			out = append(out, a)
		}
	}
	return out, nil
}

// MarkSeen records articles in the store. Duplicates are silently ignored.
func (s *Store) MarkSeen(articles []Article) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT OR IGNORE INTO articles (url, title, feed, published)
		VALUES (?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare stmt: %w", err)
	}
	defer stmt.Close()

	for _, a := range articles {
		var pub *time.Time
		if !a.PublishedAt.IsZero() {
			pub = &a.PublishedAt
		}
		if _, err := stmt.Exec(a.URL, a.Title, a.FeedName, pub); err != nil {
			return fmt.Errorf("insert %q: %w", a.URL, err)
		}
	}

	return tx.Commit()
}
