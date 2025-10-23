package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	catalogpb "github.com/ahinestrog/mybookstore/proto/gen/catalog"
	_ "github.com/mattn/go-sqlite3"
	"google.golang.org/grpc"
)

const (
	defaultPort   = "50051"
	defaultDBPath = "./data/catalog.db"
)

func mustRead(path string) string {
	b, err := os.ReadFile(path)
	if err != nil { panic(err) }
	return string(b)
}

func openSQLite(path string) (*sql.DB, error) {
	if err := os.MkdirAll("data", 0o755); err != nil { return nil, err }
	db, err := sql.Open("sqlite3", fmt.Sprintf("%s?_busy_timeout=5000&_foreign_keys=on", path))
	if err != nil { return nil, err }
	db.SetMaxOpenConns(1)
	return db, nil
}

func main() {
	port   := getenv("GRPC_PORT", defaultPort)
	dbPath := getenv("CATALOG_DB_PATH", defaultDBPath)

	// DB + migración + seed opcional
	db, err := openSQLite(dbPath)
	if err != nil { log.Fatalf("open db: %v", err) }
	defer db.Close()

	repo := NewSQLiteRepo(db)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := db.ExecContext(ctx, mustRead("./db/db.sql")); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	// Seed si está vacío
	var c int64
	if err := db.QueryRowContext(ctx, `SELECT COUNT(1) FROM books`).Scan(&c); err == nil && c == 0 {
		if _, err := db.ExecContext(ctx, mustRead("./db/seed.sql")); err != nil {
			log.Printf("seed warn: %v", err)
		}
	}

	// gRPC
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil { log.Fatalf("listen: %v", err) }
	s := grpc.NewServer()
	catalogpb.RegisterCatalogServer(s, NewCatalogServer(repo))
	log.Printf("Catalog service listening :%s  db=%s", port, dbPath)
	if err := s.Serve(lis); err != nil { log.Fatalf("serve: %v", err) }
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" { return v }
	return def
}

