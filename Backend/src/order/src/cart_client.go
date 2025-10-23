package main

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	cartpb "github.com/ahinestrog/mybookstore/proto/gen/cart"
	commonpb "github.com/ahinestrog/mybookstore/proto/gen/common"
)

type CartClient struct {
	addr string
}

func NewCartClient(addr string) *CartClient { return &CartClient{addr: addr} }

func (c *CartClient) GetCart(ctx context.Context, userID int64) (*cartpb.CartView, error) {
	cc, err := grpc.DialContext(ctx, c.addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil { return nil, err }
	defer cc.Close()
	client := cartpb.NewCartClient(cc)
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	return client.GetCart(ctx, &commonpb.UserRef{UserId: userID})
}
