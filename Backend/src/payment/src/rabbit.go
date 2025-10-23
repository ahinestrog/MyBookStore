package main

import (
	"context"
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type broker struct {
	cfg Config
	ch  *amqp.Channel
	conn *amqp.Connection
}

func newBroker(cfg Config) *broker { return &broker{cfg: cfg} }

func (b *broker) connect() error {
	conn, err := amqp.Dial(b.cfg.RabbitURL)
	if err != nil {
		return err
	}
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	if err := ch.ExchangeDeclare(
		b.cfg.ExchangeName, "topic", true, false, false, false, nil,
	); err != nil {
		return err
	}
	b.conn = conn
	b.ch = ch
	return nil
}

func (b *broker) close() {
	if b.ch != nil { _ = b.ch.Close() }
	if b.conn != nil { _ = b.conn.Close() }
}

func (b *broker) consumePaymentRequested(ctx context.Context, handler func(context.Context, []byte) error, consumerTag string, prefetch int) error {
	q, err := b.ch.QueueDeclare(b.cfg.RequestQueue, true, false, false, false, nil)
	if err != nil {
		return err
	}
	// Bind a topic key; puedes publicar con routing key "payment.charge.requested"
	if err := b.ch.QueueBind(q.Name, "payment.charge.requested", b.cfg.ExchangeName, false, nil); err != nil {
		return err
	}
	if err := b.ch.Qos(prefetch, 0, false); err != nil {
		return err
	}
	deliveries, err := b.ch.Consume(q.Name, consumerTag, false, false, false, false, nil)
	if err != nil {
		return err
	}

	go func() {
		for d := range deliveries {
			if err := handler(ctx, d.Body); err != nil {
				log.Printf("[payment] handler error: %v", err)
				_ = d.Nack(false, true) // lo encola de nuevo
				continue
			}
			_ = d.Ack(false)
		}
		log.Printf("[payment] deliveries channel closed")
	}()
	return nil
}

func (b *broker) publishJSON(exchangeKey string, v any) {
	body, _ := json.Marshal(v)
	err := b.ch.Publish(b.cfg.ExchangeName, exchangeKey, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	})
	if err != nil {
		log.Printf("[payment] publish error (%s): %v", exchangeKey, err)
	}
}
