// Servidor que utiliza la comunicación RPC (gRPC)
package main

import (
	"context"

	cartpb "github.com/ahinestrog/mybookstore/proto/gen/cart"
	catalogpb "github.com/ahinestrog/mybookstore/proto/gen/catalog"
	commonpb "github.com/ahinestrog/mybookstore/proto/gen/common"
)

type CartServer struct {
	cartpb.UnimplementedCartServer
	repo    CartRepository
	catalog catalogpb.CatalogClient
}

func NewCartServer(repo CartRepository, catalog catalogpb.CatalogClient) *CartServer {
	return &CartServer{repo: repo, catalog: catalog}
}

func (s *CartServer) GetCart(ctx context.Context, req *commonpb.UserRef) (*cartpb.CartView, error) {
	c, err := s.repo.GetOrCreateCart(ctx, req.GetUserId())
	if err != nil {
		return nil, err
	}
	return toCartView(c), nil
}

func (s *CartServer) AddItem(ctx context.Context, req *cartpb.AddItemRequest) (*cartpb.CartView, error) {
	b, err := s.catalog.GetBook(ctx, &catalogpb.GetBookRequest{Id: req.BookId})
	if err != nil {
		return nil, err
	}
	c, err := s.repo.AddItem(ctx, req.UserId, req.BookId, b.GetTitle(), b.GetPrice().GetCents(), req.Qty)
	if err != nil {
		return nil, err
	}
	return toCartView(c), nil
}

func (s *CartServer) RemoveItem(ctx context.Context, req *cartpb.RemoveItemRequest) (*cartpb.CartView, error) {
	c, err := s.repo.RemoveItem(ctx, req.UserId, req.BookId, req.Qty)
	if err != nil {
		return nil, err
	}
	return toCartView(c), nil
}

func (s *CartServer) ClearCart(ctx context.Context, req *commonpb.UserRef) (*cartpb.CartView, error) {
	c, err := s.repo.Clear(ctx, req.UserId)
	if err != nil {
		return nil, err
	}
	return toCartView(c), nil
}

func toCartView(c *Cart) *cartpb.CartView {
	view := &cartpb.CartView{}
	var total int64
	for _, it := range c.Items {
		line := it.UnitPriceCents * int64(it.Qty)
		total += line
		view.Items = append(view.Items, &cartpb.CartItem{
			BookId:    it.BookID,
			Title:     it.Title,
			Qty:       it.Qty,
			UnitPrice: &commonpb.Money{Cents: it.UnitPriceCents},
			LineTotal: &commonpb.Money{Cents: line},
		})
	}
	view.Total = &commonpb.Money{Cents: total}
	return view
}
