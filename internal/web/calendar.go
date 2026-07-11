package web

import (
	"fmt"
	"time"
)

type calendarData struct {
	Title     string
	MonthKey  string // "2026-06"
	PrevURL   string // "/2026-05-01"
	NextURL   string // "/2026-07-01"
	Weeks     []calWeek
}

type calWeek [7]calDay

type calDay struct {
	Day      int    // 0 = padding cell
	Date     string // "2026-06-01"
	HasData  bool
	IsToday  bool
	Selected bool
}

// buildCalendar generates a month calendar for the month containing date.
// datesWithData is the set of dates in that month that have articles.
func buildCalendar(date, today string, datesWithData map[string]bool) calendarData {
	t, _ := time.Parse("2006-01-02", date)
	year, month := t.Year(), t.Month()

	first := time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
	daysInMonth := time.Date(year, month+1, 0, 0, 0, 0, 0, time.Local).Day()

	prev := first.AddDate(0, -1, 0)
	next := first.AddDate(0, 1, 0)

	// Monday-first: Sun→6, Mon→0, Tue→1, …
	startOffset := (int(first.Weekday()) + 6) % 7

	var weeks []calWeek
	var week calWeek
	col := startOffset

	for day := 1; day <= daysInMonth; day++ {
		d := fmt.Sprintf("%d-%02d-%02d", year, month, day)
		week[col] = calDay{
			Day:      day,
			Date:     d,
			HasData:  datesWithData[d],
			IsToday:  d == today,
			Selected: d == date,
		}
		col++
		if col == 7 {
			weeks = append(weeks, week)
			week = calWeek{}
			col = 0
		}
	}
	if col > 0 {
		weeks = append(weeks, week)
	}

	return calendarData{
		Title:    first.Format("January 2006"),
		MonthKey: first.Format("2006-01"),
		PrevURL:  fmt.Sprintf("/%s", prev.Format("2006-01-02")),
		NextURL:  fmt.Sprintf("/%s", next.Format("2006-01-02")),
		Weeks:    weeks,
	}
}
