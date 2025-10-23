package main
import userpb "github.com/ahinestrog/mybookstore/proto/gen/user"

import (
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	cfg := LoadConfig()

	repo, err := NewUserRepository(cfg.DBPath)
	if err != nil {
		log.Fatal(err)
	}
	pub, err := NewEventPublisher(cfg.RabbitURL)
	if err != nil {
		log.Printf("[user] WARN: RabbitMQ not available (%v). Continuing without events.", err)
		pub = &EventPublisher{}
	}

	svc := NewUserService(repo, pub)

	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatal(err)
	}
	grpcServer := grpc.NewServer()
	userpb.RegisterUserServer(grpcServer, svc)
	reflection.Register(grpcServer)

	fmt.Printf("[user] gRPC listening on :%s\n", cfg.GRPCPort)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal(err)
	}
}
