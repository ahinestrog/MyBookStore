package main

import "time"

type PaymentState int

const (
	PaymentStateUnspecified PaymentState = iota
	PaymentStatePending
	PaymentStateSucceeded
	PaymentStateFailed
)

func (s PaymentState) String() string {
	switch s {
	case PaymentStatePending:
		return "PENDING"
	case PaymentStateSucceeded:
		return "SUCCEEDED"
	case PaymentStateFailed:
		return "FAILED"
	default:
		return "UNSPECIFIED"
	}
}

type Payment struct {
	OrderID     int64
	AmountCents int64
	State       PaymentState
	ProviderRef string
	UpdatedAt   time.Time
}
