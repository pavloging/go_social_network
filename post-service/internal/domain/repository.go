package domain

// PostRepository — интерфейс для работы с БД
type PostRepository interface {
	Save(post *Post) error
	GetByID(id string) (*Post, error)
	List() ([]*Post, error)
}

// EventProducer — интерфейс для отправки событий (Kafka)
type EventProducer interface {
	Publish(post *Post) error
}
