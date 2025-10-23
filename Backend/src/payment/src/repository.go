package main

import (
	"context"
	"database/sql"
	"errors"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Repository interface {
	Init(ctx context.Context) error
	UpsertPending(ctx context.Context, p Payment) error
	SetResult(ctx context.Context, orderID int64, state PaymentState, providerRef string) error
	GetByOrderID(ctx context.Context, orderID int64) (*Payment, error)
}

type sqliteRepo struct{ db *sql.DB }

func newSQLiteRepo(path string) (Repository, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	return &sqliteRepo{db: db}, nil
}

func (r *sqliteRepo) Init(ctx context.Context) error {
	ddl := `
CREATE TABLE IF NOT EXISTS payments(
  order_id INTEGER PRIMARY KEY,
  amount_cents INTEGER NOT NULL,
  state INTEGER NOT NULL,
  provider_ref TEXT,
  updated_unix INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_payments_state ON payments(state);
`
	_, err := r.db.ExecContext(ctx, ddl)
	return err
}

func (r *sqliteRepo) UpsertPending(ctx context.Context, p Payment) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO payments(order_id, amount_cents, state, provider_ref, updated_unix)
VALUES(?,?,?,?,?)
ON CONFLICT(order_id) DO UPDATE SET
  amount_cents=excluded.amount_cents,
  state=?,
  updated_unix=?;
`, p.OrderID, p.AmountCents, PaymentStatePending, "", time.Now().Unix(),
		PaymentStatePending, time.Now().Unix())
	return err
}

func (r *sqliteRepo) SetResult(ctx context.Context, orderID int64, state PaymentState, providerRef string) error {
	_, err := r.db.ExecContext(ctx, `
UPDATE payments SET state=?, provider_ref=?, updated_unix=? WHERE order_id=?;
`, state, providerRef, time.Now().Unix(), orderID)
	return err
}

func (r *sqliteRepo) GetByOrderID(ctx context.Context, orderID int64) (*Payment, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT order_id, amount_cents, state, provider_ref, updated_unix FROM payments WHERE order_id=?;
`, orderID)
	var p Payment
	var updated int64
	if err := row.Scan(&p.OrderID, &p.AmountCents, &p.State, &p.ProviderRef, &updated); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	p.UpdatedAt = time.Unix(updated, 0)
	return &p, nil
}

