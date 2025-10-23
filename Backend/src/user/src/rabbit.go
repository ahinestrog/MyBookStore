package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type EventPublisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewEventPublisher(rabbitURL string) (*EventPublisher, error) {
	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	if err := ch.ExchangeDeclare("user.events", "topic", true, false, false, false, nil); err != nil {
		return nil, err
	}
	return &EventPublisher{conn: conn, channel: ch}, nil
}

func (p *EventPublisher) Close() {
	if p.channel != nil {
		_ = p.channel.Close()
	}
	if p.conn != nil {
		_ = p.conn.Close()
	}
}

func (p *EventPublisher) Publish(eventType string, payload any) error {
	body, err := json.Marshal(struct {
		Type      string    `json:"type"`
		Timestamp time.Time `json:"timestamp"`
		Payload   any       `json:"payload"`
	}{
		Type:      eventType,
		Timestamp: time.Now().UTC(),
		Payload:   payload,
	})
	if err != nil {
		return err
	}
	log.Printf("[user] publish event %s", eventType)
	return p.channel.PublishWithContext(context.Background(),
		"user.events", eventType, false, false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})
}
