package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type Repository interface {
	Init(ctx context.Context) error
	Count(ctx context.Context, q string) (int64, error)
	List(ctx context.Context, q string, limit, offset int32) ([]*Book, error)
	Get(ctx context.Context, id int64) (*Book, error)
}

type sqliteRepo struct{ db *sql.DB }

func NewSQLiteRepo(db *sql.DB) Repository { return &sqliteRepo{db: db} }

func (r *sqliteRepo) Init(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, mustRead("src/db/db.sql"))
	return err
}

func (r *sqliteRepo) Count(ctx context.Context, q string) (int64, error) {
	if strings.TrimSpace(q) == "" {
		var c int64
		err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM books`).Scan(&c)
		return c, err
	}
	qp := "%" + strings.ToLower(q) + "%"
	var c int64
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM books
		WHERE lower(title)  LIKE ? OR lower(author) LIKE ?`, qp, qp).Scan(&c)
	return c, err
}

func (r *sqliteRepo) List(ctx context.Context, q string, limit, offset int32) ([]*Book, error) {
	var rows *sql.Rows
	var err error
	if strings.TrimSpace(q) == "" {
		rows, err = r.db.QueryContext(ctx, `
			SELECT id,title,author,price_cents,cover_url,created_unix
			FROM books ORDER BY id DESC LIMIT ? OFFSET ?`, limit, offset)
	} else {
		qp := "%" + strings.ToLower(q) + "%"
		rows, err = r.db.QueryContext(ctx, `
			SELECT id,title,author,price_cents,cover_url,created_unix
			FROM books
			WHERE lower(title) LIKE ? OR lower(author) LIKE ?
			ORDER BY id DESC LIMIT ? OFFSET ?`, qp, qp, limit, offset)
	}
	if err != nil { return nil, err }
	defer rows.Close()

	var out []*Book
	for rows.Next() {
		var b Book
		if err := rows.Scan(&b.ID, &b.Title, &b.Author, &b.PriceCents, &b.CoverURL, &b.CreatedUnix); err != nil {
			return nil, err
		}
		out = append(out, &b)
	}
	return out, rows.Err()
}

func (r *sqliteRepo) Get(ctx context.Context, id int64) (*Book, error) {
	var b Book
	err := r.db.QueryRowContext(ctx, `
		SELECT id,title,author,price_cents,cover_url,created_unix
		FROM books WHERE id=?`, id).
		Scan(&b.ID, &b.Title, &b.Author, &b.PriceCents, &b.CoverURL, &b.CreatedUnix)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("book %d not found", id)
		}
		return nil, err
	}
	return &b, nil
}
