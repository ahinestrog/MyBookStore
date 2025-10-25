package main
import cartpb "github.com/ahinestrog/mybookstore/proto/gen/cart"

import (
	"context"
	"log"
	"net"
	"os"

	"google.golang.org/grpc"
)

func main() {
	db, err := openSQLite("./data/cart.db")
	if err != nil {
		log.Fatalf("DB open: %v", err)
	}
	if err := migrate(context.Background(), db, "./sql/cart_esquema.sql"); err != nil {
		log.Fatalf("DB migrate: %v", err)
	}

	repo := NewSQLiteRepo(db)
	srv := NewCartServer(repo)

	// puerto configurable vía env CART_GRPC_PORT (por defecto 50050 en .env)
	port := getenv("CART_GRPC_PORT", "50050")
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	cartpb.RegisterCartServer(grpcServer, srv)

	log.Printf("CartService corriendo en :%s", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal(err)
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
