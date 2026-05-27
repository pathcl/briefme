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
	dryRun := flag.Bool("dry-run", false, "build EPUB but skip sending")
	flag.Parse()

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	log.Printf("fetching from %d feed(s), max %d articles", len(cfg.Feeds), cfg.MaxArticles)
	articles, err := FetchArticles(cfg.Feeds, cfg.MaxArticles)
	if err != nil {
		log.Fatalf("fetch: %v", err)
	}
	if len(articles) == 0 {
		log.Println("no articles found, nothing to do")
		os.Exit(0)
	}
	log.Printf("fetched %d articles", len(articles))

	epubPath := fmt.Sprintf("briefme-%s.epub", time.Now().Format("2006-01-02"))
	if err := BuildEPUB(articles, epubPath); err != nil {
		log.Fatalf("build epub: %v", err)
	}
	log.Printf("built EPUB: %s", epubPath)

	if *dryRun {
		log.Println("--dry-run: skipping send")
		return
	}

	if err := SendEPUB(cfg.SMTP, cfg.KoboEmail, epubPath); err != nil {
		log.Fatalf("send: %v", err)
	}
	log.Printf("sent to %s", cfg.KoboEmail)
}
