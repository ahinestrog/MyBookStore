package main

import (
	"context"
	"database/sql"
	"errors"
	"time"

	_ "modernc.org/sqlite"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(dsn string) (*UserRepository, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	repo := &UserRepository{db: db}
	if err := repo.migrate(); err != nil {
		return nil, err
	}
	return repo, nil
}

func (r *UserRepository) migrate() error {
	schema := `
CREATE TABLE IF NOT EXISTS users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  email TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL
);`
	_, err := r.db.Exec(schema)
	return err
}

func (r *UserRepository) Create(ctx context.Context, u *User) (int64, error) {
	now := time.Now().UTC()
	u.CreatedAt, u.UpdatedAt = now, now
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO users(name,email,password_hash,created_at,updated_at)
		 VALUES(?,?,?,?,?)`, u.Name, u.Email, u.PasswordHash, u.CreatedAt, u.UpdatedAt)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *UserRepository) GetByID(ctx context.Context, id int64) (*User, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id,name,email,password_hash,created_at,updated_at FROM users WHERE id=?`, id)
	u := &User{}
	if err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return u, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id,name,email,password_hash,created_at,updated_at FROM users WHERE email=?`, email)
	u := &User{}
	if err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return u, nil
}

func (r *UserRepository) UpdateName(ctx context.Context, id int64, name string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE users SET name=?, updated_at=? WHERE id=?`, name, time.Now().UTC(), id)
	return err
}
