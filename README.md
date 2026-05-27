# briefme

A Go CLI that fetches articles from RSS feeds, packages them as an EPUB, and delivers the result to your Kobo e-reader via email.

## How it works

```
RSS feeds → fetch articles → build EPUB → email to Kobo
```

Kobo's "Send to Kobo" feature accepts EPUBs sent to your device's linked email address (via Dropbox). `briefme` automates the whole pipeline so your reading list is waiting on the device when you pick it up.

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

kobo_email: "your-kobo-email@kobo.com"   # the email linked to your Kobo

smtp:
  host: "smtp.gmail.com"
  port: 587
  username: "you@gmail.com"
  password: "your-app-password"           # use an app password, not your main password
  from: "you@gmail.com"

max_articles: 20                          # cap on total articles per run (default: 20)
```

### Finding your Kobo email address

1. On your Kobo device: **Settings → My Kobo → Kobo and Dropbox**
2. Or in the Kobo app: **Settings → Kobo**
3. The address looks like `<name>@kobo.com`

### Gmail setup

Gmail requires an [App Password](https://support.google.com/accounts/answer/185833) rather than your main password. Generate one under **Google Account → Security → App passwords**.

## Usage

```bash
# Build the EPUB and send it to your Kobo
briefme --config config.yaml

# Build the EPUB locally without sending (useful for testing)
briefme --config config.yaml --dry-run

# Default config path is ./config.yaml
briefme
```

The `--dry-run` flag writes a `briefme-YYYY-MM-DD.epub` file in the current directory so you can inspect it in any EPUB reader before committing to sending.

## Running on a schedule

Use cron to get a daily delivery:

```cron
# Every morning at 7am
0 7 * * * /usr/local/bin/briefme --config /home/user/briefme/config.yaml >> /var/log/briefme.log 2>&1
```

## Project layout

```
briefme/
├── main.go               # CLI entry point
├── config.go             # config loading and validation
├── fetcher.go            # RSS/Atom feed fetching and deduplication
├── builder.go            # EPUB assembly
├── mailer.go             # SMTP email with EPUB attachment
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
| [`gopkg.in/gomail.v2`](https://pkg.go.dev/gopkg.in/gomail.v2) | Email with MIME attachments |
| [`gopkg.in/yaml.v3`](https://pkg.go.dev/gopkg.in/yaml.v3) | YAML config parsing |

## Running tests

```bash
go test ./...
```
