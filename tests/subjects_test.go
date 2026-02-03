package tests

import (
	"testing"

	"github.com/wehubfusion/Argus/internal/nats"
	"github.com/wehubfusion/Argus/pkg/event"
)

func TestGetSubjectForEventType_AllEventTypes(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		expected  string
	}{
		{
			name:      "WorkflowPublished",
			eventType: event.TypeWorkflowPublished,
			expected:  nats.SubjectWorkflowPublished,
		},
		{
			name:      "WorkflowPublished",
			eventType: event.TypeWorkflowPublished,
			expected:  nats.SubjectWorkflowPublished,
		},
		{
			name:      "RunStarted",
			eventType: event.TypeRunStarted,
			expected:  nats.SubjectRunStarted,
		},
		{
			name:      "RunEnded",
			eventType: event.TypeRunEnded,
			expected:  nats.SubjectRunEnded,
		},
		{
			name:      "PluginStarted",
			eventType: event.TypePluginStarted,
			expected:  nats.SubjectPluginStarted,
		},
		{
			name:      "PluginEnded",
			eventType: event.TypePluginEnded,
			expected:  nats.SubjectPluginEnded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := nats.GetSubjectForEventType(tt.eventType)
			if result != tt.expected {
				t.Errorf("Expected subject %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestGetSubjectForEventType_UnknownEventType(t *testing.T) {
	result := nats.GetSubjectForEventType("unknown.event.type")
	if result != "" {
		t.Errorf("Expected empty string for unknown event type, got %s", result)
	}
}

func TestGetSubjectForEventType_EmptyString(t *testing.T) {
	result := nats.GetSubjectForEventType("")
	if result != "" {
		t.Errorf("Expected empty string for empty event type, got %s", result)
	}
}

func TestGetSubjectForEventType_CaseSensitive(t *testing.T) {
	// Verify that event types are case-sensitive
	result := nats.GetSubjectForEventType("RUN.STARTED") // Wrong case
	if result != "" {
		t.Errorf("Expected empty string for wrong case, got %s", result)
	}

	// Correct case should work
	result = nats.GetSubjectForEventType(event.TypeRunStarted)
	if result != nats.SubjectRunStarted {
		t.Errorf("Expected subject for correct case, got %s", result)
	}
}
