package main

import (
	"context"
	"log"
	"net"

	"google.golang.org/grpc"
)

func main() {
	cfg := loadConfig()
	ctx := context.Background()

	repo := must(newSQLiteRepo(cfg.DBPath))
	must(struct{}{}, repo.Init(ctx))

	br := newBroker(cfg)
	must(struct{}{}, br.connect())
	defer br.close()

	svc := &service{
		cfg:      cfg,
		repo:     repo,
		provider: newFakeProvider(),
		br:       br,
	}

	// Worker de pagos (consume peticiones de cobro)
	must(struct{}{}, br.consumePaymentRequested(ctx, svc.handlePaymentRequested, cfg.ConsumerTag, cfg.PrefetchCount))

	// gRPC server
	lis := must(net.Listen("tcp", ":"+cfg.ServicePort))
	grpcServer := grpc.NewServer()
	svc.register(grpcServer)

	log.Printf("[payment] gRPC listening on :%s", cfg.ServicePort)
	log.Printf("[payment] DB at %s", cfg.DBPath)
	log.Printf("[payment] consuming queue: %s (exchange=%s)", cfg.RequestQueue, cfg.ExchangeName)

	must(struct{}{}, grpcServer.Serve(lis))
}

