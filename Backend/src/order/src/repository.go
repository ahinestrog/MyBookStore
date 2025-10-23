package main

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite" // driver 100% Go
)

type Repository struct {
	db *sql.DB
}

func NewRepository(dbPath string) (*Repository, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(ON)", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if err := migrate(db); err != nil {
		return nil, err
	}
	return &Repository{db: db}, nil
}

func migrate(db *sql.DB) error {
	schema := `
CREATE TABLE IF NOT EXISTS orders(
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  status INTEGER NOT NULL,
  total_cents INTEGER NOT NULL,
  created_unix INTEGER NOT NULL,
  updated_unix INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS order_items(
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  order_id INTEGER NOT NULL,
  book_id INTEGER NOT NULL,
  title TEXT NOT NULL,
  qty INTEGER NOT NULL,
  unit_cents INTEGER NOT NULL,
  line_cents INTEGER NOT NULL,
  FOREIGN KEY(order_id) REFERENCES orders(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_orders_user ON orders(user_id);
CREATE INDEX IF NOT EXISTS idx_items_order ON order_items(order_id);
`
	_, err := db.Exec(schema)
	return err
}

func (r *Repository) Close() error { return r.db.Close() }

func (r *Repository) CreateOrder(ctx context.Context, o *Order) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil { return 0, err }
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx, `
  INSERT INTO orders(user_id, status, total_cents, created_unix, updated_unix)
  VALUES(?,?,?,?,?)`,
		o.UserID, o.Status, o.TotalCents, o.CreatedUnix, o.UpdatedUnix)
	if err != nil { return 0, err }

	oid, err := res.LastInsertId()
	if err != nil { return 0, err }

	stmt, err := tx.PrepareContext(ctx, `
  INSERT INTO order_items(order_id, book_id, title, qty, unit_cents, line_cents)
  VALUES(?,?,?,?,?,?)`)
	if err != nil { return 0, err }
	defer stmt.Close()

	for _, it := range o.Items {
		if _, err := stmt.ExecContext(ctx,
			oid, it.BookID, it.Title, it.Qty, it.UnitCents, it.LineCents); err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(); err != nil { return 0, err }
	return oid, nil
}

func (r *Repository) UpdateStatus(ctx context.Context, orderID int64, status int32) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE orders SET status=?, updated_unix=? WHERE id=?`,
		status, nowUnix(), orderID)
	return err
}

func (r *Repository) GetOrder(ctx context.Context, orderID int64) (*Order, error) {
	row := r.db.QueryRowContext(ctx, `
    SELECT id, user_id, status, total_cents, created_unix, updated_unix
    FROM orders WHERE id=?`, orderID)
	var o Order
	if err := row.Scan(&o.ID, &o.UserID, &o.Status, &o.TotalCents, &o.CreatedUnix, &o.UpdatedUnix); err != nil {
		return nil, err
	}
	items, err := r.listItems(ctx, orderID)
	if err != nil { return nil, err }
	o.Items = items
	return &o, nil
}

func (r *Repository) listItems(ctx context.Context, orderID int64) ([]OrderItem, error) {
	rows, err := r.db.QueryContext(ctx, `
    SELECT id, order_id, book_id, title, qty, unit_cents, line_cents
    FROM order_items WHERE order_id=?`, orderID)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []OrderItem
	for rows.Next() {
		var it OrderItem
		if err := rows.Scan(&it.ID, &it.OrderID, &it.BookID, &it.Title, &it.Qty, &it.UnitCents, &it.LineCents); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}
