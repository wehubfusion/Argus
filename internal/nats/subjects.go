package nats

import "github.com/wehubfusion/Argus/pkg/event"

// Subject constants for observation events
const (
	// Stream name for observation events
	StreamName = "OBSERVATION"

	// Subject prefix for all observation events
	SubjectPrefix = "OBSERVE"

	// Workflow catalog events
	SubjectWorkflowPublished = "OBSERVE.WORKFLOW.PUBLISHED"

	// Run lifecycle events
	SubjectRunStarted = "OBSERVE.WORKFLOW.RUN.STARTED"
	SubjectRunEnded   = "OBSERVE.WORKFLOW.RUN.ENDED"

	// Plugin lifecycle events
	SubjectPluginStarted = "OBSERVE.WORKFLOW.PLUGIN.STARTED"
	SubjectPluginEnded   = "OBSERVE.WORKFLOW.PLUGIN.ENDED"
)

// GetSubjectForEventType returns the NATS subject for a given event type
func GetSubjectForEventType(eventType string) string {
	switch eventType {
	case event.TypeWorkflowPublished:
		return SubjectWorkflowPublished
	case event.TypeRunStarted:
		return SubjectRunStarted
	case event.TypeRunEnded:
		return SubjectRunEnded
	case event.TypePluginStarted:
		return SubjectPluginStarted
	case event.TypePluginEnded:
		return SubjectPluginEnded
	default:
		return ""
	}
}
