package observer

import (
	"context"
	"fmt"
	"sync"
	"time"

	natsclient "github.com/nats-io/nats.go"
	"github.com/wehubfusion/Argus/internal/nats"
	"github.com/wehubfusion/Argus/pkg/event"
	"go.uber.org/zap"
)

// Observer provides a non-blocking interface for emitting observation events
type Observer interface {
	// Emit emits an observation event asynchronously
	// Returns ErrBufferFull if buffer is full and DropOnFull is true
	// Returns ErrObserverClosed if observer is closed
	Emit(ctx context.Context, evt *event.Event) error

	// Close gracefully shuts down the observer
	// Drains the event buffer and ensures all events are published
	Close(ctx context.Context) error
}

// observer implements the Observer interface with async processing
type observer struct {
	publisher *nats.Publisher
	options   Options
	logger    *zap.Logger
	eventChan chan *event.Event
	wg        sync.WaitGroup
	closeOnce sync.Once
	closed    chan struct{}
	mu        sync.RWMutex
	isClosed  bool
}

// NewObserver creates a new Observer instance
func NewObserver(js natsclient.JetStreamContext, opts Options, logger *zap.Logger) (Observer, error) {
	if js == nil {
		return nil, fmt.Errorf("jetstream context cannot be nil")
	}
	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	// Apply defaults
	if opts.BufferSize <= 0 {
		opts = opts.WithBufferSize(DefaultOptions().BufferSize)
	}
	if opts.StreamName == "" {
		opts = opts.WithStreamName(DefaultOptions().StreamName)
	}
	if opts.StreamMaxAge == 0 {
		opts = opts.WithStreamMaxAge(DefaultOptions().StreamMaxAge)
	}
	if opts.StreamMaxMsgs == 0 {
		opts = opts.WithStreamMaxMsgs(DefaultOptions().StreamMaxMsgs)
	}
	if opts.PublishTimeout == 0 {
		opts = opts.WithPublishTimeout(DefaultOptions().PublishTimeout)
	}

	// Create internal publisher
	publisherConfig := nats.PublisherConfig{
		StreamName:     opts.StreamName,
		StreamMaxAge:   opts.StreamMaxAge,
		StreamMaxMsgs:  opts.StreamMaxMsgs,
		PublishTimeout: opts.PublishTimeout,
	}
	publisher, err := nats.NewPublisher(js, publisherConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create publisher: %w", err)
	}

	obs := &observer{
		publisher: publisher,
		options:   opts,
		logger:    logger,
		eventChan: make(chan *event.Event, opts.BufferSize),
		closed:    make(chan struct{}),
	}

	// Start background worker
	obs.wg.Add(1)
	go obs.worker()

	logger.Info("Observer created",
		zap.Int("buffer_size", opts.BufferSize),
		zap.Bool("drop_on_full", opts.DropOnFull),
		zap.String("stream_name", opts.StreamName))

	return obs, nil
}

// Emit emits an observation event asynchronously
func (o *observer) Emit(ctx context.Context, evt *event.Event) error {
	o.mu.RLock()
	closed := o.isClosed
	o.mu.RUnlock()

	if closed {
		return ErrObserverClosed
	}

	// Ensure event has required fields
	if evt.ID == "" {
		evt.ID = fmt.Sprintf("%s-%d", evt.WorkflowID, time.Now().UnixNano())
	}
	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now()
	}
	if evt.Version == "" {
		evt.Version = "v1"
	}

	// Non-blocking send to buffer
	select {
	case o.eventChan <- evt:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("emit cancelled: %w", ctx.Err())
	default:
		// Buffer is full
		if o.options.DropOnFull {
			o.logger.Warn("Event buffer full, dropping event",
				zap.String("event_id", evt.ID),
				zap.String("event_type", evt.Type),
				zap.String("workflow_id", evt.WorkflowID),
				zap.String("run_id", evt.RunID))
			return ErrBufferFull
		}

		// Block until buffer has space
		select {
		case o.eventChan <- evt:
			return nil
		case <-ctx.Done():
			return fmt.Errorf("emit cancelled: %w", ctx.Err())
		case <-o.closed:
			return ErrObserverClosed
		}
	}
}

// worker processes events from the buffer and publishes them
func (o *observer) worker() {
	defer o.wg.Done()

	o.logger.Debug("Observer worker started")

	for {
		select {
		case <-o.closed:
			// Drain remaining events before exiting
			o.drainEvents()
			o.logger.Debug("Observer worker stopped")
			return
		case evt, ok := <-o.eventChan:
			if !ok {
				// Channel closed, drain and exit
				o.drainEvents()
				o.logger.Debug("Observer worker stopped (channel closed)")
				return
			}

			// Validate and serialize event before publishing
			if err := evt.Validate(); err != nil {
				o.logger.Error("Invalid event, skipping",
					zap.String("event_id", evt.ID),
					zap.String("event_type", evt.Type),
					zap.Error(err))
				continue
			}

			// Get subject for event type
			subject := nats.GetSubjectForEventType(evt.Type)
			if subject == "" {
				o.logger.Error("Unknown event type, skipping",
					zap.String("event_id", evt.ID),
					zap.String("event_type", evt.Type))
				continue
			}

			// Serialize event to JSON
			data, err := evt.Bytes()
			if err != nil {
				o.logger.Error("Failed to serialize event, skipping",
					zap.String("event_id", evt.ID),
					zap.String("event_type", evt.Type),
					zap.Error(err))
				continue
			}

			// Publish event using internal publisher
			ctx, cancel := context.WithTimeout(context.Background(), o.options.PublishTimeout)
			if err := o.publisher.Publish(ctx, subject, evt.ID, data); err != nil {
				o.logger.Error("Failed to publish observation event",
					zap.String("event_id", evt.ID),
					zap.String("event_type", evt.Type),
					zap.String("workflow_id", evt.WorkflowID),
					zap.String("run_id", evt.RunID),
					zap.Error(err))
			}
			cancel()
		}
	}
}

// drainEvents drains any remaining events from the buffer
func (o *observer) drainEvents() {
	if o.publisher == nil {
		o.logger.Warn("Cannot drain events: publisher is nil")
		return
	}

	drained := 0
	for {
		select {
		case evt := <-o.eventChan:
			if evt == nil {
				// Channel closed
				if drained > 0 {
					o.logger.Info("Drained events from buffer",
						zap.Int("count", drained))
				}
				return
			}
			// Validate and serialize event
			if err := evt.Validate(); err != nil {
				o.logger.Error("Invalid event during drain, skipping",
					zap.String("event_id", evt.ID),
					zap.String("event_type", evt.Type),
					zap.Error(err))
				continue
			}

			// Get subject for event type
			subject := nats.GetSubjectForEventType(evt.Type)
			if subject == "" {
				o.logger.Error("Unknown event type during drain, skipping",
					zap.String("event_id", evt.ID),
					zap.String("event_type", evt.Type))
				continue
			}

			// Serialize event to JSON
			data, err := evt.Bytes()
			if err != nil {
				o.logger.Error("Failed to serialize event during drain, skipping",
					zap.String("event_id", evt.ID),
					zap.String("event_type", evt.Type),
					zap.Error(err))
				continue
			}

			// Publish event using internal publisher
			ctx, cancel := context.WithTimeout(context.Background(), o.options.PublishTimeout)
			if err := o.publisher.Publish(ctx, subject, evt.ID, data); err != nil {
				o.logger.Error("Failed to publish event during drain",
					zap.String("event_id", evt.ID),
					zap.String("event_type", evt.Type),
					zap.Error(err))
			} else {
				drained++
			}
			cancel()
		default:
			if drained > 0 {
				o.logger.Info("Drained events from buffer",
					zap.Int("count", drained))
			}
			return
		}
	}
}

// Close gracefully shuts down the observer
func (o *observer) Close(ctx context.Context) error {
	var closeErr error
	o.closeOnce.Do(func() {
		o.mu.Lock()
		o.isClosed = true
		o.mu.Unlock()

		// Signal worker to stop
		close(o.closed)

		// Close event channel to stop accepting new events
		close(o.eventChan)

		// Wait for worker to finish with timeout
		done := make(chan struct{})
		go func() {
			o.wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			o.logger.Info("Observer closed gracefully")
		case <-ctx.Done():
			closeErr = fmt.Errorf("close timeout: %w", ctx.Err())
			o.logger.Warn("Observer close timeout",
				zap.Error(ctx.Err()))
		case <-time.After(30 * time.Second):
			closeErr = fmt.Errorf("close timeout after 30 seconds")
			o.logger.Warn("Observer close timeout after 30 seconds")
		}
	})

	return closeErr
}
