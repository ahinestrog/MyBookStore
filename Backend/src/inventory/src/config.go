package main

import (
	"os"
	"time"
)

type Config struct {
	GRPCAddr     string
	DBPath       string
	RabbitURL    string
	ServiceName  string
	SeedOnStart  bool
	// Nombres de colas
	QReserveReq  string
	QReserveRes  string
	QConfirmReq  string
	QReleaseReq  string
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func LoadConfig() Config {
	return Config{
		ServiceName: getenv("INVENTORY_SERVICE_NAME", "inventory"),
		GRPCAddr:    getenv("INVENTORY_GRPC_ADDR", ":50053"),
		DBPath:      getenv("INVENTORY_DB_PATH", "inventory.db"),
		RabbitURL:   getenv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		SeedOnStart: getenv("INVENTORY_SEED", "true") == "true",

		QReserveReq: getenv("Q_INVENTORY_RESERVE_REQUEST", "inventory.reserve.request"),
		QReserveRes: getenv("Q_INVENTORY_RESERVE_RESULT",  "inventory.reserve.result"),
		QConfirmReq: getenv("Q_INVENTORY_CONFIRM_REQUEST", "inventory.confirm.request"),
		QReleaseReq: getenv("Q_INVENTORY_RELEASE_REQUEST", "inventory.release.request"),
	}
}

const (
	ShutdownGrace = 10 * time.Second
)
