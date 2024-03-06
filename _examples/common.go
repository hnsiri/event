package _examples

import (
	"context"

	"github.com/hnsiri/go-event"
)

type Product struct {
	Stock    int
	Title    string
	Supplier struct {
		Name  string
		Email string
	}
	Retailer struct {
		Name  string
		Email string
	}
	Customer struct {
		Name  string
		Email string
	}
}

func Subscribe(evt string, t any, p event.Priority) {
	event.Subscribe(evt, func(ctx context.Context, e event.Event, d any) (any, error) {
		return d, nil
	}, t, p)
}
