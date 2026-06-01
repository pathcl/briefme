# briefme

A Go CLI that fetches articles from RSS feeds, packages them as EPUBs per category, and copies the results to your Kobo e-reader over USB.

## What you get

- **Pure text, nothing else.** Every EPUB contains only article text — no images, no ads, no tracking pixels, no JavaScript. Exactly what you want to read on an e-reader.
- **One EPUB per category.** Feeds are tagged as `news` or `papers`. Each run produces separate files (`briefme-news-YYYY-MM-DD.epub`, `briefme-papers-YYYY-MM-DD.epub`) so you can read them independently.
- **Daily accumulation.** Run `briefme` multiple times in a day and each new article is appended — the EPUB grows as new content is found. Already-seen articles are never duplicated.

## How it works

```
RSS/arXiv feeds
       ↓
  fetch & scrape full article text
       ↓
  deduplicate (SQLite)
       ↓
  build one EPUB per category
       ↓
  copy to Kobo over USB
```

When your Kobo is connected via USB it mounts as a regular drive. `briefme` writes dated EPUBs directly to it. Disconnect and the files appear in your library.

## Installation

Pre-built binaries for Linux, macOS, and Windows are available on the [releases page](https://github.com/pathcl/briefme/releases).

Or install with Go:

```bash
go install github.com/pathcl/briefme/cmd/briefme@latest
```

Or build from source:

```bash
git clone https://github.com/pathcl/briefme
cd briefme
go build -o briefme ./cmd/briefme
```

## Configuration

Copy the example config and edit it:

```bash
cp config.yaml.example config.yaml
```

```yaml
feeds:
  - url: "https://hnrss.org/frontpage"
    name: "Hacker News"
    category: "news"          # → briefme-news-YYYY-MM-DD.epub

  - url: "https://arxiv.org/rss/cs.AI"
    name: "arXiv CS.AI"
    category: "papers"        # → briefme-papers-YYYY-MM-DD.epub

# Mount path of your Kobo when connected via USB.
# Leave empty to let briefme auto-detect it.
kobo_path: ""

max_per_feed: 5   # articles fetched per feed per run
db_path: "briefme.db"
```

### Categories

Each feed has a `category` field. Two categories ship in the example config:

| Category | Output file | Feeds |
|---|---|---|
| `news` | `briefme-news-YYYY-MM-DD.epub` | Hacker News, Ars Technica, Quanta Magazine, The Register |
| `papers` | `briefme-papers-YYYY-MM-DD.epub` | arXiv CS.AI, arXiv CS.LG |

You can add your own categories — any string is valid. Each distinct category produces its own EPUB.

### Finding the Kobo mount path

Connect the Kobo via USB and look for it at:

| OS | Typical path |
|---|---|
| macOS | `/Volumes/KOBOeReader` |
| Linux | `/media/<username>/KOBOeReader` or `/run/media/<username>/KOBOeReader` |
| Windows | `E:\` (or whichever drive letter Windows assigns) |

If you leave `kobo_path` empty, `briefme` checks the common locations above automatically.

## Usage

```bash
# Plug in your Kobo, then run:
briefme --config config.yaml

# Default config path is ./config.yaml
briefme
```

Each run produces one EPUB per category and copies them to the Kobo. Articles already on the device (same SHA-256) are skipped.

## Releases

`briefme` uses [GoReleaser](https://goreleaser.com/) via GitHub Actions. To cut a release:

```bash
git tag -a v0.1.0 -m "Initial release"
git push origin v0.1.0
```

The CI pipeline builds binaries for Linux, macOS, and Windows (amd64 + arm64 where applicable), produces checksums, and publishes everything to the GitHub releases page automatically.

## Automating the workflow

Because the Kobo needs to be physically connected, automation is most useful as a script you run before a commute:

```bash
#!/bin/bash
# sync-kobo.sh — run once when you plug in the Kobo
briefme --config ~/briefme/config.yaml && echo "Kobo ready — safe to eject"
```

### Linux: fix "permission denied" on write

FAT32 devices mount without a `uid=` option by default, so ownership goes to root. `remount` cannot change `uid` on FAT — you must unmount and remount from scratch.

**Quick fix** (lasts until next replug):

```bash
sudo umount /media/kobo
sudo mount -t vfat -o uid=$(id -u),gid=$(id -g),fmask=0022,dmask=0022 /dev/sda /media/kobo
```

**Permanent fix** via `/etc/fstab`:

```bash
# Find the UUID of the device
sudo blkid /dev/sda

# Add to /etc/fstab (replace UUID, device, and uid/gid with your values):
UUID=XXXX-XXXX  /media/kobo  vfat  uid=1000,gid=1000,fmask=0022,dmask=0022,nofail  0  0
```

After editing fstab: `sudo umount /media/kobo && sudo mount -a` to verify, then replug.

### Linux: run on USB connect

```
# /etc/udev/rules.d/99-kobo-sync.rules
ACTION=="add", SUBSYSTEM=="block", ENV{ID_FS_LABEL}=="KOBOeReader", \
  RUN+="/usr/local/bin/briefme --config /home/<username>/briefme/config.yaml"
```

## Project layout

```
briefme/
├── cmd/briefme/main.go       # CLI entry point
├── internal/
│   ├── config/               # YAML config loading
│   ├── feed/                 # RSS fetching, scraping, arXiv HTML extraction
│   ├── epub/                 # EPUB assembly (one per category)
│   ├── store/                # SQLite deduplication + EPUB checksum tracking
│   ├── deliver/              # copy EPUBs to Kobo (auto-detect mount)
│   └── model/                # shared Article struct
├── config.yaml.example
├── go.mod
└── go.sum
```

## Dependencies

| Library | Purpose |
|---|---|
| [`github.com/mmcdole/gofeed`](https://github.com/mmcdole/gofeed) | RSS/Atom feed parsing |
| [`github.com/bmaupin/go-epub`](https://github.com/bmaupin/go-epub) | EPUB generation |
| [`github.com/go-shiori/go-readability`](https://github.com/go-shiori/go-readability) | Full article text extraction |
| [`github.com/PuerkitoBio/goquery`](https://github.com/PuerkitoBio/goquery) | HTML parsing for text cleanup |
| [`modernc.org/sqlite`](https://pkg.go.dev/modernc.org/sqlite) | Pure-Go SQLite (no CGO — works in cross-compiled binaries) |
| [`gopkg.in/yaml.v3`](https://pkg.go.dev/gopkg.in/yaml.v3) | YAML config parsing |

## Running tests

```bash
go test ./...
```
