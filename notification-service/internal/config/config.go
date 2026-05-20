package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Config struct {
	Env   string `yaml:"env" env-default:"local"`
	Kafka Kafka  `yaml:"kafka"`
	Redis Redis  `yaml:"redis"`
	Retry Retry  `yaml:"retry"`
}

type Kafka struct {
	Brokers  []string `yaml:"brokers" env:"KAFKA_BROKERS" env-default:"localhost:9092"`
	Topic    string   `yaml:"topic" env:"KAFKA_POSTS_TOPIC" env-default:"posts"`
	DLQTopic string   `yaml:"dlq_topic" env:"KAFKA_DLQ_TOPIC" env-default:"posts.dlq"`
	GroupID  string   `yaml:"group_id" env:"KAFKA_CONSUMER_GROUP" env-default:"notification-service"`
}

type Redis struct {
	Addr         string        `yaml:"address" env:"REDIS_ADDR" env-default:"localhost:6379"`
	Password     string        `yaml:"password" env:"REDIS_PASSWORD" env-default:""`
	DB           int           `yaml:"db" env:"REDIS_DB" env-default:"0"`
	ProcessedTTL time.Duration `yaml:"processed_ttl" env:"PROCESSED_EVENTS_TTL" env-default:"168h"`
}

type Retry struct {
	MaxAttempts int           `yaml:"max_attempts" env:"MAX_RETRY_ATTEMPTS" env-default:"3"`
	Backoff     time.Duration `yaml:"backoff" env:"RETRY_BACKOFF" env-default:"1s"`
}

func MustLoad() *Config {
	_ = godotenv.Load()

	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "local"
	}

	configPath := filepath.Join("config", fmt.Sprintf("%s.yaml", env))

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		panic(fmt.Sprintf("config file not found: %s", configPath))
	}

	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		panic("cannot read config: " + err.Error())
	}

	fmt.Printf("✅ Loaded config for environment: %s\n", env)
	return &cfg
}
