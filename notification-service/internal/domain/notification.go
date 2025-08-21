package domain

import "time"

type Notification struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Author    string    `json:"author"`
	Content   string    `json:"content"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"created_at"`
}

// Отличается не только название, но и сами поля, смотри:

// package domain

// import "time"

// type Notification struct {
// 	ID        string    `json:"id"`
// 	Title     string    `json:"title"`
// 	Author    string    `json:"author"`
// 	Content   string    `json:"content"`
// 	Tags      []string  `json:"tags"`
// 	CreatedAt time.Time `json:"created_at"`
// }
