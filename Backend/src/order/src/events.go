package main

const (
	RKOrderCreated           = "order.created"
	RKPaymentChargeRequested = "payment.charge.requested"
	RKPaymentSucceeded       = "payment.succeeded"
	RKPaymentFailed          = "payment.failed"
)

type InvOrderItem struct {
	BookID int64 `json:"book_id"`
	Qty    int32 `json:"qty"`
}

type InvReserveRequest struct {
	OrderID int64          `json:"order_id"`
	UserID  int64          `json:"user_id"`
	Items   []InvOrderItem `json:"items"`
}

type InvReserveResult struct {
	OrderID int64  `json:"order_id"`
	State   string `json:"state"` // "RESERVED" | "FAILED"
	Reason  string `json:"reason,omitempty"`
}

type InvConfirmRequest struct {
	OrderID int64          `json:"order_id"`
	Items   []InvOrderItem `json:"items"`
}

type InvReleaseRequest struct {
	OrderID int64          `json:"order_id"`
	Items   []InvOrderItem `json:"items"`
}


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

type PaymentRequested struct {
	OrderID     int64 `json:"order_id"`
	UserID      int64 `json:"user_id"`
	AmountCents int64 `json:"amount_cents"`
}

type PaymentSucceeded struct {
	OrderID     int64  `json:"order_id"`
	ProviderRef string `json:"provider_ref"`
}

type PaymentFailed struct {
	OrderID     int64  `json:"order_id"`
	Reason      string `json:"reason"`
	ProviderRef string `json:"provider_ref"`
}
