# briefme

A Go CLI that fetches articles from RSS feeds, packages them as an EPUB, and copies the result to your Kobo e-reader over USB.

## How it works

```
RSS feeds → fetch articles → build EPUB → copy to Kobo (USB)
```

When your Kobo is connected via USB it mounts as a regular drive. `briefme` writes a dated EPUB directly to it. Disconnect the Kobo and the file is in your library.

## Installation

```bash
go install github.com/pathcl/briefme@latest
```

Or build from source:

```bash
git clone https://github.com/pathcl/briefme
cd briefme
go build -o briefme .
```

## Configuration

Copy the example config and fill in your details:

```bash
cp config.yaml.example config.yaml
```

```yaml
feeds:
  - url: "https://hnrss.org/frontpage"
    name: "Hacker News"
  - url: "https://example.com/feed.xml"
    name: "My Blog"

# Mount path of your Kobo when connected via USB.
# Leave empty to let briefme auto-detect it.
kobo_path: ""

max_articles: 20    # cap on total articles per run (default: 20)
```

### Finding the Kobo mount path

Connect the Kobo via USB and look for it at:

| OS | Typical path |
|---|---|
| macOS | `/Volumes/KOBOeReader` |
| Linux | `/media/<username>/KOBOeReader` or `/run/media/<username>/KOBOeReader` |
| Windows | `E:\` (or whichever drive letter Windows assigns) |

If you leave `kobo_path` empty, `briefme` will check the common locations above automatically.

## Usage

```bash
# Plug in your Kobo, then run:
briefme --config config.yaml

# Build the EPUB locally without copying to the Kobo:
briefme --config config.yaml --dry-run

# Default config path is ./config.yaml
briefme
```

The `--dry-run` flag writes `briefme-YYYY-MM-DD.epub` in the current directory so you can open it in any EPUB reader before committing.

## Automating the workflow

Because the Kobo needs to be physically connected, automation is most useful as a script you run before a commute or before going offline:

```bash
#!/bin/bash
# sync-kobo.sh — run once when you plug in the Kobo
briefme --config ~/briefme/config.yaml && echo "Kobo ready — safe to eject"
```

On Linux you can trigger it automatically on USB mount with a udev rule:

```
# /etc/udev/rules.d/99-kobo-sync.rules
ACTION=="add", SUBSYSTEM=="block", ENV{ID_FS_LABEL}=="KOBOeReader", \
  RUN+="/usr/local/bin/briefme --config /home/<username>/briefme/config.yaml"
```

## Project layout

```
briefme/
├── main.go               # CLI entry point
├── config.go             # config loading and validation
├── fetcher.go            # RSS/Atom feed fetching and deduplication
├── builder.go            # EPUB assembly
├── delivery.go           # copy EPUB to Kobo mount path (with auto-detection)
├── config.yaml.example   # example configuration
├── *_test.go             # tests for each component
├── go.mod
└── go.sum
```

## Dependencies

| Library | Purpose |
|---|---|
| [`github.com/mmcdole/gofeed`](https://github.com/mmcdole/gofeed) | RSS/Atom feed parsing |
| [`github.com/bmaupin/go-epub`](https://github.com/bmaupin/go-epub) | EPUB generation |
| [`gopkg.in/yaml.v3`](https://pkg.go.dev/gopkg.in/yaml.v3) | YAML config parsing |

## Running tests

```bash
go test ./...
```
