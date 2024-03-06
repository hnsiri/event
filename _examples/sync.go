package _examples

import (
	"context"
	"fmt"
	"time"

	"github.com/hnsiri/go-event"
)

func SampleSyncEvent() {
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

	go func() {
		event.Subscribe("OrderPlaced#SendConfirmationEmail", func(ctx context.Context, e event.Event, d any) (any, error) {
			product := d.(Product)
			// do what you want
			fmt.Println("Email sent!")
			return product, nil
		}, Product{}, 0)

		event.Subscribe("OrderPlaced#SendConfirmationSMS", func(ctx context.Context, e event.Event, d any) (any, error) {
			product := d.(Product)
			// do what you want
			fmt.Println("SMS sent!")
			return product, nil
		}, Product{}, 1)
	}()

	go func() {
		event.Subscribe("OrderPlaced#NotifyRetailer", func(ctx context.Context, e event.Event, d any) (any, error) {
			product := d.(Product)
			// do what you want
			fmt.Println("Retailer notified!")
			return product, nil
		}, Product{}, event.AboveNormal)

		event.Subscribe("OrderPlaced#NotifySupplier", func(ctx context.Context, e event.Event, d any) (any, error) {
			product := d.(Product)
			// do what you want
			fmt.Println("Supplier notified!")
			return product, nil
		}, Product{}, event.AboveNormal)
	}()

	event.Subscribe("OrderPlaced#UpdateInventory", func(ctx context.Context, e event.Event, d any) (any, error) {
		product := d.(Product)
		// do what you want
		fmt.Println("Inventory updated!")
		return product, nil
	}, Product{}, event.Low)

	// wait a second to ensure all subscribers are ready because of the async nature of goroutines
	time.Sleep(time.Second * 1)

	ctx := context.Background()
	err := event.Trigger(ctx, "OrderPlaced", data)
	if err != nil {
		fmt.Println(err)
	}
}
