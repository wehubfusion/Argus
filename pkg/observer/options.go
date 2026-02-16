package observer

import (
	"time"

	"github.com/wehubfusion/Argus/pkg/event"
)

// Options configures the Observer behavior
type Options struct {
	// BufferSize is the size of the internal channel buffer for events
	// Default: 1000
	BufferSize int

	// DropOnFull determines behavior when buffer is full
	// If true, new events are dropped when buffer is full
	// If false, Emit will block until buffer has space
	// Default: true (drop-on-pressure)
	DropOnFull bool

	// StreamName is the JetStream stream name for observation events (default: event.StreamName)
	StreamName string

	// StreamMaxAge is the maximum age for messages in the stream
	// Default: 30 days
	StreamMaxAge time.Duration

	// StreamMaxMsgs is the maximum number of messages in the stream
	// Default: 1000000
	StreamMaxMsgs int64

	// PublishTimeout is the timeout for publishing events to JetStream
	// Default: 5 seconds
	PublishTimeout time.Duration
}

// DefaultOptions returns default options for the Observer
func DefaultOptions() Options {
	return Options{
		BufferSize:     1000,
		DropOnFull:     true,
		StreamName:     event.StreamName,
		StreamMaxAge:   30 * 24 * time.Hour, // 30 days
		StreamMaxMsgs:  1000000,
		PublishTimeout: 5 * time.Second,
	}
}

// WithBufferSize sets the buffer size
func (o Options) WithBufferSize(size int) Options {
	o.BufferSize = size
	return o
}

// WithDropOnFull sets the drop-on-full behavior
func (o Options) WithDropOnFull(drop bool) Options {
	o.DropOnFull = drop
	return o
}

// WithStreamName sets the stream name
func (o Options) WithStreamName(name string) Options {
	o.StreamName = name
	return o
}

// WithStreamMaxAge sets the stream max age
func (o Options) WithStreamMaxAge(age time.Duration) Options {
	o.StreamMaxAge = age
	return o
}

// WithStreamMaxMsgs sets the stream max messages
func (o Options) WithStreamMaxMsgs(maxMsgs int64) Options {
	o.StreamMaxMsgs = maxMsgs
	return o
}

// WithPublishTimeout sets the publish timeout
func (o Options) WithPublishTimeout(timeout time.Duration) Options {
	o.PublishTimeout = timeout
	return o
}
