package redis

import (
	"context"
	"encoding/json"
	"time"

	"post-service/internal/domain"

	"log/slog"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	client *redis.Client
	log    *slog.Logger
}

func NewRedisCache(addr string, db int, log *slog.Logger) *Cache {
	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
		DB:   db,
	})

	log.Info("Redis address", slog.String("address", addr))

	// Healthcheck Redis при запуске
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	start := time.Now()
	pong, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Error("Redis healthcheck failed",
			slog.String("error", err.Error()),
			slog.Duration("duration", time.Since(start)))
		panic("Redis healthcheck failed")
	}

	log.Info("Redis connected",
		slog.String("pong", pong),
		slog.Duration("duration", time.Since(start)))

	return &Cache{
		client: rdb,
		log:    log,
	}
}

func (c *Cache) SavePost(ctx context.Context, post *domain.Post, ttl time.Duration) error {
	start := time.Now()

	data, err := json.Marshal(post)
	if err != nil {
		c.log.Error("Failed to marshal post",
			slog.String("post_id", post.ID),
			slog.String("error", err.Error()),
			slog.Duration("duration", time.Since(start)))
		return err
	}

	err = c.client.Set(ctx, "post:"+post.ID, data, ttl).Err()
	if err != nil {
		c.log.Error("Failed to save post to Redis",
			slog.String("post_id", post.ID),
			slog.String("error", err.Error()),
			slog.Duration("duration", time.Since(start)),
			slog.Int("data_size", len(data)))
		return err
	}

	c.log.Debug("Post saved to cache",
		slog.String("post_id", post.ID),
		slog.Duration("ttl", ttl),
		slog.Duration("duration", time.Since(start)),
		slog.Int("data_size", len(data)))
	return nil
}

func (c *Cache) GetPost(ctx context.Context, id string) (*domain.Post, error) {
	start := time.Now()

	data, err := c.client.Get(ctx, "post:"+id).Bytes()
	if err == redis.Nil {
		c.log.Debug("Cache miss",
			slog.String("post_id", id),
			slog.Duration("duration", time.Since(start)))
		return nil, nil
	}
	if err != nil {
		c.log.Error("Failed to get post from Redis",
			slog.String("post_id", id),
			slog.String("error", err.Error()),
			slog.Duration("duration", time.Since(start)))
		return nil, err
	}

	var post domain.Post
	if err := json.Unmarshal(data, &post); err != nil {
		c.log.Error("Failed to unmarshal post data",
			slog.String("post_id", id),
			slog.String("error", err.Error()),
			slog.Duration("duration", time.Since(start)),
			slog.Int("data_size", len(data)))
		return nil, err
	}

	c.log.Debug("Cache hit",
		slog.String("post_id", id),
		slog.Duration("duration", time.Since(start)),
		slog.Int("data_size", len(data)))
	return &post, nil
}

func (c *Cache) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}
