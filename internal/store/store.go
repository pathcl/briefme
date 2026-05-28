package store

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/pathcl/briefme/internal/model"
	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}

	return &Store{db: db}, nil
}

func migrate(db *sql.DB) error {
	// Create tables if they don't exist.
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS articles (
			url         TEXT PRIMARY KEY,
			title       TEXT NOT NULL,
			feed        TEXT NOT NULL DEFAULT '',
			category    TEXT NOT NULL DEFAULT 'news',
			content     TEXT NOT NULL DEFAULT '',
			published   DATETIME,
			recorded_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS epubs (
			sha256      TEXT PRIMARY KEY,
			filename    TEXT NOT NULL,
			produced_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return fmt.Errorf("create schema: %w", err)
	}

	// Add columns that may be missing in existing databases.
	// SQLite returns an error if the column already exists; we ignore it.
	for _, col := range []string{
		"ALTER TABLE articles ADD COLUMN category TEXT NOT NULL DEFAULT 'news'",
		"ALTER TABLE articles ADD COLUMN content  TEXT NOT NULL DEFAULT ''",
	} {
		db.Exec(col) // intentionally ignore error
	}

	return nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

// FilterNew returns only articles whose URL is not already in the store.
func (s *Store) FilterNew(articles []model.Article) ([]model.Article, error) {
	var out []model.Article
	for _, a := range articles {
		var n int
		if err := s.db.QueryRow("SELECT COUNT(*) FROM articles WHERE url = ?", a.URL).Scan(&n); err != nil {
			return nil, fmt.Errorf("query %q: %w", a.URL, err)
		}
		if n == 0 {
			out = append(out, a)
		}
	}
	return out, nil
}

// MarkSeen records articles including their content and category.
// Duplicates are silently ignored.
func (s *Store) MarkSeen(articles []model.Article) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT OR IGNORE INTO articles (url, title, feed, category, content, published)
		VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare stmt: %w", err)
	}
	defer stmt.Close()

	for _, a := range articles {
		var pub *time.Time
		if !a.PublishedAt.IsZero() {
			pub = &a.PublishedAt
		}
		if _, err := stmt.Exec(a.URL, a.Title, a.FeedName, a.Category, a.Content, pub); err != nil {
			return fmt.Errorf("insert %q: %w", a.URL, err)
		}
	}
	return tx.Commit()
}

// GetArticlesByDate returns all articles for the given category recorded on date (YYYY-MM-DD),
// ordered by recorded_at ascending so the EPUB reflects arrival order.
func (s *Store) GetArticlesByDate(category, date string) ([]model.Article, error) {
	rows, err := s.db.Query(`
		SELECT url, title, feed, category, content, published
		FROM articles
		WHERE category = ?
		  AND DATE(recorded_at, 'localtime') = ?
		ORDER BY recorded_at ASC`,
		category, date)
	if err != nil {
		return nil, fmt.Errorf("query articles by date: %w", err)
	}
	defer rows.Close()

	var articles []model.Article
	for rows.Next() {
		var a model.Article
		var pub sql.NullString
		if err := rows.Scan(&a.URL, &a.Title, &a.FeedName, &a.Category, &a.Content, &pub); err != nil {
			return nil, fmt.Errorf("scan article: %w", err)
		}
		if pub.Valid && pub.String != "" {
			if t, err := time.Parse(time.RFC3339, pub.String); err == nil {
				a.PublishedAt = t
			}
		}
		articles = append(articles, a)
	}
	return articles, rows.Err()
}

// LookupEPUB checks whether an EPUB with this SHA-256 was previously produced.
func (s *Store) LookupEPUB(sha256sum string) (filename string, found bool, err error) {
	qErr := s.db.QueryRow("SELECT filename FROM epubs WHERE sha256 = ?", sha256sum).Scan(&filename)
	if qErr == sql.ErrNoRows {
		return "", false, nil
	}
	if qErr != nil {
		return "", false, fmt.Errorf("lookup epub: %w", qErr)
	}
	return filename, true, nil
}

func (s *Store) RecordEPUB(sha256sum, filename string) error {
	_, err := s.db.Exec("INSERT OR IGNORE INTO epubs (sha256, filename) VALUES (?, ?)", sha256sum, filename)
	if err != nil {
		return fmt.Errorf("record epub: %w", err)
	}
	return nil
}

// ChecksumFile returns the hex-encoded SHA-256 of the file at path.
func ChecksumFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open for checksum: %w", err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("hash file: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
