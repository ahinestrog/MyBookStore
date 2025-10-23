package main

import (
	"log"
	"os"
)

type Config struct {
	GRPCPort   string
	DBPath     string
	RabbitURL  string
	ServiceEnv string
}

func LoadConfig() Config {
	cfg := Config{
		GRPCPort:   env("USER_GRPC_PORT", "50055"),
		DBPath:     env("USER_DB_PATH", "/data/user.db"),
		RabbitURL:  env("RABBITMQ_URL", "amqp://guest:guest@rabbitmq:5672/"),
		ServiceEnv: env("SERVICE_ENV", "dev"),
	}
	log.Printf("[user] config loaded: %+v", cfg)
	return cfg
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

