package main

import (
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	orderpb "github.com/ahinestrog/mybookstore/proto/gen/order"
)

func main() {
	cfg := LoadConfig()

	repo, err := NewRepository(cfg.DBPath)
	if err != nil { log.Fatalf("db err: %v", err) }
	defer repo.Close()

	rb, err := NewRabbit(cfg.RabbitURL, cfg.RabbitExchange)
	if err != nil { log.Fatalf("rabbit err: %v", err) }
	defer rb.Close()

	cartClient := NewCartClient(cfg.CartGRPCAddr)

	srv := NewOrderServer(repo, rb, cartClient)
	if err := srv.StartConsumers(); err != nil {
		log.Fatalf("consumers err: %v", err)
	}

	grpcServer := grpc.NewServer()
	orderpb.RegisterOrderServer(grpcServer, srv)
	reflection.Register(grpcServer)

	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil { log.Fatalf("listen err: %v", err) }
	log.Printf("[order] gRPC listening on %s", cfg.GRPCAddr)

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("serve err: %v", err)
	}
}

