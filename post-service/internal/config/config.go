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
	Env         string `yaml:"env" env-default:"local"`
	DatabaseURL string `env:"DATABASE_URL,required"`
	HTTPServer  `yaml:"http_server"`
	Kafka       `yaml:"kafka"`
	Redis       `yaml:"redis"`
	Outbox      `yaml:"outbox"`
}

type HTTPServer struct {
	Address     string        `yaml:"address" env-default:"localhost:8080"`
	Timeout     time.Duration `yaml:"timeout" env-default:"4s"`
	IdleTimeout time.Duration `yaml:"idle_timeout" env-default:"60s"`
}

type Kafka struct {
	Brokers []string `yaml:"brokers" env:"KAFKA_BROKERS" env-default:"localhost:9092"`
	Topic   string   `yaml:"topic" env:"KAFKA_POSTS_TOPIC" env-default:"posts"`
	GroupID string   `yaml:"group_id" env-default:"notification-service"`
}

type Redis struct {
	Addr string `yaml:"address" env:"REDIS_ADDR" env-default:"localhost:6379"`
	DB   int    `yaml:"db" env:"REDIS_DB" env-default:"0"`
}

type Outbox struct {
	PollInterval time.Duration `yaml:"poll_interval" env:"OUTBOX_POLL_INTERVAL" env-default:"1s"`
	RetryDelay   time.Duration `yaml:"retry_delay" env:"OUTBOX_RETRY_DELAY" env-default:"5s"`
	MaxAttempts  int           `yaml:"max_attempts" env:"OUTBOX_MAX_ATTEMPTS" env-default:"5"`
	BatchSize    int           `yaml:"batch_size" env:"OUTBOX_BATCH_SIZE" env-default:"10"`
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

	if cfg.DatabaseURL = os.Getenv("DATABASE_URL"); cfg.DatabaseURL == "" {
		panic("DATABASE_URL not set in environment")
	}

	fmt.Printf("✅ Loaded config for environment: %s\n", env)
	return &cfg
}
