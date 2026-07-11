package web

import (
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"regexp"
	"sort"
	"time"

	"github.com/pathcl/briefme/internal/config"
	"github.com/pathcl/briefme/internal/model"
	"github.com/pathcl/briefme/internal/store"
)

var svgRe = regexp.MustCompile(`(?si)<svg[\s>].*?</svg>`)

//go:embed templates
var templateFS embed.FS

var (
	tmpl     = template.Must(template.ParseFS(templateFS, "templates/layout.html"))
	tagsTmpl = template.Must(template.ParseFS(templateFS, "templates/tags.html"))
)

// IngestFunc is the function the server calls on each scheduled fetch.
type IngestFunc func(cfg *config.Config, db *store.Store)

type Server struct {
	store  *store.Store
	cfg    *config.Config
	addr   string
	ingest IngestFunc
}

// pageData is passed to layout.html.
type pageData struct {
	Date     string
	Today    string
	Calendar calendarData
	Sections []categorySection
}

// tagsPageData is passed to tags.html.
type tagsPageData struct {
	ActiveTag string
	AllTags   []store.TagCount
	Sections  []categorySection // populated only when ActiveTag != ""
}

type categorySection struct {
	Name     string
	Articles []webArticle
}

type webArticle struct {
	Title         string
	URL           string
	FeedName      string
	Category      string
	PublishedAt   time.Time
	Content       template.HTML
	Tags          []string
	SuggestedTags []string
}

func New(s *store.Store, cfg *config.Config, bind, port string, ingest IngestFunc) *Server {
	return &Server{
		store:  s,
		cfg:    cfg,
		addr:   fmt.Sprintf("%s:%s", bind, port),
		ingest: ingest,
	}
}

func (srv *Server) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /tags/{tag}", srv.handleTagArticles)
	mux.HandleFunc("GET /tags", srv.handleTagIndex)
	mux.HandleFunc("POST /tag", srv.handleAddTag)
	mux.HandleFunc("POST /untag", srv.handleRemoveTag)
	mux.HandleFunc("/", srv.handle)
	return mux
}

// ServeHTTP implements http.Handler so the server can be used in tests directly.
func (srv *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	srv.handler().ServeHTTP(w, r)
}

// Start launches the background daily fetch and the HTTP server.
func (srv *Server) Start() error {
	go srv.runScheduler()
	log.Printf("web: listening on http://%s", srv.addr)
	return http.ListenAndServe(srv.addr, srv.handler())
}

// categories returns the sorted distinct categories from config feeds.
func (srv *Server) categories() []string {
	seen := make(map[string]bool)
	var out []string
	for _, f := range srv.cfg.Feeds {
		if !seen[f.Category] {
			seen[f.Category] = true
			out = append(out, f.Category)
		}
	}
	sort.Strings(out)
	return out
}

func stripSVG(html string) string {
	return svgRe.ReplaceAllString(html, "")
}

func (srv *Server) articleToWeb(a model.Article) webArticle {
	tags, _ := srv.store.GetTagsForArticle(a.URL)
	content := stripSVG(a.Content)
	suggested := SuggestTags(content, 8)

	// Remove already-applied tags from suggestions.
	applied := make(map[string]bool, len(tags))
	for _, t := range tags {
		applied[t] = true
	}
	var filtered []string
	for _, s := range suggested {
		if !applied[s] {
			filtered = append(filtered, s)
		}
	}

	return webArticle{
		Title:         a.Title,
		URL:           a.URL,
		FeedName:      a.FeedName,
		Category:      a.Category,
		PublishedAt:   a.PublishedAt,
		Content:       template.HTML(content),
		Tags:          tags,
		SuggestedTags: filtered,
	}
}
