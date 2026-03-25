package observer

import (
	"time"

	"github.com/wehubfusion/Argus/pkg/event"
)

// Options configures the Observer behavior
type Options struct {
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
		StreamName:     event.StreamName,
		StreamMaxAge:   30 * 24 * time.Hour, // 30 days
		StreamMaxMsgs:  1000000,
		PublishTimeout: 5 * time.Second,
	}
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
