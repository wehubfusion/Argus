package tests

import (
	"testing"
	"time"

	"github.com/wehubfusion/Argus/pkg/observer"
)

func TestDefaultOptions(t *testing.T) {
	opts := observer.DefaultOptions()

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

func TestOptions_WithStreamName(t *testing.T) {
	opts := observer.DefaultOptions()
	newOpts := opts.WithStreamName("CUSTOM_STREAM")

	if opts.StreamName != "OBSERVATION" {
		t.Errorf("Original StreamName should be unchanged, got %s", opts.StreamName)
	}
	if newOpts.StreamName != "CUSTOM_STREAM" {
		t.Errorf("Expected StreamName 'CUSTOM_STREAM', got %s", newOpts.StreamName)
	}
}

func TestOptions_WithStreamMaxAge(t *testing.T) {
	opts := observer.DefaultOptions()
	newMaxAge := 48 * time.Hour
	newOpts := opts.WithStreamMaxAge(newMaxAge)

	if opts.StreamMaxAge != 30*24*time.Hour {
		t.Errorf("Original StreamMaxAge should be unchanged, got %v", opts.StreamMaxAge)
	}
	if newOpts.StreamMaxAge != newMaxAge {
		t.Errorf("Expected StreamMaxAge %v, got %v", newMaxAge, newOpts.StreamMaxAge)
	}
}

func TestOptions_WithStreamMaxMsgs(t *testing.T) {
	opts := observer.DefaultOptions()
	newMaxMsgs := int64(2000000)
	newOpts := opts.WithStreamMaxMsgs(newMaxMsgs)

	if opts.StreamMaxMsgs != 1000000 {
		t.Errorf("Original StreamMaxMsgs should be unchanged, got %d", opts.StreamMaxMsgs)
	}
	if newOpts.StreamMaxMsgs != newMaxMsgs {
		t.Errorf("Expected StreamMaxMsgs %d, got %d", newMaxMsgs, newOpts.StreamMaxMsgs)
	}
}

func TestOptions_WithPublishTimeout(t *testing.T) {
	opts := observer.DefaultOptions()
	newTimeout := 10 * time.Second
	newOpts := opts.WithPublishTimeout(newTimeout)

	if opts.PublishTimeout != 5*time.Second {
		t.Errorf("Original PublishTimeout should be unchanged, got %v", opts.PublishTimeout)
	}
	if newOpts.PublishTimeout != newTimeout {
		t.Errorf("Expected PublishTimeout %v, got %v", newTimeout, newOpts.PublishTimeout)
	}
}

func TestOptions_Chaining(t *testing.T) {
	opts := observer.DefaultOptions().
		WithStreamName("CUSTOM_STREAM").
		WithStreamMaxAge(48 * time.Hour).
		WithStreamMaxMsgs(2000000).
		WithPublishTimeout(10 * time.Second)

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
	modified := original.WithStreamName("MODIFIED")

	if original.StreamName != "OBSERVATION" {
		t.Error("Original StreamName was modified")
	}
	if modified.StreamName != "MODIFIED" {
		t.Error("Modified StreamName not set correctly")
	}
}
