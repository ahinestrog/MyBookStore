package main
import cartpb "github.com/ahinestrog/mybookstore/proto/gen/cart"

import (
	"context"
	"log"
	"net"

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

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	cartpb.RegisterCartServer(grpcServer, srv)

	log.Println("CartService corriendo en :50051")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal(err)
	}
}
