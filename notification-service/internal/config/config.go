package config

import (
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env        string `yaml:"env" env-default:"local"`
	HTTPServer `yaml:"http_server"`
	Kafka      `yaml:"kafka"`
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

const configPath = "./config/prod.yaml"

func MustLoad() *Config {
	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		panic("cannot read config " + err.Error())
	}
	return &cfg
}
