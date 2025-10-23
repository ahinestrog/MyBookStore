package main

import (
	"context"
	"database/sql"
	_ "modernc.org/sqlite"
	"time"
)

type Repository struct {
	DB *sql.DB
}

func NewRepository(dbPath string) (*Repository, error) {
	// _pragma busy_timeout para evitar "database is locked" -> No puede pasar, de lo contrario no nos dejería ingresar en los datos
	dsn := dbPath + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout=5000"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetConnMaxIdleTime(2 * time.Minute)
	db.SetMaxOpenConns(1) 

	r := &Repository{DB: db}
	if err := r.migrate(context.Background()); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Repository) migrate(ctx context.Context) error {
	schema := `
CREATE TABLE IF NOT EXISTS stock(
  book_id      INTEGER PRIMARY KEY,
  total_qty    INTEGER NOT NULL DEFAULT 0,
  reserved_qty INTEGER NOT NULL DEFAULT 0,
  updated_at   INTEGER NOT NULL DEFAULT (strftime('%s','now'))
);
CREATE INDEX IF NOT EXISTS idx_stock_updated ON stock(updated_at);
`
	_, err := r.DB.ExecContext(ctx, schema)
	return err
}

func (r *Repository) Close() error { return r.DB.Close() }

// seed inicial opcional (para pruebas)
func (r *Repository) Seed(ctx context.Context) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil { return err }
	defer tx.Rollback()

	stmt := `
INSERT INTO stock(book_id,total_qty,reserved_qty,updated_at)
VALUES(?,?,?,strftime('%s','now'))
ON CONFLICT(book_id) DO NOTHING;
`
	inserts := [][]any{
		{1, 10, 0},
		{2,  5, 0},
		{3,  0, 0},
		{4, 20, 0},
		{5,  1, 0},
	}
	for _, v := range inserts {
		if _, err := tx.ExecContext(ctx, stmt, v...); err != nil { return err }
	}
	return tx.Commit()
}

func (r *Repository) GetAvailability(ctx context.Context, bookIDs []int64) (map[int64]int32, error) {
	if len(bookIDs) == 0 {
		rows, err := r.DB.QueryContext(ctx, `SELECT book_id,total_qty,reserved_qty FROM stock`)
		if err != nil { return nil, err }
		defer rows.Close()
		out := map[int64]int32{}
		for rows.Next() {
			var id int64; var tot, res int32
			if err := rows.Scan(&id, &tot, &res); err != nil { return nil, err }
			avail := tot - res
			if avail < 0 { avail = 0 }
			out[id] = avail
		}
		return out, rows.Err()
	}

	q := `SELECT book_id,total_qty,reserved_qty FROM stock WHERE book_id IN (` +
		placeholders(len(bookIDs)) + `)`
	args := toAny(bookIDs)
	rows, err := r.DB.QueryContext(ctx, q, args...)
	if err != nil { return nil, err }
	defer rows.Close()

	out := map[int64]int32{}
	for rows.Next() {
		var id int64; var tot, res int32
		if err := rows.Scan(&id, &tot, &res); err != nil { return nil, err }
		avail := tot - res
		if avail < 0 { avail = 0 }
		out[id] = avail
	}
	return out, rows.Err()
}

type OrderItem struct {
	BookID int64 `json:"book_id"`
	Qty    int32 `json:"qty"`
}

func (r *Repository) TryReserve(ctx context.Context, items []OrderItem) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil { return err }
	defer tx.Rollback()

	// Valida disponibilidad
	for _, it := range items {
		var tot, res int32
		err := tx.QueryRowContext(ctx,
			`SELECT total_qty,reserved_qty FROM stock WHERE book_id=?`, it.BookID).
			Scan(&tot, &res)
		if err == sql.ErrNoRows {
			return ErrNoStockForBook{BookID: it.BookID}
		}
		if err != nil { return err }
		avail := tot - res
		if avail < it.Qty {
			return ErrInsufficient{BookID: it.BookID, Need: it.Qty, Avail: avail}
		}
	}

	// Aplicar reservas
	for _, it := range items {
		if _, err := tx.ExecContext(ctx,
			`UPDATE stock SET reserved_qty=reserved_qty+?, updated_at=strftime('%s','now') WHERE book_id=?`,
			it.Qty, it.BookID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *Repository) Confirm(ctx context.Context, items []OrderItem) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil { return err }
	defer tx.Rollback()

	for _, it := range items {
		// Descuenta del total y libera de reserved (cantidad reservada)
		_, err := tx.ExecContext(ctx, `
UPDATE stock
SET total_qty = total_qty - ?,
    reserved_qty = reserved_qty - ?,
    updated_at = strftime('%s','now')
WHERE book_id=?`, it.Qty, it.Qty, it.BookID)
		if err != nil { return err }
	}
	return tx.Commit()
}

func (r *Repository) Release(ctx context.Context, items []OrderItem) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil { return err }
	defer tx.Rollback()

	for _, it := range items {
		_, err := tx.ExecContext(ctx, `
UPDATE stock
SET reserved_qty = CASE
    WHEN reserved_qty >= ? THEN reserved_qty - ?
    ELSE 0 END,
    updated_at = strftime('%s','now')
WHERE book_id=?`, it.Qty, it.Qty, it.BookID)
		if err != nil { return err }
	}
	return tx.Commit()
}

// helpers
func placeholders(n int) string {
	s := "?"
	for i := 1; i < n; i++ { s += ",?" }
	return s
}
func toAny[T any](xs []T) []any {
	out := make([]any, len(xs))
	for i, v := range xs { out[i] = v }
	return out
}

type ErrNoStockForBook struct{ BookID int64 }
func (e ErrNoStockForBook) Error() string { return "stock inexistente para book_id" }

type ErrInsufficient struct{
	BookID int64; Need, Avail int32
}
func (e ErrInsufficient) Error() string { return "stock insuficiente" }
