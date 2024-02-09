package event

import (
	"context"
	"errors"
	"io"
	"reflect"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/hnsiri/go-event/queue"
)

type Priority uint8

const (
	High        Priority = 0
	AboveNormal Priority = 64
	Normal      Priority = 128
	BelowNormal Priority = 192
	Low         Priority = 255

	Splitter string = "#"
)

const (
	DefaultConcurrencyLevel uint          = 1
	DefaultDequeueInterval  time.Duration = time.Second
)

var (
	ErrEventNotRegistered       = errors.New("event not registered yet")
	ErrDataTypeMismatched       = errors.New("triggered data type and registered data type are not equal")
	ErrReturnDataTypeMismatched = errors.New("returned data type and triggered data type are not equal")
	ErrFailedToPutInQueue       = errors.New("failed to put event in the queue")
)

type Listener func(ctx context.Context, e Event, d any) (any, error)

type Event struct {
	// Name represents the identifier or label for the event.
	Name string
	// TriggeredAt stores the time when the event was triggered.
	TriggeredAt time.Time
	// ExecutedAt stores the timestamp when the event was executed.
	ExecutedAt time.Time
	// Priority indicates the priority level of the event execution.
	Priority Priority

	listener Listener
}

type Registry struct {
	mu                 sync.RWMutex
	registered         map[string]map[Priority][]Event
	eventsType         map[string]reflect.Type
	listenerPriorities map[string][]Priority

	concurrencyLevel uint
	dequeueInterval  time.Duration
	logger           io.Writer
	queue            queue.Driver
}

type Option struct {
	// ConcurrencyLevel: By default, asynchronous events are triggered one by one.
	// However, by increasing this value, you can control the number of events processed concurrently.
	ConcurrencyLevel uint

	// Interval: Defines the time duration between successive attempts to dequeue an event from the queue.
	// It dictates how often the event manager checks the queue for new events,
	// ensuring smooth and efficient event processing.
	DequeueInterval time.Duration

	Logger io.Writer

	// Queue: Where asynchronous events are stored for processing. It serves as a buffer for incoming events,
	// allowing the event manager to efficiently manage and prioritize them for handling.
	// When utilizing Kafka and Redis drivers, the queue gains distributed features,
	// enabling scalable and fault-tolerant event processing across distributed systems.
	Queue queue.Driver
}

var registry Registry

func options(opt Option) Option {
	if opt.ConcurrencyLevel == 0 {
		opt.ConcurrencyLevel = DefaultConcurrencyLevel
	}
	if opt.DequeueInterval.Nanoseconds() == 0 {
		opt.DequeueInterval = DefaultDequeueInterval
	}
	if opt.Queue == nil {
		q := queue.NewInMemory()
		opt.Queue = &q
	}

	return opt
}

// Initialize sets up the event package to function as a registry, allowing direct usage of Subscribe, Trigger,
// and TriggerAsync functions.
// On the other hand, by using the New function, you can obtain an instance of the registry.
func Initialize(ctx context.Context, opt Option) {
	registry = New(options(opt))
	go registry.ObserveQueue(ctx)
}

func New(opt Option) Registry {
	return Registry{
		registered:         make(map[string]map[Priority][]Event),
		eventsType:         make(map[string]reflect.Type),
		listenerPriorities: make(map[string][]Priority),

		concurrencyLevel: opt.ConcurrencyLevel,
		dequeueInterval:  opt.DequeueInterval,
		logger:           opt.Logger,
		queue:            opt.Queue,
	}
}

// init initializes and allocates an object for the given event
func (r *Registry) init(en string) {
	if r.registered[en] == nil {
		r.registered[en] = make(map[Priority][]Event)
	}
}

// Subscribe subscribes or registers the given listener for the specified event.
// Listeners will run based on their defined priorities.
// If priorities are the same, they will run in the order they were added.
//
// The EventNameSplitter "#" is used to separate components in an event name.
// This allows for easy identification of triggered events, aiding in error tracing
// when multiple listeners are registered for an event.
// Pattern: EventName#ListenerLabel
// Example: OrderPlaced#SendConfirmationEmail
// Example: OrderPlaced#UpdateInventory
// Example: OrderPlaced#NotifySalesTeam
//
// Listeners within an event pass the provided data among themselves,
// which indicates the implementation of a chain of responsibility pattern.
//
// Priority must be between 0 and 255; lower values indicate higher priorities.
func (r *Registry) Subscribe(evt string, l Listener, t any, p Priority) {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := evt
	if strings.Contains(evt, Splitter) && strings.Count(evt, Splitter) == 1 &&
		!strings.HasPrefix(evt, Splitter) && !strings.HasSuffix(evt, Splitter) {
		evt = strings.Split(evt, Splitter)[0]
	}

	r.init(evt)
	r.registered[evt][p] = append(r.registered[evt][p], Event{
		Name:     name,
		listener: l,
	})
	r.eventsType[evt] = reflect.TypeOf(t)

	// sort event listeners by their priorities
	r.listenerPriorities[evt] = []Priority{}
	for pr := range r.registered[evt] {
		r.listenerPriorities[evt] = append(r.listenerPriorities[evt], pr)
	}
	slices.Sort(r.listenerPriorities[evt])
}

func (r *Registry) Trigger(ctx context.Context, evt string, d any) error {
	return r.trigger(ctx, evt, d)
}

// TriggerAsync adds incoming events to the queue for asynchronous processing.
func (r *Registry) TriggerAsync(ctx context.Context, evt string, d any) error {
	err := r.queue.Set(ctx, evt, d)
	if err != nil {
		err = errors.Join(ErrFailedToPutInQueue, err)
		r.log(err.Error())
		return err
	}

	return nil
}

// Trigger triggers the specified event with given data.
// The data type must be the same as the data type registered for the event,
// otherwise, the ErrDataTypeMismatched error will be returned.
//
// If one of the listeners returns an error, subsequent listeners in the chain will not be executed.
func (r *Registry) trigger(ctx context.Context, evt string, d any) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, ok := r.registered[evt]; !ok {
		return ErrEventNotRegistered
	}

	dType := reflect.TypeOf(d)
	if dType != r.eventsType[evt] {
		return ErrDataTypeMismatched
	}

	triggeredAt := time.Now()
	for _, p := range r.listenerPriorities[evt] {
		for _, ev := range r.registered[evt][p] {
			ev.Priority = p
			ev.TriggeredAt = triggeredAt
			ev.ExecutedAt = time.Now()
			res, err := ev.listener(ctx, ev, d)
			if err != nil {
				return err
			}

			if reflect.TypeOf(res) != dType {
				return ErrReturnDataTypeMismatched
			}

			d = res
		}
	}

	return nil
}

// ObserveQueue observes the queue and dispatches asynchronous events.
func (r *Registry) ObserveQueue(ctx context.Context) {
	ticker := time.NewTicker(r.dequeueInterval)

	for {
		select {
		case <-ctx.Done():
			r.log("event queue observer closed")
			return
		case <-ticker.C:
			triggered := uint(0)
			wg := sync.WaitGroup{}
			for {
				ev, ok := r.queue.Listen()
				if !ok {
					break
				}

				triggered++
				go func() {
					wg.Add(1)
					defer func() {
						if err := recover(); err != nil {
							r.log(err.(error).Error())
						}
						wg.Done()
					}()

					err := r.trigger(context.Background(), ev.Name, ev.Data)
					if err != nil {
						r.log(err.Error())
					}
				}()

				if triggered == r.concurrencyLevel {
					break
				}
			}
			wg.Wait()
		}
	}
}

func (r *Registry) log(s string) {
	if r.logger == nil {
		return
	}

	_, _ = r.logger.Write([]byte(s))
}

// Subscribe subscribes or registers the given listener for the specified event.
// See Subscribe method of the Registry for more info.
func Subscribe(evt string, l Listener, t any, p Priority) {
	registry.Subscribe(evt, l, t, p)
}

// Trigger triggers the specified event with given data.
// See the Trigger method of the Registry for more info.
func Trigger(ctx context.Context, evt string, d any) error {
	return registry.Trigger(ctx, evt, d)
}

// TriggerAsync triggers the specified event with given data asynchronously.
// See the TriggerAsync method of the Registry for more info.
func TriggerAsync(ctx context.Context, evt string, d any) error {
	return registry.TriggerAsync(ctx, evt, d)
}
