package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/hnsiri/go-event"
	"github.com/hnsiri/go-event/_examples"
)

func main() {
	ctx := context.Background()
	event.Initialize(ctx, event.Option{
		DequeueInterval:  time.Second * 5,
		ConcurrencyLevel: uint(runtime.NumCPU()),
		Logger:           os.Stdout,
	})

	_examples.SampleSyncEvent()

	_examples.SampleAsyncEvent()
	fmt.Println("do your own work... events will be triggered in 5 seconds")

	select {}
}
