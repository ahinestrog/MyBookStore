package main

import (
	"context"
	"math"

	catalogpb "github.com/ahinestrog/mybookstore/proto/gen/catalog"
	commonpb  "github.com/ahinestrog/mybookstore/proto/gen/common"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultPageSize int32 = 20
	maxPageSize     int32 = 100
)

type CatalogServer struct {
	catalogpb.UnimplementedCatalogServer
	repo Repository
}

func NewCatalogServer(repo Repository) *CatalogServer { return &CatalogServer{repo: repo} }

func (s *CatalogServer) ListBooks(ctx context.Context, in *catalogpb.ListBooksRequest) (*catalogpb.ListBooksResponse, error) {
	// Normaliza paginación
	page := int32(1)
	size := defaultPageSize
	if in.GetPage() != nil {
		if in.Page.Page > 0 { page = in.Page.Page }
		if in.Page.PageSize > 0 { size = in.Page.PageSize }
	}
	if size > maxPageSize { size = maxPageSize }
	offset := (page - 1) * size

	total, err := s.repo.Count(ctx, in.GetQ())
	if err != nil { return nil, status.Errorf(codes.Internal, "count: %v", err) }

	items, err := s.repo.List(ctx, in.GetQ(), size, offset)
	if err != nil { return nil, status.Errorf(codes.Internal, "list: %v", err) }

	out := make([]*catalogpb.Book, 0, len(items))
	for _, b := range items { out = append(out, bookToPB(b)) }

	totalPages := int32(0)
	if size > 0 {
		totalPages = int32(math.Ceil(float64(total) / float64(size)))
	}

	return &catalogpb.ListBooksResponse{
		Items: out,
		Page: &commonpb.PageResponse{
			Page:       page,
			PageSize:   size,
			TotalPages: totalPages,
			TotalItems: total,
		},
	}, nil
}

func (s *CatalogServer) GetBook(ctx context.Context, in *catalogpb.GetBookRequest) (*catalogpb.Book, error) {
	if in.GetId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "id must be > 0")
	}
	b, err := s.repo.Get(ctx, in.GetId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "%v", err)
	}
	return bookToPB(b), nil
}

