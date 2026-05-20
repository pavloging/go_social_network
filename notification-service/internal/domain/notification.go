package domain

import "time"

type Notification struct {
	EventID   string    `json:"event_id"`
	PostID    string    `json:"post_id"`
	Author    string    `json:"author"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}
