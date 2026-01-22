package nats

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// PublisherConfig holds configuration for the NATS publisher
type PublisherConfig struct {
	StreamName     string
	StreamMaxAge   time.Duration
	StreamMaxMsgs  int64
	PublishTimeout time.Duration
}

// Publisher handles publishing messages to NATS JetStream
type Publisher struct {
	js          nats.JetStreamContext
	config      PublisherConfig
	logger      *zap.Logger
	streamSetup bool
}

// NewPublisher creates a new NATS publisher
func NewPublisher(js nats.JetStreamContext, config PublisherConfig, logger *zap.Logger) (*Publisher, error) {
	if js == nil {
		return nil, fmt.Errorf("jetstream context cannot be nil")
	}
	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	p := &Publisher{
		js:     js,
		config: config,
		logger: logger,
	}

	// Ensure the observation stream exists
	if err := p.ensureStream(); err != nil {
		return nil, fmt.Errorf("failed to ensure observation stream: %w", err)
	}

	return p, nil
}

// ensureStream creates the OBSERVATION stream if it doesn't exist
func (p *Publisher) ensureStream() error {
	if p.streamSetup {
		return nil
	}

	// Check if stream already exists
	streamInfo, err := p.js.StreamInfo(p.config.StreamName)
	if err != nil && err != nats.ErrStreamNotFound {
		return fmt.Errorf("failed to check stream info: %w", err)
	}

	if streamInfo != nil {
		// Stream already exists
		p.logger.Info("Observation stream already exists",
			zap.String("stream", p.config.StreamName),
			zap.Uint64("messages", streamInfo.State.Msgs))
		p.streamSetup = true
		return nil
	}

	// Create stream with custom config
	streamConfig := &nats.StreamConfig{
		Name:     p.config.StreamName,
		Subjects: []string{fmt.Sprintf("%s.>", SubjectPrefix)}, // SubjectPrefix from subjects.go
		Storage:  nats.FileStorage,
		MaxAge:   p.config.StreamMaxAge,
		MaxMsgs:  p.config.StreamMaxMsgs,
		Replicas: 1,
	}

	_, err = p.js.AddStream(streamConfig)
	if err != nil {
		return fmt.Errorf("failed to create observation stream: %w", err)
	}

	p.logger.Info("Created observation stream",
		zap.String("stream", p.config.StreamName),
		zap.Strings("subjects", streamConfig.Subjects),
		zap.Duration("max_age", streamConfig.MaxAge),
		zap.Int64("max_msgs", streamConfig.MaxMsgs))

	p.streamSetup = true
	return nil
}

// Publish publishes a message to JetStream with deduplication support
// subject: The NATS subject to publish to
// msgID: The message ID for deduplication
// data: The message payload as raw bytes
func (p *Publisher) Publish(ctx context.Context, subject, msgID string, data []byte) error {
	if subject == "" {
		return fmt.Errorf("subject cannot be empty")
	}
	if msgID == "" {
		return fmt.Errorf("message ID cannot be empty")
	}
	if len(data) == 0 {
		return fmt.Errorf("data cannot be empty")
	}

	// Publish with deduplication support
	publishOpts := []nats.PubOpt{
		nats.MsgId(msgID),
	}

	// Create context with timeout
	publishCtx, cancel := context.WithTimeout(ctx, p.config.PublishTimeout)
	defer cancel()

	// Publish in goroutine to respect context
	type publishResult struct {
		ack *nats.PubAck
		err error
	}
	resultCh := make(chan publishResult, 1)
	go func() {
		ack, err := p.js.Publish(subject, data, publishOpts...)
		resultCh <- publishResult{ack: ack, err: err}
	}()

	var result publishResult
	select {
	case <-publishCtx.Done():
		p.logger.Warn("Publish cancelled",
			zap.String("msg_id", msgID),
			zap.String("subject", subject),
			zap.Error(publishCtx.Err()))
		return fmt.Errorf("publish cancelled: %w", publishCtx.Err())
	case result = <-resultCh:
	}

	// Handle publish error (nack scenario)
	if result.err != nil {
		p.logger.Error("Failed to publish message (nack)",
			zap.String("msg_id", msgID),
			zap.String("subject", subject),
			zap.Error(result.err))
		return fmt.Errorf("failed to publish message to JetStream: %w", result.err)
	}

	// Verify PubAck was received (ack scenario)
	if result.ack == nil {
		p.logger.Error("Publish returned nil PubAck (nack)",
			zap.String("msg_id", msgID),
			zap.String("subject", subject))
		return fmt.Errorf("publish returned nil PubAck - message not acknowledged")
	}

	// Verify PubAck contains valid stream information
	if result.ack.Stream == "" {
		p.logger.Warn("Publish returned PubAck with empty stream",
			zap.String("msg_id", msgID),
			zap.String("subject", subject))
		// This is unusual but not necessarily an error - log and continue
	}

	// Log successful ack with sequence number
	p.logger.Debug("Published message (ack)",
		zap.String("msg_id", msgID),
		zap.String("subject", subject),
		zap.String("stream", result.ack.Stream),
		zap.Uint64("sequence", result.ack.Sequence))

	return nil
}
