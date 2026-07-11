package web

import (
	"log"
	"time"
)

const fetchHour = 6 // 06:00 local time

// runScheduler runs an ingest immediately, then daily at fetchHour.
func (srv *Server) runScheduler() {
	log.Printf("web: running initial fetch")
	srv.ingest(srv.cfg, srv.store)

	for {
		next := nextFetchTime()
		log.Printf("web: next fetch scheduled at %s", next.Format(time.RFC3339))
		time.Sleep(time.Until(next))
		log.Printf("web: running scheduled fetch")
		srv.ingest(srv.cfg, srv.store)
	}
}

func nextFetchTime() time.Time {
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day(), fetchHour, 0, 0, 0, now.Location())
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next
}
