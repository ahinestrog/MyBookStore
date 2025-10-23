package main

import (
	"context"

	paymentpb "github.com/ahinestrog/mybookstore/proto/gen/payment"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type service struct {
	paymentpb.UnimplementedPaymentServer
	cfg      Config
	repo     Repository
	provider PaymentProvider
	br       *broker
}

func (s *service) register(grpcServer *grpc.Server) {
	paymentpb.RegisterPaymentServer(grpcServer, s)
	reflection.Register(grpcServer)
}

func (s *service) GetPaymentStatus(ctx context.Context, req *paymentpb.GetPaymentStatusRequest) (*paymentpb.GetPaymentStatusResponse, error) {
	p, err := s.repo.GetByOrderID(ctx, req.OrderId)
	if err != nil || p == nil {
		return &paymentpb.GetPaymentStatusResponse{
			OrderId: req.OrderId,
			State:   paymentpb.PaymentState_PAYMENT_STATE_UNSPECIFIED,
		}, nil
	}
	return &paymentpb.GetPaymentStatusResponse{
		OrderId:     p.OrderID,
		State:       toPBState(p.State),
		ProviderRef: p.ProviderRef,
		UpdatedUnix: p.UpdatedAt.Unix(),
	}, nil
}

func toPBState(s PaymentState) paymentpb.PaymentState {
	switch s {
	case PaymentStatePending:
		return paymentpb.PaymentState_PAYMENT_STATE_PENDING
	case PaymentStateSucceeded:
		return paymentpb.PaymentState_PAYMENT_STATE_SUCCEEDED
	case PaymentStateFailed:
		return paymentpb.PaymentState_PAYMENT_STATE_FAILED
	default:
		return paymentpb.PaymentState_PAYMENT_STATE_UNSPECIFIED
	}
}

// helper para publicar desde service
func (s *service) publishJSON(key string, v any) { s.br.publishJSON(key, v) }

