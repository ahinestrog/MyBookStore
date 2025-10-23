package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

type PaymentProvider interface {
	Charge(ctx context.Context, orderID, amountCents int64) (ok bool, providerRef, failReason string)
}

type fakeProvider struct{}

func newFakeProvider() PaymentProvider { return &fakeProvider{} }

// Regla determinística simple para pruebas (evita flakes):
// - Si orderID es PAR ⇒ éxito.
// - Si es IMPAR ⇒ 80% éxito y 20% falla por fondos insuficientes.
func (f *fakeProvider) Charge(ctx context.Context, orderID, amountCents int64) (bool, string, string) {
	ref := fmt.Sprintf("FAKE-%d-%d", orderID, time.Now().UnixNano())
	if orderID%2 == 0 {
		return true, ref, ""
	}
	r := rand.New(rand.NewSource(orderID + time.Now().UnixNano()))
	if r.Intn(10) < 8 {
		return true, ref, ""
	}
	return false, ref, "insufficient_funds"
}
