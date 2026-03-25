package tests

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/wehubfusion/Argus/pkg/event"
	"github.com/wehubfusion/Argus/pkg/observer"
	"go.uber.org/zap"
)

func TestNewObserver_Success(t *testing.T) {
	obs, err := observer.NewObserver(NewMockJetStream(), observer.DefaultOptions(), zap.NewNop())
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if obs == nil {
		t.Fatal("Expected observer to be non-nil")
	}
	obs.Close(context.Background())
}

func TestNewObserver_NilJetStream(t *testing.T) {
	obs, err := observer.NewObserver(nil, observer.DefaultOptions(), zap.NewNop())
	if err == nil {
		t.Fatal("Expected error for nil JetStream context")
	}
	if obs != nil {
		t.Fatal("Expected observer to be nil")
	}
}

func TestNewObserver_DefaultOptions(t *testing.T) {
	mockJS := NewMockJetStream()
	_, err := observer.NewObserver(mockJS, observer.Options{}, zap.NewNop())
	if err != nil {
		t.Fatalf("Expected no error with empty options, got: %v", err)
	}
	if len(mockJS.GetStreams()) == 0 {
		t.Error("Expected stream to be created with default options")
	}
}

func TestNewObserver_CustomOptions(t *testing.T) {
	mockJS := NewMockJetStream()
	opts := observer.DefaultOptions().WithStreamName("CUSTOM_STREAM")

	_, err := observer.NewObserver(mockJS, opts, zap.NewNop())
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if _, exists := mockJS.GetStreams()["CUSTOM_STREAM"]; !exists {
		t.Error("Expected custom stream name to be used")
	}
}

func TestObserver_Emit_Success(t *testing.T) {
	mockJS := NewMockJetStream()
	obs, err := observer.NewObserver(mockJS, observer.DefaultOptions(), zap.NewNop())
	if err != nil {
		t.Fatalf("Failed to create observer: %v", err)
	}
	defer obs.Close(context.Background())

	evt := event.New(event.TypeRunStarted).
		WithClient("client_123").
		WithWorkflow("wf_456").
		WithRun("run_789")

	if err := obs.Emit(context.Background(), evt); err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Emit is synchronous — message is available immediately
	messages := mockJS.GetPublishedMessages()
	if len(messages) != 1 {
		t.Fatalf("Expected 1 published message, got %d", len(messages))
	}

	var published event.Event
	if err := json.Unmarshal(messages[0].Data, &published); err != nil {
		t.Fatalf("Failed to unmarshal published event: %v", err)
	}
	if published.Type != event.TypeRunStarted {
		t.Errorf("Expected type %s, got %s", event.TypeRunStarted, published.Type)
	}
	if published.ClientID != "client_123" {
		t.Errorf("Expected client_id 'client_123', got %s", published.ClientID)
	}
}

func TestObserver_Emit_ClosedObserver(t *testing.T) {
	mockJS := NewMockJetStream()
	obs, err := observer.NewObserver(mockJS, observer.DefaultOptions(), zap.NewNop())
	if err != nil {
		t.Fatalf("Failed to create observer: %v", err)
	}

	obs.Close(context.Background())

	evt := event.New(event.TypeRunStarted).
		WithClient("client_123").
		WithWorkflow("wf_456").
		WithRun("run_789")

	err = obs.Emit(context.Background(), evt)
	if !errors.Is(err, observer.ErrObserverClosed) {
		t.Errorf("Expected ErrObserverClosed, got: %v", err)
	}
}

func TestObserver_Emit_PublishError(t *testing.T) {
	mockJS := NewMockJetStream()
	obs, err := observer.NewObserver(mockJS, observer.DefaultOptions(), zap.NewNop())
	if err != nil {
		t.Fatalf("Failed to create observer: %v", err)
	}
	defer obs.Close(context.Background())

	mockJS.SetPublishError(errors.New("nats: stream unavailable"))

	evt := event.New(event.TypeRunStarted).
		WithClient("client_123").
		WithWorkflow("wf_456").
		WithRun("run_789")

	err = obs.Emit(context.Background(), evt)
	if err == nil {
		t.Fatal("Expected error from publish failure, got nil")
	}
}

func TestObserver_Emit_InvalidEvent(t *testing.T) {
	obs, err := observer.NewObserver(NewMockJetStream(), observer.DefaultOptions(), zap.NewNop())
	if err != nil {
		t.Fatalf("Failed to create observer: %v", err)
	}
	defer obs.Close(context.Background())

	// Event with no ClientID should fail validation
	evt := &event.Event{
		Type: event.TypeRunStarted,
	}
	if err := obs.Emit(context.Background(), evt); err == nil {
		t.Error("Expected validation error for incomplete event, got nil")
	}
}

func TestObserver_Emit_ContextCancellation(t *testing.T) {
	obs, err := observer.NewObserver(NewMockJetStream(), observer.DefaultOptions(), zap.NewNop())
	if err != nil {
		t.Fatalf("Failed to create observer: %v", err)
	}
	defer obs.Close(context.Background())

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled

	evt := event.New(event.TypeRunStarted).
		WithClient("client_123").
		WithWorkflow("wf_456").
		WithRun("run_789")

	// The cancelled context is passed to publisher.Publish which honours it.
	// With a mock publisher the call may succeed before context is checked;
	// the important thing is that no panic occurs.
	_ = obs.Emit(ctx, evt)
}

func TestObserver_Emit_AutoPopulation(t *testing.T) {
	mockJS := NewMockJetStream()
	obs, err := observer.NewObserver(mockJS, observer.DefaultOptions(), zap.NewNop())
	if err != nil {
		t.Fatalf("Failed to create observer: %v", err)
	}
	defer obs.Close(context.Background())

	evt := &event.Event{
		Type:       event.TypeRunStarted,
		ClientID:   "client_123",
		WorkflowID: "wf_456",
		RunID:      "run_789",
	}

	if err := obs.Emit(context.Background(), evt); err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Fields should be auto-populated in place (synchronous, no race)
	if evt.ID == "" {
		t.Error("Expected ID to be auto-populated")
	}
	if evt.Timestamp.IsZero() {
		t.Error("Expected Timestamp to be auto-populated")
	}
	if evt.Version == "" {
		t.Error("Expected Version to be auto-populated")
	}
}

func TestObserver_Emit_Multiple(t *testing.T) {
	mockJS := NewMockJetStream()
	obs, err := observer.NewObserver(mockJS, observer.DefaultOptions(), zap.NewNop())
	if err != nil {
		t.Fatalf("Failed to create observer: %v", err)
	}
	defer obs.Close(context.Background())

	const count = 5
	for i := 0; i < count; i++ {
		evt := event.New(event.TypeRunStarted).
			WithClient("client_123").
			WithWorkflow("wf_456").
			WithRun("run_789")
		if err := obs.Emit(context.Background(), evt); err != nil {
			t.Fatalf("Emit %d failed: %v", i, err)
		}
	}

	if n := len(mockJS.GetPublishedMessages()); n != count {
		t.Errorf("Expected %d published messages, got %d", count, n)
	}
}

func TestObserver_Close_Idempotent(t *testing.T) {
	obs, err := observer.NewObserver(NewMockJetStream(), observer.DefaultOptions(), zap.NewNop())
	if err != nil {
		t.Fatalf("Failed to create observer: %v", err)
	}

	ctx := context.Background()
	if err := obs.Close(ctx); err != nil {
		t.Errorf("First Close failed: %v", err)
	}
	if err := obs.Close(ctx); err != nil {
		t.Errorf("Second Close failed: %v", err)
	}
	if err := obs.Close(ctx); err != nil {
		t.Errorf("Third Close failed: %v", err)
	}
}

func TestObserver_Close_BlocksNewEmits(t *testing.T) {
	mockJS := NewMockJetStream()
	obs, err := observer.NewObserver(mockJS, observer.DefaultOptions(), zap.NewNop())
	if err != nil {
		t.Fatalf("Failed to create observer: %v", err)
	}

	obs.Close(context.Background())

	// Any emit after close must return ErrObserverClosed, never publish
	for i := 0; i < 3; i++ {
		evt := event.New(event.TypeRunStarted).
			WithClient("c").WithWorkflow("w").WithRun("r")
		if err := obs.Emit(context.Background(), evt); !errors.Is(err, observer.ErrObserverClosed) {
			t.Errorf("Emit %d after close: expected ErrObserverClosed, got %v", i, err)
		}
	}
	if n := len(mockJS.GetPublishedMessages()); n != 0 {
		t.Errorf("Expected 0 published messages after close, got %d", n)
	}
}

func TestObserver_Close_ContextIgnored(t *testing.T) {
	obs, err := observer.NewObserver(NewMockJetStream(), observer.DefaultOptions(), zap.NewNop())
	if err != nil {
		t.Fatalf("Failed to create observer: %v", err)
	}

	// Close with an already-expired context — should still succeed (no drain needed)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(time.Millisecond)

	if err := obs.Close(ctx); err != nil {
		t.Errorf("Expected no error from Close with expired context, got: %v", err)
	}
}
