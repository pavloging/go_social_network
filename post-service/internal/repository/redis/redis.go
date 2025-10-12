package redis

import (
	"context"
	"encoding/json"
	"time"

	"post-service/internal/domain"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	client *redis.Client
}

func NewRedisCache(addr string, db int) *Cache {
	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
		DB:   db,
	})

	println("Adress: " + addr)

	// Healthcheck Redis при запуске
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	pong, err := rdb.Ping(ctx).Result()
	if err != nil {
		println("❌ Redis healthcheck failed:", err.Error())
		panic("❌ Redis healthcheck failed:")
	} else {
		println("✅ Redis connected:", pong)
	}

	return &Cache{client: rdb}
}

// Сохраняем пост в Redis
func (c *Cache) SavePost(ctx context.Context, post *domain.Post, ttl time.Duration) error {
	data, err := json.Marshal(post)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, "post:"+post.ID, data, ttl).Err()
}

// Достаём пост из Redis
func (c *Cache) GetPost(ctx context.Context, id string) (*domain.Post, error) {
	data, err := c.client.Get(ctx, "post:"+id).Bytes()
	if err == redis.Nil {
		return nil, nil // кеш-мисс
	}
	if err != nil {
		return nil, err
	}

	var post domain.Post
	if err := json.Unmarshal(data, &post); err != nil {
		return nil, err
	}
	return &post, nil
}

// func (c *Cache) List(ctx context.Context) ([]*domain.Post, error) {

// 	return nil, nil
// }
