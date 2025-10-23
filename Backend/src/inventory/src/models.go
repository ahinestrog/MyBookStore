package main

// Inventario:
// total_qty: stock físico disponible en bodega
// reserved_qty: unidades reservadas temporalmente (pendientes de confirmación)
// available = total_qty - reserved_qty
type Stock struct {
	BookID      int64 `db:"book_id"`
	TotalQty    int32 `db:"total_qty"`
	ReservedQty int32 `db:"reserved_qty"`
}
