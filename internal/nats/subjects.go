package nats

import "github.com/wehubfusion/Argus/pkg/event"

// Subject constants for observation events
const (
	// Stream name for observation events
	StreamName = "OBSERVATION"

	// Subject prefix for all observation events
	SubjectPrefix = "OBSERVE"

	// Workflow catalog events (Level 1)
	SubjectWorkflowPublished     = "OBSERVE.WORKFLOW.PUBLISHED"
	SubjectWorkflowUnpublished   = "OBSERVE.WORKFLOW.UNPUBLISHED"
	SubjectWorkflowHealthChanged = "OBSERVE.WORKFLOW.HEALTH"

	// Run lifecycle events (Level 2)
	SubjectRunStarted = "OBSERVE.RUN.STARTED"
	SubjectRunEnded   = "OBSERVE.RUN.ENDED"

	// Plugin lifecycle events (Level 3)
	SubjectPluginStarted = "OBSERVE.PLUGIN.STARTED"
	SubjectPluginEnded   = "OBSERVE.PLUGIN.ENDED"
)

// GetSubjectForEventType returns the NATS subject for a given event type
func GetSubjectForEventType(eventType string) string {
	switch eventType {
	case event.TypeWorkflowPublished:
		return SubjectWorkflowPublished
	case event.TypeWorkflowUnpublished:
		return SubjectWorkflowUnpublished
	case event.TypeWorkflowHealth:
		return SubjectWorkflowHealthChanged
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
