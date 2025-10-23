package main

import (
	"context"
	"fmt"
)

type Events interface {
	Publish(ctx context.Context, routingKey string, body []byte) error
}

type Service struct {
	repo   Repository
	events Events 
}

func NewService(repo Repository, events Events) *Service {
	return &Service{repo: repo, events: events}
}

func (s *Service) publish(ctx context.Context, key string, payload []byte) {
	if s.events == nil { return }
	_ = s.events.Publish(ctx, key, payload)
}

func (s *Service) OnCreated(ctx context.Context, b *Book) {
	s.publish(ctx, "catalog.book.created", []byte(fmt.Sprintf(`{"id":%d}`, b.ID)))
}
func (s *Service) OnUpdated(ctx context.Context, b *Book) {
	s.publish(ctx, "catalog.book.updated", []byte(fmt.Sprintf(`{"id":%d}`, b.ID)))
}
func (s *Service) OnDeleted(ctx context.Context, id int64) {
	s.publish(ctx, "catalog.book.deleted", []byte(fmt.Sprintf(`{"id":%d}`, id)))
}
