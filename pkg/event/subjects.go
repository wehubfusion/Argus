// Package event defines observation event types, payloads, validation, and the NATS
// subject/stream contract. Event types have three levels: workflow catalog (Level 1),
// run lifecycle (Level 2), and plugin lifecycle (Level 3). The Version field is set on
// every event; consumers should check Version before parsing Data and skip or log unknown
// versions. The NATS contract (stream name, subject prefix, SubjectPatternAll) is defined
// here for both producers and consumers.

package event

// NATS stream and subject constants for observation events.
// Consumers (e.g. Athena) should subscribe using StreamName and SubjectPatternAll.
const (
	// StreamName is the JetStream stream name for observation events.
	StreamName = "OBSERVATION"

	// SubjectPrefix is the subject prefix for all observation events (e.g. "OBSERVE").
	SubjectPrefix = "OBSERVE"

	// SubjectPatternAll is the subscribe-all pattern for consumers ("OBSERVE.>").
	SubjectPatternAll = "OBSERVE.>"

	// SubjectWorkflowPublished is the subject for workflow catalog events.
	SubjectWorkflowPublished = "OBSERVE.WORKFLOW.PUBLISHED"

	// SubjectWorkflowStarted is the subject for workflow started events.
	SubjectWorkflowStarted = "OBSERVE.WORKFLOW.STARTED"

	// SubjectRunStarted is the subject for run started events.
	SubjectRunStarted = "OBSERVE.WORKFLOW.RUN.STARTED"

	// SubjectRunEnded is the subject for run ended events.
	SubjectRunEnded = "OBSERVE.WORKFLOW.RUN.ENDED"

	// SubjectPluginStarted is the subject for plugin started events.
	SubjectPluginStarted = "OBSERVE.WORKFLOW.PLUGIN.STARTED"

	// SubjectPluginEnded is the subject for plugin ended events.
	SubjectPluginEnded = "OBSERVE.WORKFLOW.PLUGIN.ENDED"

	// SubjectNodeTriggered is the subject for node trigger events.
	// These are emitted when an execution unit (and its embedded nodes)
	// is dispatched by an orchestrator.
	SubjectNodeTriggered = "OBSERVE.WORKFLOW.NODE.TRIGGERED"

	// SubjectNodeStarted is the subject for node started events.
	// These are emitted when a node begins execution on a worker.
	SubjectNodeStarted = "OBSERVE.WORKFLOW.NODE.STARTED"

	// SubjectNodeEnded is the subject for node ended events.
	// These are emitted when a node finishes execution on a worker.
	SubjectNodeEnded = "OBSERVE.WORKFLOW.NODE.ENDED"
)

// SubjectForEventType returns the NATS subject for a given event type.
// Returns empty string for unknown event types.
func SubjectForEventType(eventType string) string {
	switch eventType {
	case TypeWorkflowPublished:
		return SubjectWorkflowPublished
	case TypeWorkflowStarted:
		return SubjectWorkflowStarted
	case TypeRunStarted:
		return SubjectRunStarted
	case TypeRunEnded:
		return SubjectRunEnded
	case TypePluginStarted:
		return SubjectPluginStarted
	case TypePluginEnded:
		return SubjectPluginEnded
	case TypeNodeTriggered:
		return SubjectNodeTriggered
	case TypeNodeStarted:
		return SubjectNodeStarted
	case TypeNodeEnded:
		return SubjectNodeEnded
	default:
		return ""
	}
}
