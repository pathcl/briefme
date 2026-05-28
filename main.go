package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	dryRun := flag.Bool("dry-run", false, "build EPUBs locally but skip copying to Kobo")
	flag.Parse()

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	store, err := OpenStore(cfg.DBPath)
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer store.Close()

	log.Printf("fetching from %d feed(s), max %d articles each", len(cfg.Feeds), cfg.MaxPerFeed)
	articles, err := FetchArticles(cfg.Feeds, cfg.MaxPerFeed)
	if err != nil {
		log.Fatalf("fetch: %v", err)
	}

	articles, err = store.FilterNew(articles)
	if err != nil {
		log.Fatalf("filter: %v", err)
	}
	log.Printf("%d new articles (unseen)", len(articles))

	if len(articles) == 0 {
		log.Println("nothing new to deliver")
		os.Exit(0)
	}

	// Group by category so each gets its own EPUB.
	groups := make(map[string][]Article)
	for _, a := range articles {
		groups[a.Category] = append(groups[a.Category], a)
	}

	// Sort categories for deterministic ordering.
	categories := make([]string, 0, len(groups))
	for c := range groups {
		categories = append(categories, c)
	}
	sort.Strings(categories)

	date := time.Now().Format("2006-01-02")
	for _, category := range categories {
		group := groups[category]
		log.Printf("[%s] fetching full article content (%d articles)...", category, len(group))
		group = EnrichArticles(group)
		if len(group) == 0 {
			log.Printf("[%s] no articles with extractable content — skipping", category)
			continue
		}

		title := fmt.Sprintf("Briefme %s – %s", capitalize(category), date)
		epubPath := fmt.Sprintf("briefme-%s-%s.epub", category, date)

		if err := BuildEPUB(group, epubPath, title); err != nil {
			log.Printf("[%s] build epub failed: %v — skipping", category, err)
			continue
		}
		log.Printf("[%s] built %s", category, epubPath)

		sum, err := checksumFile(epubPath)
		if err != nil {
			log.Printf("[%s] checksum failed: %v — skipping", category, err)
			continue
		}
		log.Printf("[%s] SHA-256: %s", category, sum)

		if prevFile, found, err := store.LookupEPUB(sum); err != nil {
			log.Printf("[%s] epub lookup failed: %v — skipping", category, err)
			continue
		} else if found {
			if _, statErr := os.Stat(prevFile); statErr == nil {
				log.Printf("[%s] identical EPUB already exists at %s — skipping", category, prevFile)
				continue
			}
			log.Printf("[%s] checksum matches previous build (%s) but file no longer exists — re-delivering", category, prevFile)
		}

		if err := store.MarkSeen(group); err != nil {
			log.Printf("[%s] mark seen failed: %v — skipping", category, err)
			continue
		}
		if err := store.RecordEPUB(sum, epubPath); err != nil {
			log.Printf("[%s] record epub failed: %v — skipping", category, err)
			continue
		}

		if *dryRun {
			log.Printf("[%s] --dry-run: skipping copy to Kobo", category)
			continue
		}

		if err := DeliverEPUB(cfg.KoboPath, epubPath); err != nil {
			log.Printf("[%s] deliver failed: %v", category, err)
			continue
		}
		log.Printf("[%s] copied to Kobo", category)
	}

	if !*dryRun {
		log.Println("all done — safely eject and enjoy reading")
	}
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
