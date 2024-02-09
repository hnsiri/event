package queue

import (
	"context"
	"io"
)

type Event struct {
	Name string
	Data any
}

type Driver interface {
	Set(ctx context.Context, e string, d any) error
	Listen() (Event, bool)
	SetLogger(l io.Writer)
}
