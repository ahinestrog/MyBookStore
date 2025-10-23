package main

import (
	"context"
	"database/sql"
	_ "modernc.org/sqlite"
	"os"
)

func openSQLite(dbPath string) (*sql.DB, error) {
	if err := os.MkdirAll("./data", 0o755); err != nil {
		return nil, err
	}
	// Busy timeout + WAL para concurrencia
	dsn := dbPath
	return sql.Open("sqlite", dsn+"?_pragma=busy_timeout=5000&_pragma=journal_mode=WAL")
}

func migrate(ctx context.Context, db *sql.DB, schemaFile string) error {
	b, err := os.ReadFile(schemaFile)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, string(b))
	return err
}
