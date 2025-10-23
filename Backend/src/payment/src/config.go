package main

import (
	"log"
	"os"
	"time"
)

type Config struct {
	ServicePort   string // gRPC
	DBPath        string
	RabbitURL     string
	ExchangeName  string
	RequestQueue  string
	ConsumerTag   string
	PrefetchCount int
}

func loadConfig() Config {
	cfg := Config{
		ServicePort:   getEnv("PAYMENT_GRPC_PORT", "50053"),
		DBPath:        getEnv("PAYMENT_SQLITE_PATH", "/data/payment.db"),
		RabbitURL:     getEnv("RABBITMQ_URL", "amqp://guest:guest@rabbitmq:5672/"),
		ExchangeName:  getEnv("EVENTS_EXCHANGE", "mybookstore.events"),
		RequestQueue:  getEnv("PAYMENT_REQUEST_QUEUE", "payment.charge.requested"),
		ConsumerTag:   getEnv("PAYMENT_CONSUMER_TAG", "payment-service"),
		PrefetchCount: 10,
	}
	return cfg
}

func must[T any](v T, err error) T {
	if err != nil {
		log.Fatalf("fatal: %v", err)
	}
	return v
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func nowUnix() int64 { return time.Now().Unix() }

