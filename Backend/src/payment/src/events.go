package main

import (
	"context"
	"encoding/json"
	"log"
)

// Mensajes que viajan por RabbitMQ (JSON).

// Publicado por Order → consumido por Payment.
type PaymentRequested struct {
	OrderID     int64 `json:"order_id"`
	UserID      int64 `json:"user_id"`
	AmountCents int64 `json:"amount_cents"`
}

// Publicado por Payment → consumido por Order.
type PaymentSucceeded struct {
	OrderID     int64  `json:"order_id"`
	ProviderRef string `json:"provider_ref"`
}

type PaymentFailed struct {
	OrderID     int64  `json:"order_id"`
	Reason      string `json:"reason"`
	ProviderRef string `json:"provider_ref"`
}

func (s *service) handlePaymentRequested(ctx context.Context, body []byte) error {
	var msg PaymentRequested
	if err := json.Unmarshal(body, &msg); err != nil {
		log.Printf("[payment] invalid message: %v", err)
		return nil // NACK infinito no sirve; descartamos.
	}

	if err := s.repo.UpsertPending(ctx, Payment{
		OrderID:     msg.OrderID,
		AmountCents: msg.AmountCents,
		State:       PaymentStatePending,
	}); err != nil {
		return err
	}

	// Ejecutar cobro con la pasarela fake 
	ok, providerRef, failReason := s.provider.Charge(ctx, msg.OrderID, msg.AmountCents)

	if ok {
		if err := s.repo.SetResult(ctx, msg.OrderID, PaymentStateSucceeded, providerRef); err != nil {
			return err
		}
		ev := PaymentSucceeded{OrderID: msg.OrderID, ProviderRef: providerRef}
		s.publishJSON("payment.succeeded", ev)
		log.Printf("[payment] SUCCEEDED order=%d ref=%s", msg.OrderID, providerRef)
	} else {
		if err := s.repo.SetResult(ctx, msg.OrderID, PaymentStateFailed, providerRef); err != nil {
			return err
		}
		ev := PaymentFailed{OrderID: msg.OrderID, Reason: failReason, ProviderRef: providerRef}
		s.publishJSON("payment.failed", ev)
		log.Printf("[payment] FAILED order=%d reason=%s ref=%s", msg.OrderID, failReason, providerRef)
	}
	return nil
}
