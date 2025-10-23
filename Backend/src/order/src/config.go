package main

import (
	"log"
	"os"
)

type Config struct {
	GRPCAddr       string
	DBPath         string
	RabbitURL      string
	RabbitExchange string
	CartGRPCAddr   string
}

func LoadConfig() *Config {
	cfg := &Config{
		GRPCAddr:       getEnv("ORDER_GRPC_ADDR", ":50053"),
		DBPath:         getEnv("ORDER_DB_PATH", "./order.db"),
		RabbitURL:      getEnv("RABBIT_URL", "amqp://guest:guest@localhost:5672/"),
		RabbitExchange: getEnv("RABBIT_EXCHANGE", "domain_events"),
		CartGRPCAddr:   getEnv("CART_GRPC_ADDR", "localhost:50051"),
	}
	log.Printf("[order] config: %+v", *cfg)
	return cfg
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

