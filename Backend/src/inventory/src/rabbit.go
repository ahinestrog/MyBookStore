package main

import (
	"context"
	"encoding/json"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/streadway/amqp"
)

type Rabbit struct {
	cfg  Config
	conn *amqp.Connection
	ch   *amqp.Channel
	repo *Repository
}

func NewRabbit(cfg Config, repo *Repository) (*Rabbit, error) {
	conn, err := amqp.Dial(cfg.RabbitURL)
	if err != nil { return nil, err }
	ch, err := conn.Channel()
	if err != nil { conn.Close(); return nil, err }

	r := &Rabbit{cfg: cfg, conn: conn, ch: ch, repo: repo}
	qNames := []string{cfg.QReserveReq, cfg.QReserveRes, cfg.QConfirmReq, cfg.QReleaseReq}
	for _, q := range qNames {
		if _, err := ch.QueueDeclare(q, true, false, false, false, nil); err != nil {
			r.Close()
			return nil, err
		}
	}
	return r, nil
}

func (r *Rabbit) Close() {
	if r.ch != nil { _ = r.ch.Close() }
	if r.conn != nil { _ = r.conn.Close() }
}

// Mensajes
type ReserveRequest struct {
	OrderID int64       `json:"order_id"`
	UserID  int64       `json:"user_id"`
	Items   []OrderItem `json:"items"`
}
type ReserveResult struct {
	OrderID int64  `json:"order_id"`
	State   string `json:"state"`
	Reason  string `json:"reason,omitempty"`
}

type ConfirmRequest struct {
	OrderID int64       `json:"order_id"`
	Items   []OrderItem `json:"items"`
}
type ReleaseRequest struct {
	OrderID int64       `json:"order_id"`
	Items   []OrderItem `json:"items"`
}

// Helpers para los publicadores
func (r *Rabbit) publishJSON(q string, v any) error {
	body, _ := json.Marshal(v)
	return r.ch.Publish("", q, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	})
}

// Consumidores RabbitMQ
func (r *Rabbit) StartConsumers(ctx context.Context) error {
	// reserve.request
	if err := r.consumeReserve(ctx); err != nil { return err }
	if err := r.consumeConfirm(ctx); err != nil { return err }
	if err := r.consumeRelease(ctx); err != nil { return err }
	return nil
}

func (r *Rabbit) consumeReserve(ctx context.Context) error {
	msgs, err := r.ch.Consume(r.cfg.QReserveReq, "inventory-reserve-worker", false, false, false, false, nil)
	if err != nil { return err }

	go func() {
		for m := range msgs {
			var req ReserveRequest
			if err := json.Unmarshal(m.Body, &req); err != nil {
				log.Error().Err(err).Msg("reserve: invalid json")
				_ = m.Ack(false)
				continue
			}
			log.Info().Int64("order", req.OrderID).Msg("reserve: received")
			res := ReserveResult{OrderID: req.OrderID}

			// Intentar reservar
			if err := r.repo.TryReserve(ctx, req.Items); err != nil {
				res.State = "FAILED"
				res.Reason = err.Error()
			} else {
				res.State = "RESERVED"
			}
			if err := r.publishJSON(r.cfg.QReserveRes, res); err != nil {
				log.Error().Err(err).Msg("reserve: publish result failed")
			}
			_ = m.Ack(false)
		}
	}()
	return nil
}

func (r *Rabbit) consumeConfirm(ctx context.Context) error {
	msgs, err := r.ch.Consume(r.cfg.QConfirmReq, "inventory-confirm-worker", false, false, false, false, nil)
	if err != nil { return err }
	go func() {
		for m := range msgs {
			var req ConfirmRequest
			if err := json.Unmarshal(m.Body, &req); err != nil {
				log.Error().Err(err).Msg("confirm: invalid json")
				_ = m.Ack(false); continue
			}
			log.Info().Int64("order", req.OrderID).Msg("confirm: received")
			if err := r.repo.Confirm(ctx, req.Items); err != nil {
				log.Error().Err(err).Msg("confirm: repo error")
			}
			_ = m.Ack(false)
		}
	}()
	return nil
}

func (r *Rabbit) consumeRelease(ctx context.Context) error {
	msgs, err := r.ch.Consume(r.cfg.QReleaseReq, "inventory-release-worker", false, false, false, false, nil)
	if err != nil { return err }
	go func() {
		for m := range msgs {
			var req ReleaseRequest
			if err := json.Unmarshal(m.Body, &req); err != nil {
				log.Error().Err(err).Msg("release: invalid json")
				_ = m.Ack(false); continue
			}
			log.Info().Int64("order", req.OrderID).Msg("release: received")
			if err := r.repo.Release(ctx, req.Items); err != nil {
				log.Error().Err(err).Msg("release: repo error")
			}
			_ = m.Ack(false)
		}
	}()
	return nil
}

// retry para reintentos de publicar, en caso de que se necesite
func retry(n int, sleep time.Duration, fn func() error) error {
	var err error
	for i := 0; i < n; i++ {
		if err = fn(); err == nil { return nil }
		time.Sleep(sleep)
	}
	return err
}
