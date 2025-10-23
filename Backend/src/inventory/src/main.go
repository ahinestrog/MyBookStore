package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"

	inventorypb "github.com/ahinestrog/mybookstore/proto/gen/inventory"
)

func main() {
	// Logger
	zerolog.TimeFieldFormat = time.RFC3339
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})

	cfg := LoadConfig()
	log.Info().
		Str("addr", cfg.GRPCAddr).
		Str("db", cfg.DBPath).
		Str("rabbit", cfg.RabbitURL).
		Msg("starting inventory service")

	// Repo
	repo, err := NewRepository(cfg.DBPath)
	must(err)
	defer repo.Close()

	if cfg.SeedOnStart {
		must(repo.Seed(context.Background()))
		log.Info().Msg("seeded initial stock")
	}

	// Rabbit
	rabbit, err := NewRabbit(cfg, repo)
	must(err)
	defer rabbit.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	must(rabbit.StartConsumers(ctx))
	log.Info().Msg("rabbit consumers started")

	// gRPC server
	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	must(err)
	grpcSrv := grpc.NewServer()
	inventorypb.RegisterInventoryServer(grpcSrv, &InventoryServer{Repo: repo})

	// Señales para apagado limpio
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		<-ch
		log.Warn().Msg("shutting down...")
		grpcSrv.GracefulStop()
		cancel()
		time.Sleep(ShutdownGrace)
		os.Exit(0)
	}()

	log.Info().Msg("gRPC listening")
	must(grpcSrv.Serve(lis))
}

func must(err error) {
	if err != nil {
		log.Fatal().Err(err).Msg("fatal")
	}
}
