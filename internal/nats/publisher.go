package nats

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/wehubfusion/Argus/pkg/event"
	"go.uber.org/zap"
)

// PublisherConfig holds configuration for the NATS publisher.
type PublisherConfig struct {
	StreamName     string
	StreamMaxAge   time.Duration
	StreamMaxMsgs  int64
	PublishTimeout time.Duration
}

// Publisher handles publishing messages to NATS JetStream.
// All methods are safe for concurrent use.
type Publisher struct {
	js     nats.JetStreamContext
	config PublisherConfig
	logger *zap.Logger
}

// NewPublisher creates a new Publisher and ensures the target stream exists.
func NewPublisher(js nats.JetStreamContext, config PublisherConfig, logger *zap.Logger) (*Publisher, error) {
	if js == nil {
		return nil, fmt.Errorf("jetstream context cannot be nil")
	}
	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	p := &Publisher{js: js, config: config, logger: logger}
	if err := p.ensureStream(); err != nil {
		return nil, fmt.Errorf("failed to ensure observation stream: %w", err)
	}
	return p, nil
}

// ensureStream creates the observation stream if it does not already exist.
func (p *Publisher) ensureStream() error {
	streamInfo, err := p.js.StreamInfo(p.config.StreamName)
	if err != nil && err != nats.ErrStreamNotFound {
		return fmt.Errorf("failed to check stream info: %w", err)
	}

	if streamInfo != nil {
		p.logger.Info("Observation stream already exists",
			zap.String("stream", p.config.StreamName),
			zap.Uint64("messages", streamInfo.State.Msgs))
		return nil
	}

	streamConfig := &nats.StreamConfig{
		Name:     p.config.StreamName,
		Subjects: []string{event.SubjectPatternAll},
		Storage:  nats.FileStorage,
		MaxAge:   p.config.StreamMaxAge,
		MaxMsgs:  p.config.StreamMaxMsgs,
		Replicas: 1,
	}
	if _, err := p.js.AddStream(streamConfig); err != nil {
		return fmt.Errorf("failed to create observation stream: %w", err)
	}

	p.logger.Info("Created observation stream",
		zap.String("stream", p.config.StreamName),
		zap.Strings("subjects", streamConfig.Subjects),
		zap.Duration("max_age", streamConfig.MaxAge),
		zap.Int64("max_msgs", streamConfig.MaxMsgs))
	return nil
}

// Publish publishes a message to JetStream with exactly-once deduplication support.
//
// It blocks until the server acknowledges receipt, the caller's context is cancelled,
// or PublishTimeout elapses — whichever comes first. No goroutines are spawned;
// backpressure is applied directly to the caller.
//
// nats.Context(publishCtx) passes the context into the JetStream ack-wait select
// loop (RequestMsgWithContext), so cancellation is handled natively by the NATS
// client rather than via a wrapper goroutine.
func (p *Publisher) Publish(ctx context.Context, subject, msgID string, data []byte) error {
	if subject == "" {
		return fmt.Errorf("publisher: subject cannot be empty")
	}
	if msgID == "" {
		return fmt.Errorf("publisher: message ID cannot be empty")
	}
	if len(data) == 0 {
		return fmt.Errorf("publisher: data cannot be empty")
	}

	// Fast-path: reject immediately if the caller's context is already done.
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("publisher: context already done: %w", err)
	}

	// Apply per-publish timeout as a tighter bound on the caller's context.
	// nats.Context(publishCtx) is mutually exclusive with nats.AckWait, so
	// we do NOT also pass an AckWait option.
	publishCtx, cancel := context.WithTimeout(ctx, p.config.PublishTimeout)
	defer cancel()

	ack, err := p.js.Publish(subject, data,
		nats.MsgId(msgID),
		nats.Context(publishCtx),
	)
	if err != nil {
		return fmt.Errorf("publisher: publish failed: %w", err)
	}

	p.logger.Debug("Published message",
		zap.String("msg_id", msgID),
		zap.String("subject", subject),
		zap.String("stream", ack.Stream),
		zap.Uint64("sequence", ack.Sequence))

	return nil
}
