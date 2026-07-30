# briefme

A Go application that fetches articles from RSS feeds, stores them in SQLite, and serves them as a clean reading web UI. Articles can also be packaged as EPUBs and delivered to a Kobo e-reader over USB.

## What you get

- **Clean reading UI.** Browse articles by date with a monthly calendar nav. External media (images, videos, iframes) is blocked by default and revealed on demand.
- **Tagging.** Tag articles manually or accept keyword suggestions. Browse your tags at `/tags`.
- **One EPUB per category.** Each run produces `briefme-news-YYYY-MM-DD.epub`, `briefme-papers-YYYY-MM-DD.epub`, etc. for Kobo delivery.
- **No duplicates.** SQLite tracks seen articles across runs.
- **Resilient fetching.** A failing or timing-out feed is skipped; the rest continue.

## How it works

```
RSS/arXiv feeds
      ↓
 fetch & scrape full article text (non-HTML content rejected)
      ↓
 deduplicate (SQLite)
      ↓
 serve via web UI   ─── or ───   build EPUBs → copy to Kobo
```

## Quick start

```bash
git clone https://github.com/pathcl/briefme
cd briefme
cp config.yaml.example config.yaml   # edit feeds, db_path, refresh_key
make serve                            # builds and starts the web server
```

Open `http://localhost:8080`.

## Makefile

| Command | What it does |
|---|---|
| `make build` | Compile binary to `./briefme` |
| `make serve` | Build and start the web server on `0.0.0.0:8080` |
| `make run` | Build and run the fetch/EPUB mode |
| `make test` | Run the full test suite |
| `make clean` | Remove the binary |

## Configuration

```yaml
feeds:
  - url: "https://hnrss.org/frontpage"
    name: "Hacker News"
    category: "news"

  - url: "https://arxiv.org/rss/cs.AI"
    name: "arXiv CS.AI"
    category: "papers"

kobo_path: ""        # leave empty to auto-detect
max_per_feed: 5      # articles fetched per feed per run
db_path: "briefme.db"

# Secret phrase for the on-demand refresh button in the web UI.
# Prefer the env var so the secret is never committed.
refresh_key: ""
```

`config.yaml` is gitignored. Copy from `config.yaml.example` and keep it local.

### refresh_key

Set a secret phrase to enable the **↻ refresh** button in the navbar. Clicking it reveals a password input; submitting the correct key triggers an immediate feed fetch in the background.

The key can also be set via environment variable (takes precedence over the config):

```bash
BRIEFME_REFRESH_KEY=yourphrase briefme serve
```

## Web UI

| Route | Description |
|---|---|
| `GET /` | Redirects to today's date |
| `GET /YYYY-MM-DD` | Articles for that date, paginated (20 per page) |
| `GET /YYYY-MM-DD?page=N` | Subsequent pages |
| `GET /tags` | Tag index with article counts |
| `GET /tags/{tag}` | All articles carrying that tag |
| `POST /refresh` | Trigger an immediate fetch (requires `key` form field) |

### Media toggle

External images, videos, and iframes are blocked on page load. Click **show media** in the navbar to reveal them. The preference is stored in `localStorage` and persists across pages.

## CLI mode (Kobo)

```bash
briefme --config config.yaml           # fetch, build EPUBs, deliver to Kobo
briefme --config config.yaml --dry-run # build EPUBs locally, skip Kobo copy
```

## Subcommands

```
briefme [flags]          # fetch + EPUB + Kobo delivery (default)
briefme serve [flags]    # start the web server

serve flags:
  -config string   path to config file (default "config.yaml")
  -port   string   HTTP listen port (default "8080")
  -bind   string   HTTP bind address (default "0.0.0.0")
```

## Deployment on Fly.io

```bash
# One-time setup
fly launch --no-deploy
fly volumes create briefme_data --size 1 --region ams
fly secrets set BRIEFME_REFRESH_KEY=yourphrase

# Deploy
fly deploy
```

The included `fly.toml` mounts the volume at `/data`. Set `db_path: /data/briefme.db` in your config so the database survives deploys. Keep `min_machines_running = 1` so the daily scheduler stays alive.

## Scheduled fetch

When running `briefme serve`, an initial fetch runs on startup and then daily at **06:00 local time**. Use the **↻ refresh** button for on-demand fetches.

## Project layout

```
briefme/
├── cmd/briefme/main.go       # CLI entry point (serve + fetch subcommands)
├── internal/
│   ├── config/               # YAML config loading + env var overrides
│   ├── feed/                 # RSS fetching, scraping, arXiv HTML extraction
│   ├── epub/                 # EPUB assembly (one per category)
│   ├── store/                # SQLite: articles, tags, EPUB checksums
│   ├── deliver/              # copy EPUBs to Kobo (auto-detect mount)
│   ├── web/                  # HTTP server, handlers, templates, scheduler
│   └── model/                # shared Article struct
├── config.yaml.example
├── Dockerfile
├── fly.toml
├── Makefile
├── go.mod
└── go.sum
```

## Dependencies

| Library | Purpose |
|---|---|
| [`github.com/mmcdole/gofeed`](https://github.com/mmcdole/gofeed) | RSS/Atom feed parsing |
| [`github.com/bmaupin/go-epub`](https://github.com/bmaupin/go-epub) | EPUB generation |
| [`github.com/go-shiori/go-readability`](https://github.com/go-shiori/go-readability) | Full article text extraction |
| [`github.com/PuerkitoBio/goquery`](https://github.com/PuerkitoBio/goquery) | HTML parsing for media handling |
| [`modernc.org/sqlite`](https://pkg.go.dev/modernc.org/sqlite) | Pure-Go SQLite (no CGO) |
| [`gopkg.in/yaml.v3`](https://pkg.go.dev/gopkg.in/yaml.v3) | YAML config parsing |

## Running tests

```bash
make test
# or
go test ./...
```
