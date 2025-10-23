package main

type Cart struct {
	ID     int64
	UserID int64
	Items  []CartItem
}

type CartItem struct {
	ID             int64
	CartID         int64
	BookID         int64
	Title          string
	UnitPriceCents int64
	Qty            int32
}

type Money struct{ Cents int64 }

func (m Money) Add(o Money) Money  { return Money{Cents: m.Cents + o.Cents} }
func (m Money) Mul(qty int32) Money { return Money{Cents: m.Cents * int64(qty)} }
