package _examples

import (
	"context"
	"fmt"

	"github.com/hnsiri/go-event"
)

func SampleAsyncEvent() {
	data := Product{
		Stock: 10,
		Title: "Sample Product",
		Supplier: struct {
			Name  string
			Email string
		}{Name: "John", Email: "s0BqK@example.com"},
		Retailer: struct {
			Name  string
			Email string
		}{Name: "Jane", Email: "l7h6K@example.com"},
		Customer: struct {
			Name  string
			Email string
		}{Name: "Joe", Email: "wZ8yA@example.com"},
	}

	event.Subscribe("ProductCreated#NotifyAdmin", func(ctx context.Context, e event.Event, d any) (any, error) {
		product := d.(Product)
		// do what you want
		fmt.Println("Admin notified!")
		return product, nil
	}, Product{}, 0)

	event.Subscribe("ProductCreated#SendReviewRequiredEmail", func(ctx context.Context, e event.Event, d any) (any, error) {
		product := d.(Product)
		// do what you want
		fmt.Println("Notify Supplier about review required!")
		return product, nil
	}, Product{}, 1)

	ctx := context.Background()
	err := event.TriggerAsync(ctx, "ProductCreated", data)
	if err != nil {
		fmt.Println(err)
	}
}
