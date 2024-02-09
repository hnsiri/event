package queue

import (
	"container/list"
	"context"
	"io"
	"sync"
)

type InMemory struct {
	mu   sync.Mutex
	list list.List
}

func NewInMemory() InMemory {
	return InMemory{}
}

func (m *InMemory) Set(ctx context.Context, e string, d any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.list.PushBack(Event{
		Name: e,
		Data: d,
	})

	return nil
}

func (m *InMemory) Listen() (Event, bool) {
	element := m.list.Front()
	if element == nil {
		return Event{}, false
	}

	m.list.Remove(element)
	return element.Value.(Event), true
}

func (m *InMemory) SetLogger(l io.Writer) {}
