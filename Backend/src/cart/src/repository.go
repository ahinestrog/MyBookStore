// Operaciones de carrito
package main

import (
	"context"
	"database/sql"
	"errors"
)

var ErrNotFound = errors.New("not found")

type CartRepository interface {
	GetOrCreateCart(ctx context.Context, userID int64) (*Cart, error)
	GetCart(ctx context.Context, userID int64) (*Cart, error)
	AddItem(ctx context.Context, userID, bookID int64, title string, unitPriceCents int64, qty int32) (*Cart, error)
	RemoveItem(ctx context.Context, userID, bookID int64, qty int32) (*Cart, error) // qty==0 => borra línea
	Clear(ctx context.Context, userID int64) (*Cart, error)
}

type sqliteRepo struct{ db *sql.DB }

func NewSQLiteRepo(db *sql.DB) CartRepository { return &sqliteRepo{db: db} }

func (r *sqliteRepo) GetOrCreateCart(ctx context.Context, userID int64) (*Cart, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil { return nil, err }
	defer func() { _ = tx.Rollback() }()

	var cartID int64
	err = tx.QueryRowContext(ctx, `SELECT id FROM carts WHERE user_id=?`, userID).Scan(&cartID)
	if err == sql.ErrNoRows {
		res, err := tx.ExecContext(ctx, `INSERT INTO carts(user_id) VALUES (?)`, userID)
		if err != nil { return nil, err }
		cartID, err = res.LastInsertId()
		if err != nil { return nil, err }
	} else if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil { return nil, err }
	return r.GetCart(ctx, userID)
}

func (r *sqliteRepo) GetCart(ctx context.Context, userID int64) (*Cart, error) {
	var cart Cart
	err := r.db.QueryRowContext(ctx, `SELECT id, user_id FROM carts WHERE user_id=?`, userID).
		Scan(&cart.ID, &cart.UserID)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil { return nil, err }

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, cart_id, book_id, title, unit_price_cents, qty
		FROM cart_items WHERE cart_id=?`, cart.ID)
	if err != nil { return nil, err }
	defer rows.Close()

	for rows.Next() {
		var it CartItem
		if err := rows.Scan(&it.ID, &it.CartID, &it.BookID, &it.Title, &it.UnitPriceCents, &it.Qty); err != nil {
			return nil, err
		}
		cart.Items = append(cart.Items, it)
	}
	return &cart, nil
}

func (r *sqliteRepo) AddItem(ctx context.Context, userID, bookID int64, title string, unitPriceCents int64, qty int32) (*Cart, error) {
	cart, err := r.GetOrCreateCart(ctx, userID)
	if err != nil { return nil, err }

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO cart_items(cart_id, book_id, title, unit_price_cents, qty)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(cart_id, book_id)
		DO UPDATE SET qty = qty + excluded.qty
	`, cart.ID, bookID, title, unitPriceCents, qty)
	if err != nil { return nil, err }

	return r.GetCart(ctx, userID)
}

func (r *sqliteRepo) RemoveItem(ctx context.Context, userID, bookID int64, qty int32) (*Cart, error) {
	cart, err := r.GetCart(ctx, userID)
	if err != nil { return nil, err }

	if qty <= 0 {
		// eliminar línea
		_, err = r.db.ExecContext(ctx, `DELETE FROM cart_items WHERE cart_id=? AND book_id=?`, cart.ID, bookID)
	} else {
		// decrementar; si llega a 0 o menos borra
		_, err = r.db.ExecContext(ctx, `
			UPDATE cart_items SET qty = qty - ?
			WHERE cart_id=? AND book_id=?`, qty, cart.ID, bookID)
		if err == nil {
			_, _ = r.db.ExecContext(ctx, `
				DELETE FROM cart_items WHERE cart_id=? AND book_id=? AND qty <= 0`, cart.ID, bookID)
		}
	}
	if err != nil { return nil, err }
	return r.GetCart(ctx, userID)
}

func (r *sqliteRepo) Clear(ctx context.Context, userID int64) (*Cart, error) {
	cart, err := r.GetCart(ctx, userID)
	if err != nil { return nil, err }
	_, err = r.db.ExecContext(ctx, `DELETE FROM cart_items WHERE cart_id=?`, cart.ID)
	if err != nil { return nil, err }
	return r.GetCart(ctx, userID)
}
