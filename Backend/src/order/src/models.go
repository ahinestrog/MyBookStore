package main

import "time"

const (
	OrderStatusUnspecified = 0
	OrderStatusCreated     = 1
	OrderStatusPaid        = 2
	OrderStatusCancelled   = 3
	OrderStatusFailed      = 4
)

type Order struct {
	ID          int64     
	UserID      int64     
	Status      int32     
	TotalCents  int64     
	CreatedUnix int64     
	UpdatedUnix int64     
	Items       []OrderItem
}

type OrderItem struct {
	ID         int64 
	OrderID    int64 
	BookID     int64 
	Title      string 
	Qty        int32  
	UnitCents  int64  
	LineCents  int64  
}

func nowUnix() int64 { return time.Now().Unix() }
