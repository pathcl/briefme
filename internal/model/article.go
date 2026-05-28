package model

import "time"

type Article struct {
	Title       string
	Author      string
	URL         string
	Content     string
	PublishedAt time.Time
	FeedName    string
	Category    string
}
