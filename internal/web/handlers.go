package web

import (
	"log"
	"net/http"
	"strconv"
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
	month := date[:7]

	page := 1
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 1 {
		page = p
	}
	offset := (page - 1) * pageSize

	datesInMonth, err := srv.store.GetDatesInMonth(month)
	if err != nil {
		log.Printf("web: GetDatesInMonth: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	datesInMonth[date] = datesInMonth[date]

	cal := buildCalendar(date, today, datesInMonth)

	articles, total, err := srv.store.GetArticlesByDatePaged(date, pageSize, offset)
	if err != nil {
		log.Printf("web: GetArticlesByDatePaged(%s): %v", date, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Group into category sections preserving order.
	grouped := make(map[string]*categorySection)
	var order []string
	for _, a := range articles {
		if _, ok := grouped[a.Category]; !ok {
			grouped[a.Category] = &categorySection{Name: a.Category}
			order = append(order, a.Category)
		}
		grouped[a.Category].Articles = append(grouped[a.Category].Articles, srv.articleToWeb(a))
	}
	var sections []categorySection
	for _, cat := range order {
		sections = append(sections, *grouped[cat])
	}

	prevPage, nextPage := 0, 0
	if page > 1 {
		prevPage = page - 1
	}
	if offset+pageSize < total {
		nextPage = page + 1
	}

	data := pageData{
		Date:     date,
		Today:    today,
		Calendar: cal,
		Sections: sections,
		Page:     page,
		PrevPage: prevPage,
		NextPage: nextPage,
		Total:    total,
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

func (srv *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	if srv.cfg.RefreshKey == "" {
		http.Error(w, "refresh not configured", http.StatusForbidden)
		return
	}
	if r.FormValue("key") != srv.cfg.RefreshKey {
		http.Error(w, "invalid key", http.StatusUnauthorized)
		return
	}
	go srv.ingest(srv.cfg, srv.store)
	ref := r.FormValue("ref")
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
