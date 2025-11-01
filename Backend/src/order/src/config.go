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
	// Inventory queue names (direct queues)
	QReserveReq string
	QReserveRes string
	QConfirmReq string
	QReleaseReq string
}

func LoadConfig() *Config {
	cfg := &Config{
		GRPCAddr: getEnv("ORDER_GRPC_ADDR", ":50053"),
		DBPath:   getEnv("ORDER_DB_PATH", "./order.db"),
		// Prefer RABBITMQ_URL (used by other services), fallback to RABBIT_URL
		RabbitURL: firstNonEmpty(os.Getenv("RABBITMQ_URL"), os.Getenv("RABBIT_URL"), "amqp://guest:guest@localhost:5672/"),
		// Align with Payment default exchange
		RabbitExchange: getEnv("EVENTS_EXCHANGE", "mybookstore.events"),
		CartGRPCAddr:   getEnv("CART_GRPC_ADDR", "localhost:50051"),
		// Inventory queues (match Inventory service defaults)
		QReserveReq: getEnv("Q_INVENTORY_RESERVE_REQUEST", "inventory.reserve.request"),
		QReserveRes: getEnv("Q_INVENTORY_RESERVE_RESULT", "inventory.reserve.result"),
		QConfirmReq: getEnv("Q_INVENTORY_CONFIRM_REQUEST", "inventory.confirm.request"),
		QReleaseReq: getEnv("Q_INVENTORY_RELEASE_REQUEST", "inventory.release.request"),
	}
	log.Printf("[order] config: addr=%s rabbit=%s exchange=%s cart=%s queues=[req=%s res=%s conf=%s rel=%s]",
		cfg.GRPCAddr, cfg.RabbitURL, cfg.RabbitExchange, cfg.CartGRPCAddr,
		cfg.QReserveReq, cfg.QReserveRes, cfg.QConfirmReq, cfg.QReleaseReq)
	return cfg
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// firstNonEmpty returns the first non-empty string, otherwise def
func firstNonEmpty(v1, v2, def string) string {
	if v1 != "" {
		return v1
	}
	if v2 != "" {
		return v2
	}
	return def
}
