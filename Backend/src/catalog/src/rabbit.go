package main

import (
	"context"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Rabbit struct {
	ch        *amqp.Channel
	exchange  string
}

func NewRabbit(url, exchange string) (*Rabbit, error) {
	if url == "" { return nil, nil }
	conn, err := amqp.Dial(url)
	if err != nil { return nil, err }
	ch, err := conn.Channel()
	if err != nil { return nil, err }
	if err := ch.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
		return nil, err
	}
	return &Rabbit{ch: ch, exchange: exchange}, nil
}

func (r *Rabbit) Publish(ctx context.Context, key string, body []byte) error {
	if r == nil || r.ch == nil { return nil }
	return r.ch.PublishWithContext(ctx, r.exchange, key, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
		Timestamp:   time.Now(),
	})
}
