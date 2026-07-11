package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/pathcl/briefme/internal/config"
	"github.com/pathcl/briefme/internal/deliver"
	"github.com/pathcl/briefme/internal/epub"
	"github.com/pathcl/briefme/internal/feed"
	"github.com/pathcl/briefme/internal/store"
	"github.com/pathcl/briefme/internal/web"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "serve" {
		runServe(os.Args[2:])
		return
	}
	runFetch(os.Args[1:])
}

// ingest fetches new articles from all feeds and stores them.
// It does not build EPUBs or deliver to Kobo.
func ingest(cfg *config.Config, db *store.Store) {
	log.Printf("fetching from %d feed(s), max %d per feed", len(cfg.Feeds), cfg.MaxPerFeed)
	candidates, err := feed.FetchArticles(cfg.Feeds, cfg.MaxPerFeed)
	if err != nil {
		log.Printf("fetch: %v", err)
		return
	}

	newArticles, err := db.FilterNew(candidates)
	if err != nil {
		log.Printf("filter: %v", err)
		return
	}
	log.Printf("%d new articles to fetch", len(newArticles))

	if len(newArticles) == 0 {
		return
	}
	newArticles = feed.EnrichArticles(newArticles)
	if len(newArticles) > 0 {
		if err := db.MarkSeen(newArticles); err != nil {
			log.Printf("store articles: %v", err)
		}
	}
}

func runFetch(args []string) {
	fs := flag.NewFlagSet("briefme", flag.ExitOnError)
	configPath := fs.String("config", "config.yaml", "path to config file")
	dryRun := fs.Bool("dry-run", false, "build EPUBs locally but skip copying to Kobo")
	fs.Parse(args)

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := store.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer db.Close()

	// Step 1: ingest new articles
	ingest(cfg, db)

	// Step 2: build one EPUB per category from all of today's articles
	date := time.Now().Format("2006-01-02")
	for _, category := range uniqueCategories(cfg.Feeds) {
		todayArticles, err := db.GetArticlesByDate(category, date)
		if err != nil {
			log.Printf("[%s] query failed: %v — skipping", category, err)
			continue
		}
		if len(todayArticles) == 0 {
			log.Printf("[%s] no articles for today", category)
			continue
		}

		log.Printf("[%s] building EPUB with %d article(s) for %s", category, len(todayArticles), date)
		title := fmt.Sprintf("Briefme %s – %s", capitalize(category), date)
		epubPath := fmt.Sprintf("briefme-%s-%s.epub", category, date)

		if err := epub.Build(todayArticles, epubPath, title); err != nil {
			log.Printf("[%s] build failed: %v — skipping", category, err)
			continue
		}

		sum, err := store.ChecksumFile(epubPath)
		if err != nil {
			log.Printf("[%s] checksum failed: %v — skipping", category, err)
			continue
		}

		if prevFile, found, err := db.LookupEPUB(sum); err != nil {
			log.Printf("[%s] lookup failed: %v", category, err)
		} else if found {
			if _, statErr := os.Stat(prevFile); statErr == nil {
				log.Printf("[%s] no new articles since last delivery — nothing to do", category)
				continue
			}
			log.Printf("[%s] re-delivering (previous file %s no longer exists)", category, prevFile)
		}

		if err := db.RecordEPUB(sum, epubPath); err != nil {
			log.Printf("[%s] record epub failed: %v", category, err)
		}

		if *dryRun {
			log.Printf("[%s] --dry-run: %s ready (%d articles)", category, epubPath, len(todayArticles))
			continue
		}

		if err := deliver.ToKobo(cfg.KoboPath, epubPath); err != nil {
			log.Printf("[%s] deliver failed: %v", category, err)
			continue
		}
		log.Printf("[%s] delivered %s (%d articles)", category, epubPath, len(todayArticles))
	}

	if !*dryRun {
		log.Println("done")
	}
}

func runServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	configPath := fs.String("config", "config.yaml", "path to config file")
	port := fs.String("port", "8080", "HTTP listen port")
	bind := fs.String("bind", "127.0.0.1", "HTTP bind address")
	fs.Parse(args)

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := store.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer db.Close()

	srv := web.New(db, cfg, *bind, *port, ingest)
	if err := srv.Start(); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func uniqueCategories(feeds []config.FeedConfig) []string {
	seen := make(map[string]bool)
	var out []string
	for _, f := range feeds {
		if !seen[f.Category] {
			seen[f.Category] = true
			out = append(out, f.Category)
		}
	}
	sort.Strings(out)
	return out
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
