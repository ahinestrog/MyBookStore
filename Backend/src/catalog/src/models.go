package main

import (
	catalogpb "github.com/ahinestrog/mybookstore/proto/gen/catalog"
	commonpb  "github.com/ahinestrog/mybookstore/proto/gen/common"
)

type Book struct {
	ID          int64
	Title       string
	Author      string
	PriceCents  int64
	CoverURL    string
	CreatedUnix int64
}

// ---- mapping entidad <-> protobuf ----

func bookToPB(b *Book) *catalogpb.Book {
	return &catalogpb.Book{
		Id:         b.ID,
		Title:      b.Title,
		Author:     b.Author,
		Price:      &commonpb.Money{Cents: b.PriceCents},
		CoverUrl:   b.CoverURL,
		CreatedUnix:b.CreatedUnix,
	}
}

