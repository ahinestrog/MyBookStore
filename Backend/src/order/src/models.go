package main

import "time"

// Estados presentes con la firma del order.proto
const (
	OrderStatusUnspecified = 0
	OrderStatusCreated     = 1
	OrderStatusPaid        = 2
	OrderStatusCancelled   = 3
	OrderStatusFailed      = 4
)

type Order struct {
	ID          int64     `db:"id"`
	UserID      int64     `db:"user_id"`
	Status      int32     `db:"status"`
	TotalCents  int64     `db:"total_cents"`
	CreatedUnix int64     `db:"created_unix"`
	UpdatedUnix int64     `db:"updated_unix"`
	Items       []OrderItem
}

type OrderItem struct {
	ID         int64 `db:"id"`
	OrderID    int64 `db:"order_id"`
	BookID     int64 `db:"book_id"`
	Title      string `db:"title"`
	Qty        int32  `db:"qty"`
	UnitCents  int64  `db:"unit_cents"`
	LineCents  int64  `db:"line_cents"`
}

func nowUnix() int64 { return time.Now().Unix() }
