package main

import (
	"context"

	"github.com/rs/zerolog/log"
	inventorypb "github.com/ahinestrog/mybookstore/proto/gen/inventory"
)

type InventoryServer struct {
	inventorypb.UnimplementedInventoryServer
	Repo *Repository
}

func (s *InventoryServer) GetAvailability(ctx context.Context, req *inventorypb.GetAvailabilityRequest) (*inventorypb.GetAvailabilityResponse, error) {
	avail, err := s.Repo.GetAvailability(ctx, req.GetBookIds())
	if err != nil { return nil, err }

	resp := &inventorypb.GetAvailabilityResponse{ Items: make([]*inventorypb.StockItem, 0, len(avail)) }
	for id, q := range avail {
		resp.Items = append(resp.Items, &inventorypb.StockItem{
			BookId:       id,
			AvailableQty: int32(q),
		})
	}
	log.Debug().Int("count", len(resp.Items)).Msg("GetAvailability")
	return resp, nil
}
