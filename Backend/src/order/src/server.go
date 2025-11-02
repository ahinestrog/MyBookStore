package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	commonpb "github.com/ahinestrog/mybookstore/proto/gen/common"
	orderpb "github.com/ahinestrog/mybookstore/proto/gen/order"
)

type OrderServer struct {
	orderpb.UnimplementedOrderServer
	repo   *Repository
	rabbit *Rabbit
	cart   *CartClient
	cfg    *Config
}

func NewOrderServer(cfg *Config, repo *Repository, rb *Rabbit, cart *CartClient) *OrderServer {
	return &OrderServer{cfg: cfg, repo: repo, rabbit: rb, cart: cart}
}

func (s *OrderServer) CreateOrder(ctx context.Context, req *orderpb.CreateOrderRequest) (*orderpb.CreateOrderResponse, error) {
	if req.GetUserId() == 0 {
		return nil, errors.New("user_id requerido")
	}
	// 1) Obtener carrito
	cv, err := s.cart.GetCart(ctx, req.GetUserId())
	if err != nil {
		return nil, err
	}
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
	if err != nil {
		return nil, err
	}

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

	// 3b) Enviar solicitud de reserva a Inventario (cola directa)
	invItems := make([]InvOrderItem, 0, len(o.Items))
	for _, it := range o.Items {
		invItems = append(invItems, InvOrderItem{BookID: it.BookID, Qty: it.Qty})
	}
	if err := s.rabbit.PublishJSONToQueue(s.cfg.QReserveReq, InvReserveRequest{
		OrderID: oid,
		UserID:  o.UserID,
		Items:   invItems,
	}); err != nil {
		log.Printf("[order] WARN publish reserve.request failed: %v", err)
	}

	// 4) Responder
	respItems := make([]*orderpb.OrderItem, 0, len(o.Items))
	for _, it := range o.Items {
		respItems = append(respItems, &orderpb.OrderItem{
			BookId:    it.BookID,
			Title:     it.Title,
			Qty:       it.Qty,
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
	if err != nil {
		return nil, err
	}
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
	// 1) Consumir resultados de inventario desde la cola directa
	if err := s.rabbit.ConsumeQueue(s.cfg.QReserveRes, "order-inventory-results", s.handleInventoryResult); err != nil {
		return err
	}
	// 2) Consumir eventos de pago
	return s.rabbit.ConsumeTopic("order-service", []string{RKPaymentSucceeded, RKPaymentFailed}, s.handleEvent)
}

func (s *OrderServer) handleEvent(rk string, body []byte) error {
	switch rk {
	case RKPaymentSucceeded:
		var p PaymentSucceeded
		if err := json.Unmarshal(body, &p); err != nil {
			return err
		}
		// Confirmar stock y marcar como pagada
		o, err := s.repo.GetOrder(context.Background(), p.OrderID)
		if err != nil {
			return err
		}
		invItems := make([]InvOrderItem, 0, len(o.Items))
		for _, it := range o.Items {
			invItems = append(invItems, InvOrderItem{BookID: it.BookID, Qty: it.Qty})
		}
		if err := s.rabbit.PublishJSONToQueue(s.cfg.QConfirmReq, InvConfirmRequest{OrderID: o.ID, Items: invItems}); err != nil {
			log.Printf("[order] publish inventory confirm error: %v", err)
		}
		_ = s.repo.UpdateStatus(context.Background(), p.OrderID, OrderStatusPaid)

	case RKPaymentFailed:
		var p PaymentFailed
		if err := json.Unmarshal(body, &p); err != nil {
			return err
		}
		// Liberar reserva en inventario y marcar como fallida
		o, err := s.repo.GetOrder(context.Background(), p.OrderID)
		if err != nil {
			return err
		}
		invItems := make([]InvOrderItem, 0, len(o.Items))
		for _, it := range o.Items {
			invItems = append(invItems, InvOrderItem{BookID: it.BookID, Qty: it.Qty})
		}
		if err := s.rabbit.PublishJSONToQueue(s.cfg.QReleaseReq, InvReleaseRequest{OrderID: o.ID, Items: invItems}); err != nil {
			log.Printf("[order] publish inventory release error: %v", err)
		}
		_ = s.repo.UpdateStatus(context.Background(), p.OrderID, OrderStatusFailed)
	}
	return nil
}

func (s *OrderServer) handleInventoryResult(body []byte) error {
	var res InvReserveResult
	if err := json.Unmarshal(body, &res); err != nil {
		return err
	}
	if res.State == "RESERVED" {
		// Solicitar cobro
		o, err := s.repo.GetOrder(context.Background(), res.OrderID)
		if err != nil {
			return err
		}
		req := PaymentRequested{OrderID: o.ID, UserID: o.UserID, AmountCents: o.TotalCents}
		if err := s.rabbit.PublishJSON(RKPaymentChargeRequested, req); err != nil {
			log.Printf("[order] publish payment.requested error: %v", err)
		}
		return nil
	}
	// Reserva falló → marcar orden como fallida
	_ = s.repo.UpdateStatus(context.Background(), res.OrderID, OrderStatusFailed)
	return nil
}
