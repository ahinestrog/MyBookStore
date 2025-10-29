package main

import (
	"context"
	"log"
	"net"
	"os"

	cartpb "github.com/ahinestrog/mybookstore/proto/gen/cart"
	catalogpb "github.com/ahinestrog/mybookstore/proto/gen/catalog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

	// Conexión al servicio de catálogo para obtener título y precio
	catalogAddr := getenv("CATALOG_GRPC_ADDR", "catalog:50051")
	catCC, err := grpc.Dial(catalogAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("dial catalog %s: %v", catalogAddr, err)
	}
	defer catCC.Close()

	srv := NewCartServer(repo, catalogpb.NewCatalogClient(catCC))

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
