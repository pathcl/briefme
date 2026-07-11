package web

import (
	"log"
	"net/http"
	"strings"
	"time"
)

func (srv *Server) handle(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		today := time.Now().Format("2006-01-02")
		http.Redirect(w, r, "/"+today, http.StatusFound)
		return
	}
	if _, err := time.Parse("2006-01-02", path); err != nil {
		http.NotFound(w, r)
		return
	}
	srv.renderDate(w, r, path)
}

func (srv *Server) renderDate(w http.ResponseWriter, r *http.Request, date string) {
	today := time.Now().Format("2006-01-02")
	month := date[:7] // "2026-06"

	datesInMonth, err := srv.store.GetDatesInMonth(month)
	if err != nil {
		log.Printf("web: GetDatesInMonth: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	// Ensure selected date appears in the calendar even if it has no articles.
	datesInMonth[date] = datesInMonth[date] // no-op if already present

	cal := buildCalendar(date, today, datesInMonth)

	var sections []categorySection
	for _, cat := range srv.categories() {
		articles, err := srv.store.GetArticlesByDate(cat, date)
		if err != nil {
			log.Printf("web: GetArticlesByDate(%s, %s): %v", cat, date, err)
			continue
		}
		if len(articles) == 0 {
			continue
		}
		sec := categorySection{Name: cat}
		for _, a := range articles {
			sec.Articles = append(sec.Articles, srv.articleToWeb(a))
		}
		sections = append(sections, sec)
	}

	data := pageData{
		Date:     date,
		Today:    today,
		Calendar: cal,
		Sections: sections,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("web: template: %v", err)
	}
}

func (srv *Server) handleTagIndex(w http.ResponseWriter, r *http.Request) {
	allTags, err := srv.store.GetAllTags()
	if err != nil {
		log.Printf("web: GetAllTags: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	data := tagsPageData{AllTags: allTags}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tagsTmpl.Execute(w, data); err != nil {
		log.Printf("web: tags template: %v", err)
	}
}

func (srv *Server) handleTagArticles(w http.ResponseWriter, r *http.Request) {
	tag := r.PathValue("tag")

	allTags, _ := srv.store.GetAllTags()
	articles, err := srv.store.GetArticlesByTag(tag)
	if err != nil {
		log.Printf("web: GetArticlesByTag(%s): %v", tag, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Group by category.
	grouped := make(map[string][]webArticle)
	var order []string
	seen := make(map[string]bool)
	for _, a := range articles {
		wa := srv.articleToWeb(a)
		if !seen[a.Category] {
			seen[a.Category] = true
			order = append(order, a.Category)
		}
		grouped[a.Category] = append(grouped[a.Category], wa)
	}
	var sections []categorySection
	for _, cat := range order {
		sections = append(sections, categorySection{Name: cat, Articles: grouped[cat]})
	}

	data := tagsPageData{
		ActiveTag: tag,
		AllTags:   allTags,
		Sections:  sections,
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tagsTmpl.Execute(w, data); err != nil {
		log.Printf("web: tags template: %v", err)
	}
}

func (srv *Server) handleAddTag(w http.ResponseWriter, r *http.Request) {
	url := r.FormValue("url")
	tag := strings.TrimSpace(strings.ToLower(r.FormValue("tag")))
	ref := r.FormValue("ref") // page to redirect back to

	if url == "" || tag == "" {
		http.Error(w, "url and tag required", http.StatusBadRequest)
		return
	}
	if err := srv.store.AddTag(url, tag); err != nil {
		log.Printf("web: AddTag: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if ref == "" {
		ref = "/"
	}
	http.Redirect(w, r, ref, http.StatusSeeOther)
}

func (srv *Server) handleRemoveTag(w http.ResponseWriter, r *http.Request) {
	url := r.FormValue("url")
	tag := r.FormValue("tag")
	ref := r.FormValue("ref")

	if url == "" || tag == "" {
		http.Error(w, "url and tag required", http.StatusBadRequest)
		return
	}
	if err := srv.store.RemoveTag(url, tag); err != nil {
		log.Printf("web: RemoveTag: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if ref == "" {
		ref = "/"
	}
	http.Redirect(w, r, ref, http.StatusSeeOther)
}

func containsDate(dates []string, d string) bool {
	for _, x := range dates {
		if x == d {
			return true
		}
	}
	return false
}
