package main

// Eventos enviados por Order a otros servicios
const (
	RKOrderCreated        = "order.created"
	RKPaymentCharge       = "payment.charge" // publicado por Order cuando inventario confirma
)

// Eventos recibidos por Order
const (
	RKInventoryReserved   = "inventory.reserved"
	RKInventoryRejected   = "inventory.rejected"
	RKPaymentCaptured     = "payment.captured"
	RKPaymentFailed       = "payment.failed"
)

type OrderCreatedPayload struct {
	OrderID    int64          `json:"order_id"`
	UserID     int64          `json:"user_id"`
	Items      []OrderItemEvt `json:"items"`
	TotalCents int64          `json:"total_cents"`
}

type OrderItemEvt struct {
	BookID    int64  `json:"book_id"`
	Title     string `json:"title"`
	Qty       int32  `json:"qty"`
	UnitCents int64  `json:"unit_cents"`
	LineCents int64  `json:"line_cents"`
}

type InventoryResultPayload struct {
	OrderID int64  `json:"order_id"`
	OK      bool   `json:"ok"`
	Reason  string `json:"reason,omitempty"`
}

type PaymentChargePayload struct {
	OrderID    int64  `json:"order_id"`
	UserID     int64  `json:"user_id"`
	TotalCents int64  `json:"total_cents"`
}

type PaymentResultPayload struct {
	OrderID int64  `json:"order_id"`
	OK      bool   `json:"ok"`
	Reason  string `json:"reason,omitempty"`
}

