package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	dryRun := flag.Bool("dry-run", false, "build EPUB locally but skip copying to Kobo")
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

	log.Println("fetching full article content...")
	articles = EnrichArticles(articles)

	if len(articles) == 0 {
		log.Println("no articles with extractable content")
		os.Exit(0)
	}

	epubPath := fmt.Sprintf("briefme-%s.epub", time.Now().Format("2006-01-02"))
	if err := BuildEPUB(articles, epubPath); err != nil {
		log.Fatalf("build epub: %v", err)
	}
	log.Printf("built EPUB: %s", epubPath)

	sum, err := checksumFile(epubPath)
	if err != nil {
		log.Fatalf("checksum: %v", err)
	}
	log.Printf("EPUB SHA-256: %s", sum)

	if seen, err := store.EPUBSeen(sum); err != nil {
		log.Fatalf("check epub: %v", err)
	} else if seen {
		log.Println("identical EPUB already produced and delivered — nothing to do")
		return
	}

	if err := store.MarkSeen(articles); err != nil {
		log.Fatalf("mark seen: %v", err)
	}
	if err := store.RecordEPUB(sum, epubPath); err != nil {
		log.Fatalf("record epub: %v", err)
	}

	if *dryRun {
		log.Println("--dry-run: skipping copy to Kobo")
		return
	}

	if err := DeliverEPUB(cfg.KoboPath, epubPath); err != nil {
		log.Fatalf("deliver: %v", err)
	}
	log.Println("copied to Kobo — safely eject and enjoy reading")
}
