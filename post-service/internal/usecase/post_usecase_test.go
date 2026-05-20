package usecase

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"post-service/internal/domain"
)

type fakePostRepo struct {
	savedPost  *domain.Post
	savedEvent domain.EventEnvelope
}

func (f *fakePostRepo) Save(ctx context.Context, post *domain.Post) error {
	return nil
}

func (f *fakePostRepo) SaveWithOutbox(ctx context.Context, post *domain.Post, event domain.EventEnvelope) error {
	f.savedPost = post
	f.savedEvent = event
	return nil
}

func (f *fakePostRepo) GetByID(ctx context.Context, id string) (*domain.Post, error) {
	return nil, nil
}

func (f *fakePostRepo) List(ctx context.Context) ([]*domain.Post, error) {
	return nil, nil
}

type fakeCache struct{}

func (f *fakeCache) GetPost(ctx context.Context, id string) (*domain.Post, error) {
	return nil, nil
}

func (f *fakeCache) SavePost(ctx context.Context, post *domain.Post, ttl time.Duration) error {
	return nil
}

func TestCreatePostCreatesEventEnvelope(t *testing.T) {
	repo := &fakePostRepo{}
	uc := NewPostUsecase(repo, &fakeCache{}, slog.New(slog.NewTextHandler(os.Stdout, nil)))

	post, err := uc.CreatePost(context.Background(), "Title", "Author", "Content", []string{"go"})
	if err != nil {
		t.Fatalf("CreatePost returned error: %v", err)
	}

	if post.ID == "" {
		t.Fatal("expected post id to be set")
	}

	event := repo.savedEvent
	if event.EventID == "" {
		t.Fatal("expected event id to be set")
	}
	if event.EventType != domain.EventTypePostCreated {
		t.Fatalf("expected event type %q, got %q", domain.EventTypePostCreated, event.EventType)
	}
	if event.Version != domain.EventVersion {
		t.Fatalf("expected event version %d, got %d", domain.EventVersion, event.Version)
	}
	if event.Payload.ID != post.ID {
		t.Fatalf("expected payload id %q, got %q", post.ID, event.Payload.ID)
	}
	if event.Payload.Title != post.Title {
		t.Fatalf("expected payload title %q, got %q", post.Title, event.Payload.Title)
	}
}
