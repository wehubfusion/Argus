package tests

import (
	"testing"

	"github.com/wehubfusion/Argus/pkg/event"
)

func TestSubjectForEventType_AllEventTypes(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		expected  string
	}{
		{
			name:      "WorkflowPublished",
			eventType: event.TypeWorkflowPublished,
			expected:  event.SubjectWorkflowPublished,
		},
		{
			name:      "RunStarted",
			eventType: event.TypeRunStarted,
			expected:  event.SubjectRunStarted,
		},
		{
			name:      "RunEnded",
			eventType: event.TypeRunEnded,
			expected:  event.SubjectRunEnded,
		},
		{
			name:      "PluginStarted",
			eventType: event.TypePluginStarted,
			expected:  event.SubjectPluginStarted,
		},
		{
			name:      "PluginEnded",
			eventType: event.TypePluginEnded,
			expected:  event.SubjectPluginEnded,
		},
		{
			name:      "NodeTriggered",
			eventType: event.TypeNodeTriggered,
			expected:  event.SubjectNodeTriggered,
		},
		{
			name:      "NodeStarted",
			eventType: event.TypeNodeStarted,
			expected:  event.SubjectNodeStarted,
		},
		{
			name:      "NodeEnded",
			eventType: event.TypeNodeEnded,
			expected:  event.SubjectNodeEnded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := event.SubjectForEventType(tt.eventType)
			if result != tt.expected {
				t.Errorf("Expected subject %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestSubjectForEventType_UnknownEventType(t *testing.T) {
	result := event.SubjectForEventType("unknown.event.type")
	if result != "" {
		t.Errorf("Expected empty string for unknown event type, got %s", result)
	}
}

func TestSubjectForEventType_EmptyString(t *testing.T) {
	result := event.SubjectForEventType("")
	if result != "" {
		t.Errorf("Expected empty string for empty event type, got %s", result)
	}
}

func TestSubjectForEventType_CaseSensitive(t *testing.T) {
	// Verify that event types are case-sensitive
	result := event.SubjectForEventType("RUN.STARTED") // Wrong case
	if result != "" {
		t.Errorf("Expected empty string for wrong case, got %s", result)
	}

	// Correct case should work
	result = event.SubjectForEventType(event.TypeRunStarted)
	if result != event.SubjectRunStarted {
		t.Errorf("Expected subject for correct case, got %s", result)
	}
}
