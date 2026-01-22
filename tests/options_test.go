package tests

import (
	"testing"
	"time"

	"github.com/wehubfusion/Argus/pkg/observer"
)

func TestDefaultOptions(t *testing.T) {
	opts := observer.DefaultOptions()

	if opts.BufferSize != 1000 {
		t.Errorf("Expected BufferSize 1000, got %d", opts.BufferSize)
	}
	if opts.DropOnFull != true {
		t.Errorf("Expected DropOnFull true, got %v", opts.DropOnFull)
	}
	if opts.StreamName != "OBSERVATION" {
		t.Errorf("Expected StreamName 'OBSERVATION', got %s", opts.StreamName)
	}
	if opts.StreamMaxAge != 30*24*time.Hour {
		t.Errorf("Expected StreamMaxAge 30 days, got %v", opts.StreamMaxAge)
	}
	if opts.StreamMaxMsgs != 1000000 {
		t.Errorf("Expected StreamMaxMsgs 1000000, got %d", opts.StreamMaxMsgs)
	}
	if opts.PublishTimeout != 5*time.Second {
		t.Errorf("Expected PublishTimeout 5s, got %v", opts.PublishTimeout)
	}
}

func TestOptions_WithBufferSize(t *testing.T) {
	opts := observer.DefaultOptions()
	newOpts := opts.WithBufferSize(500)

	// Original should be unchanged (immutability)
	if opts.BufferSize != 1000 {
		t.Errorf("Original BufferSize should be unchanged, got %d", opts.BufferSize)
	}

	// New should have updated value
	if newOpts.BufferSize != 500 {
		t.Errorf("Expected BufferSize 500, got %d", newOpts.BufferSize)
	}

	// Other fields should be unchanged
	if newOpts.DropOnFull != opts.DropOnFull {
		t.Error("DropOnFull should be unchanged")
	}
	if newOpts.StreamName != opts.StreamName {
		t.Error("StreamName should be unchanged")
	}
}

func TestOptions_WithDropOnFull(t *testing.T) {
	opts := observer.DefaultOptions()
	newOpts := opts.WithDropOnFull(false)

	// Original should be unchanged
	if opts.DropOnFull != true {
		t.Errorf("Original DropOnFull should be unchanged, got %v", opts.DropOnFull)
	}

	// New should have updated value
	if newOpts.DropOnFull != false {
		t.Errorf("Expected DropOnFull false, got %v", newOpts.DropOnFull)
	}
}

func TestOptions_WithStreamName(t *testing.T) {
	opts := observer.DefaultOptions()
	newOpts := opts.WithStreamName("CUSTOM_STREAM")

	// Original should be unchanged
	if opts.StreamName != "OBSERVATION" {
		t.Errorf("Original StreamName should be unchanged, got %s", opts.StreamName)
	}

	// New should have updated value
	if newOpts.StreamName != "CUSTOM_STREAM" {
		t.Errorf("Expected StreamName 'CUSTOM_STREAM', got %s", newOpts.StreamName)
	}
}

func TestOptions_WithStreamMaxAge(t *testing.T) {
	opts := observer.DefaultOptions()
	newMaxAge := 48 * time.Hour
	newOpts := opts.WithStreamMaxAge(newMaxAge)

	// Original should be unchanged
	if opts.StreamMaxAge != 30*24*time.Hour {
		t.Errorf("Original StreamMaxAge should be unchanged, got %v", opts.StreamMaxAge)
	}

	// New should have updated value
	if newOpts.StreamMaxAge != newMaxAge {
		t.Errorf("Expected StreamMaxAge %v, got %v", newMaxAge, newOpts.StreamMaxAge)
	}
}

func TestOptions_WithStreamMaxMsgs(t *testing.T) {
	opts := observer.DefaultOptions()
	newMaxMsgs := int64(2000000)
	newOpts := opts.WithStreamMaxMsgs(newMaxMsgs)

	// Original should be unchanged
	if opts.StreamMaxMsgs != 1000000 {
		t.Errorf("Original StreamMaxMsgs should be unchanged, got %d", opts.StreamMaxMsgs)
	}

	// New should have updated value
	if newOpts.StreamMaxMsgs != newMaxMsgs {
		t.Errorf("Expected StreamMaxMsgs %d, got %d", newMaxMsgs, newOpts.StreamMaxMsgs)
	}
}

func TestOptions_WithPublishTimeout(t *testing.T) {
	opts := observer.DefaultOptions()
	newTimeout := 10 * time.Second
	newOpts := opts.WithPublishTimeout(newTimeout)

	// Original should be unchanged
	if opts.PublishTimeout != 5*time.Second {
		t.Errorf("Original PublishTimeout should be unchanged, got %v", opts.PublishTimeout)
	}

	// New should have updated value
	if newOpts.PublishTimeout != newTimeout {
		t.Errorf("Expected PublishTimeout %v, got %v", newTimeout, newOpts.PublishTimeout)
	}
}

func TestOptions_Chaining(t *testing.T) {
	opts := observer.DefaultOptions().
		WithBufferSize(500).
		WithDropOnFull(false).
		WithStreamName("CUSTOM_STREAM").
		WithStreamMaxAge(48 * time.Hour).
		WithStreamMaxMsgs(2000000).
		WithPublishTimeout(10 * time.Second)

	// Verify all changes were applied
	if opts.BufferSize != 500 {
		t.Errorf("Expected BufferSize 500, got %d", opts.BufferSize)
	}
	if opts.DropOnFull != false {
		t.Errorf("Expected DropOnFull false, got %v", opts.DropOnFull)
	}
	if opts.StreamName != "CUSTOM_STREAM" {
		t.Errorf("Expected StreamName 'CUSTOM_STREAM', got %s", opts.StreamName)
	}
	if opts.StreamMaxAge != 48*time.Hour {
		t.Errorf("Expected StreamMaxAge 48h, got %v", opts.StreamMaxAge)
	}
	if opts.StreamMaxMsgs != 2000000 {
		t.Errorf("Expected StreamMaxMsgs 2000000, got %d", opts.StreamMaxMsgs)
	}
	if opts.PublishTimeout != 10*time.Second {
		t.Errorf("Expected PublishTimeout 10s, got %v", opts.PublishTimeout)
	}
}

func TestOptions_Immutability(t *testing.T) {
	original := observer.DefaultOptions()

	// Apply multiple changes
	modified := original.
		WithBufferSize(200).
		WithDropOnFull(false).
		WithStreamName("MODIFIED")

	// Original should be completely unchanged
	if original.BufferSize != 1000 {
		t.Error("Original BufferSize was modified")
	}
	if original.DropOnFull != true {
		t.Error("Original DropOnFull was modified")
	}
	if original.StreamName != "OBSERVATION" {
		t.Error("Original StreamName was modified")
	}

	// Modified should have all changes
	if modified.BufferSize != 200 {
		t.Error("Modified BufferSize not set correctly")
	}
	if modified.DropOnFull != false {
		t.Error("Modified DropOnFull not set correctly")
	}
	if modified.StreamName != "MODIFIED" {
		t.Error("Modified StreamName not set correctly")
	}
}
