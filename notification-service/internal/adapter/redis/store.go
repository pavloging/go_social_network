package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"notification-service/internal/domain"

	"github.com/redis/go-redis/v9"
)

const saveNotificationOnceScript = `
local processedKey = KEYS[1]
local notificationsKey = KEYS[2]
local notification = ARGV[1]
local ttlSeconds = tonumber(ARGV[2])

if redis.call("EXISTS", processedKey) == 1 then
    return 0
end

redis.call("LPUSH", notificationsKey, notification)
redis.call("LTRIM", notificationsKey, 0, 99)
redis.call("SET", processedKey, "true", "EX", ttlSeconds)

return 1
`

type Store struct {
	client       *redis.Client
	log          *slog.Logger
	processedTTL time.Duration
}

func NewStore(addr, password string, db int, processedTTL time.Duration, log *slog.Logger) (*Store, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	log.Info("Redis connected", slog.String("address", addr))

	return &Store{
		client:       client,
		log:          log,
		processedTTL: processedTTL,
	}, nil
}

func (s *Store) SaveNotificationOnce(ctx context.Context, eventID string, notification domain.Notification) (bool, error) {
	data, err := json.Marshal(notification)
	if err != nil {
		return false, fmt.Errorf("marshal notification: %w", err)
	}

	processedKey := "processed_events:" + eventID
	ttlSeconds := int(s.processedTTL.Seconds())
	if ttlSeconds <= 0 {
		ttlSeconds = int((168 * time.Hour).Seconds())
	}

	result, err := s.client.Eval(ctx, saveNotificationOnceScript,
		[]string{processedKey, "notifications"},
		string(data),
		ttlSeconds,
	).Int()
	if err != nil {
		return false, fmt.Errorf("save notification once script: %w", err)
	}

	return result == 1, nil
}

func (s *Store) Close() error {
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}
