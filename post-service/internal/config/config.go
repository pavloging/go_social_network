package config

import (
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	// "github.com/joho/godotenv"
)

type Config struct {
	Env         string `yaml:"env" env-default:"local"`
	DatabaseURL string `env:"DATABASE_URL,required"`
	HTTPServer  `yaml:"http_server"`
	Kafka       `yaml:"kafka"`
	Redis       `yaml:"redis"`
}

type HTTPServer struct {
	Address     string        `yaml:"address" env-default:"localhost:8080"`
	Timeout     time.Duration `yaml:"timeout" env-default:"4s"`
	IdleTimeout time.Duration `yaml:"idle_timeout" env-default:"60s"`
}

type Kafka struct {
	Brokers []string `yaml:"brokers" env-default:"localhost:9092"`
	Topic   string   `yaml:"topic" env-default:"posts"`
	GroupID string   `yaml:"group_id" env-default:"notification-service"`
}

type Redis struct {
	Addr string `yaml:"address" env-default:"localhost:6379"`
	DB   int    `yaml:"db" env-default:"0"`
}

const configPath = "./config/prod.yaml"

func MustLoad() *Config {
	// _ = godotenv.Load()
	// if err != nil {
	// 	panic("error loading .env file")
	// }

	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		panic("cannot read config " + err.Error())
	}

	if cfg.DatabaseURL = os.Getenv("DATABASE_URL"); cfg.DatabaseURL == "" {
		panic("error loading DATABASE_URL from environment, please set it in .env file")
	}

	return &cfg
}
