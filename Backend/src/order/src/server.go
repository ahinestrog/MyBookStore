package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	orderpb "github.com/ahinestrog/mybookstore/proto/gen/order"
	commonpb "github.com/ahinestrog/mybookstore/proto/gen/common"
)

type OrderServer struct {
	orderpb.UnimplementedOrderServer
	repo   *Repository
	rabbit *Rabbit
	cart   *CartClient
}

func NewOrderServer(repo *Repository, rb *Rabbit, cart *CartClient) *OrderServer {
	return &OrderServer{repo: repo, rabbit: rb, cart: cart}
}

func (s *OrderServer) CreateOrder(ctx context.Context, req *orderpb.CreateOrderRequest) (*orderpb.CreateOrderResponse, error) {
	if req.GetUserId() == 0 {
		return nil, errors.New("user_id requerido")
	}
	// 1) Obtener carrito
	cv, err := s.cart.GetCart(ctx, req.GetUserId())
	if err != nil { return nil, err }
	if len(cv.Items) == 0 {
		return nil, errors.New("carrito vacío")
	}

	// 2) Mapear ítems y totales
	var o Order
	o.UserID = req.GetUserId()
	o.Status = OrderStatusCreated
	o.CreatedUnix = nowUnix()
	o.UpdatedUnix = o.CreatedUnix

	var itemsEvt []OrderItemEvt
	for _, it := range cv.Items {
		unit := it.UnitPrice.GetCents()
		line := it.LineTotal.GetCents()
		o.Items = append(o.Items, OrderItem{
			BookID:    it.BookId,
			Title:     it.Title,
			Qty:       it.Qty,
			UnitCents: unit,
			LineCents: line,
		})
		itemsEvt = append(itemsEvt, OrderItemEvt{
			BookID:    it.BookId,
			Title:     it.Title,
			Qty:       it.Qty,
			UnitCents: unit,
			LineCents: line,
		})
		o.TotalCents += line
	}

	oid, err := s.repo.CreateOrder(ctx, &o)
	if err != nil { return nil, err }

	// 3) Publicar evento order.created
	payload := OrderCreatedPayload{
		OrderID:    oid,
		UserID:     o.UserID,
		Items:      itemsEvt,
		TotalCents: o.TotalCents,
	}
	if err := s.rabbit.PublishJSON(RKOrderCreated, payload); err != nil {
		log.Printf("[order] WARN publish order.created failed: %v", err)
	}

	// 4) Responder
	respItems := make([]*orderpb.OrderItem, 0, len(o.Items))
	for _, it := range o.Items {
		respItems = append(respItems, &orderpb.OrderItem{
			BookId:   it.BookID,
			Title:    it.Title,
			Qty:      it.Qty,
			UnitPrice: &commonpb.Money{Cents: it.UnitCents},
			LineTotal: &commonpb.Money{Cents: it.LineCents},
		})
	}
	return &orderpb.CreateOrderResponse{
		OrderId: oid,
		Status:  orderpb.OrderStatus_ORDER_STATUS_CREATED,
		Items:   respItems,
		Total:   &commonpb.Money{Cents: o.TotalCents},
	}, nil
}

func (s *OrderServer) GetOrderStatus(ctx context.Context, req *orderpb.GetOrderStatusRequest) (*orderpb.GetOrderStatusResponse, error) {
	o, err := s.repo.GetOrder(ctx, req.GetOrderId())
	if err != nil { return nil, err }
	return &orderpb.GetOrderStatusResponse{
		OrderId:     o.ID,
		Status:      orderStatusToPB(o.Status),
		Total:       &commonpb.Money{Cents: o.TotalCents},
		UpdatedUnix: o.UpdatedUnix,
	}, nil
}

func orderStatusToPB(st int32) orderpb.OrderStatus {
	switch st {
	case OrderStatusCreated:
		return orderpb.OrderStatus_ORDER_STATUS_CREATED
	case OrderStatusPaid:
		return orderpb.OrderStatus_ORDER_STATUS_PAID
	case OrderStatusCancelled:
		return orderpb.OrderStatus_ORDER_STATUS_CANCELLED
	case OrderStatusFailed:
		return orderpb.OrderStatus_ORDER_STATUS_FAILED
	default:
		return orderpb.OrderStatus_ORDER_STATUS_UNSPECIFIED
	}
}

// Consumidores de RabbitMQ

func (s *OrderServer) StartConsumers() error {
	// Cola dedicada del servicio order
	return s.rabbit.ConsumeTopic("order-service",
		[]string{RKInventoryReserved, RKInventoryRejected, RKPaymentCaptured, RKPaymentFailed},
		s.handleEvent)
}

func (s *OrderServer) handleEvent(rk string, body []byte) error {
	switch rk {
	case RKInventoryReserved:
		var p InventoryResultPayload
		if err := json.Unmarshal(body, &p); err != nil { return err }
		if p.OK {
			// Ya confirmado stock → solicitar cobro
			o, err := s.repo.GetOrder(context.Background(), p.OrderID)
			if err != nil { return err }
			ch := PaymentChargePayload{
				OrderID:    o.ID,
				UserID:     o.UserID,
				TotalCents: o.TotalCents,
			}
			if err := s.rabbit.PublishJSON(RKPaymentCharge, ch); err != nil {
				log.Printf("[order] publish payment.charge error: %v", err)
			}
		} else {
			_ = s.repo.UpdateStatus(context.Background(), p.OrderID, OrderStatusFailed)
		}

	case RKInventoryRejected:
		var p InventoryResultPayload
		if err := json.Unmarshal(body, &p); err != nil { return err }
		_ = s.repo.UpdateStatus(context.Background(), p.OrderID, OrderStatusFailed)

	case RKPaymentCaptured:
		var p PaymentResultPayload
		if err := json.Unmarshal(body, &p); err != nil { return err }
		if p.OK {
			_ = s.repo.UpdateStatus(context.Background(), p.OrderID, OrderStatusPaid)
		} else {
			_ = s.repo.UpdateStatus(context.Background(), p.OrderID, OrderStatusFailed)
		}

	case RKPaymentFailed:
		var p PaymentResultPayload
		if err := json.Unmarshal(body, &p); err != nil { return err }
		_ = s.repo.UpdateStatus(context.Background(), p.OrderID, OrderStatusFailed)
	}
	return nil
}

