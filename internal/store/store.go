package store

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"log"
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

	s := &Store{db: db}
	if n, err := s.PurgeInvalidArticles(); err != nil {
		db.Close()
		return nil, err
	} else if n > 0 {
		log.Printf("store: purged %d invalid article(s) (binary/PDF content)", n)
	}
	return s, nil
}

// TagCount is returned by GetAllTags.
type TagCount struct {
	Tag   string
	Count int
}

func migrate(db *sql.DB) error {
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
		);
		CREATE TABLE IF NOT EXISTS tags (
			url        TEXT NOT NULL,
			tag        TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (url, tag)
		)`)
	if err != nil {
		return fmt.Errorf("create schema: %w", err)
	}

	for _, stmt := range []string{
		"ALTER TABLE articles ADD COLUMN category TEXT NOT NULL DEFAULT 'news'",
		"ALTER TABLE articles ADD COLUMN content  TEXT NOT NULL DEFAULT ''",
	} {
		db.Exec(stmt) // intentionally ignore error (column already exists)
	}

	return nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

// PurgeInvalidArticles removes articles whose stored content is binary (e.g. a
// PDF fetched by mistake). Detection: content starts with the PDF magic bytes
// "%PDF", or the URL ends with ".pdf" (case-insensitive).
// Returns the number of rows deleted.
func (s *Store) PurgeInvalidArticles() (int, error) {
	res, err := s.db.Exec(`
		DELETE FROM articles
		WHERE content LIKE '%PDF-%'
		   OR LOWER(url) LIKE '%.pdf'
		   OR LOWER(url) LIKE '%.pdf?%'`)
	if err != nil {
		return 0, fmt.Errorf("purge invalid articles: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
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

// GetDates returns all distinct dates (YYYY-MM-DD) that have articles, newest first.
func (s *Store) GetDates() ([]string, error) {
	rows, err := s.db.Query(`
		SELECT DISTINCT DATE(recorded_at, 'localtime')
		FROM articles
		ORDER BY recorded_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("query dates: %w", err)
	}
	defer rows.Close()

	var dates []string
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			return nil, fmt.Errorf("scan date: %w", err)
		}
		dates = append(dates, d)
	}
	return dates, rows.Err()
}

// GetDatesInMonth returns the set of dates (YYYY-MM-DD) in the given month
// (formatted as YYYY-MM) that have at least one article.
func (s *Store) GetDatesInMonth(yearMonth string) (map[string]bool, error) {
	rows, err := s.db.Query(`
		SELECT DISTINCT DATE(recorded_at, 'localtime')
		FROM articles
		WHERE strftime('%Y-%m', recorded_at, 'localtime') = ?`,
		yearMonth)
	if err != nil {
		return nil, fmt.Errorf("query dates in month: %w", err)
	}
	defer rows.Close()

	out := make(map[string]bool)
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			return nil, fmt.Errorf("scan date: %w", err)
		}
		out[d] = true
	}
	return out, rows.Err()
}

// AddTag attaches a tag to an article URL. Duplicate tags are silently ignored.
func (s *Store) AddTag(url, tag string) error {
	_, err := s.db.Exec(
		"INSERT OR IGNORE INTO tags (url, tag) VALUES (?, ?)", url, tag)
	if err != nil {
		return fmt.Errorf("add tag: %w", err)
	}
	return nil
}

// RemoveTag detaches a tag from an article URL.
func (s *Store) RemoveTag(url, tag string) error {
	_, err := s.db.Exec("DELETE FROM tags WHERE url = ? AND tag = ?", url, tag)
	if err != nil {
		return fmt.Errorf("remove tag: %w", err)
	}
	return nil
}

// GetTagsForArticle returns all tags for a given article URL, sorted alphabetically.
func (s *Store) GetTagsForArticle(url string) ([]string, error) {
	rows, err := s.db.Query(
		"SELECT tag FROM tags WHERE url = ? ORDER BY tag ASC", url)
	if err != nil {
		return nil, fmt.Errorf("get tags: %w", err)
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

// GetArticlesByTag returns all articles that carry the given tag, newest first.
func (s *Store) GetArticlesByTag(tag string) ([]model.Article, error) {
	rows, err := s.db.Query(`
		SELECT a.url, a.title, a.feed, a.category, a.content, a.published
		FROM articles a
		JOIN tags t ON a.url = t.url
		WHERE t.tag = ?
		ORDER BY a.recorded_at DESC`, tag)
	if err != nil {
		return nil, fmt.Errorf("get articles by tag: %w", err)
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

// GetAllTags returns all tags with their article counts, sorted by count descending.
func (s *Store) GetAllTags() ([]TagCount, error) {
	rows, err := s.db.Query(`
		SELECT tag, COUNT(*) as n
		FROM tags
		GROUP BY tag
		ORDER BY n DESC, tag ASC`)
	if err != nil {
		return nil, fmt.Errorf("get all tags: %w", err)
	}
	defer rows.Close()

	var out []TagCount
	for rows.Next() {
		var tc TagCount
		if err := rows.Scan(&tc.Tag, &tc.Count); err != nil {
			return nil, fmt.Errorf("scan tag count: %w", err)
		}
		out = append(out, tc)
	}
	return out, rows.Err()
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
