package main

import (
	"context"
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Rabbit struct {
	conn     *amqp.Connection
	ch       *amqp.Channel
	exchange string
}

func NewRabbit(url, exchange string) (*Rabbit, error) {
	conn, err := amqp.Dial(url)
	if err != nil { return nil, err }
	ch, err := conn.Channel()
	if err != nil { return nil, err }
	if err := ch.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
		return nil, err
	}
	return &Rabbit{conn: conn, ch: ch, exchange: exchange}, nil
}

func (r *Rabbit) Close() {
	if r.ch != nil { _ = r.ch.Close() }
	if r.conn != nil { _ = r.conn.Close() }
}

func (r *Rabbit) PublishJSON(routingKey string, v any) error {
	body, err := json.Marshal(v)
	if err != nil { return err }
	return r.ch.PublishWithContext(context.Background(), r.exchange, routingKey, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	})
}

type ConsumerHandler func(rk string, body []byte) error

func (r *Rabbit) ConsumeTopic(queueName string, bindings []string, handler ConsumerHandler) error {
	q, err := r.ch.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil { return err }
	for _, rk := range bindings {
		if err := r.ch.QueueBind(q.Name, rk, r.exchange, false, nil); err != nil {
			return err
		}
	}
	msgs, err := r.ch.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil { return err }

	go func() {
		for d := range msgs {
			if err := handler(d.RoutingKey, d.Body); err != nil {
				log.Printf("[rabbit] handler error for rk=%s: %v", d.RoutingKey, err)
			}
		}
		log.Printf("[rabbit] consumer %s stopped", queueName)
	}()
	return nil
}

