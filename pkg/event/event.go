package event

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Event type constants
const (
	TypeWorkflowPublished   = "workflow.published"
	TypeWorkflowUnpublished = "workflow.unpublished"
	TypeWorkflowHealth      = "workflow.health.changed"
	TypeRunStarted          = "run.started"
	TypeRunEnded            = "run.ended"
	TypePluginStarted       = "plugin.started"
	TypePluginEnded         = "plugin.ended"
)

// Event is the universal observation event structure.
// Generic and service-agnostic, works for any producer service.
//
// Design Philosophy:
// - Producers emit minimal events: Only IDs (workflow_id, run_id, node_id) and execution data
// - Consumers can enrich events by fetching metadata (names, tags, icons) from catalog services
// - Data field: Type-specific payload as json.RawMessage for flexibility
//
// Field Requirements by Level:
// - Level 1 (Workflow Catalog): Requires client_id, workflow_id (no run_id)
// - Level 2 (Run Lifecycle): Requires client_id, workflow_id, run_id
// - Level 3 (Plugin Lifecycle): Requires client_id, workflow_id, run_id, node_id
type Event struct {
	// === Routing & Identity ===
	ID        string    `json:"id"`        // Unique ID (for JetStream deduplication)
	Type      string    `json:"type"`      // Event type (e.g., "run.started", "plugin.ended")
	Version   string    `json:"v"`         // Schema version
	Timestamp time.Time `json:"timestamp"` // When event occurred

	// === Context (correlation & multi-tenancy) ===
	ClientID   string `json:"client_id"`
	WorkflowID string `json:"workflow_id,omitempty"`
	RunID      string `json:"run_id,omitempty"`
	NodeID     string `json:"node_id,omitempty"`

	// === Event-Specific Data ===
	// Raw JSON - consumer parses based on Type
	Data json.RawMessage `json:"data,omitempty"`
}

// New creates a new event with generated ID and timestamp.
func New(eventType string) *Event {
	return &Event{
		ID:        uuid.New().String(),
		Type:      eventType,
		Version:   "v1",
		Timestamp: time.Now(),
	}
}

// === Fluent Builder Methods ===

// WithClient sets the client_id field.
func (e *Event) WithClient(clientID string) *Event {
	e.ClientID = clientID
	return e
}

// WithWorkflow sets the workflow_id field.
func (e *Event) WithWorkflow(workflowID string) *Event {
	e.WorkflowID = workflowID
	return e
}

// WithRun sets the run_id field.
func (e *Event) WithRun(runID string) *Event {
	e.RunID = runID
	return e
}

// WithNode sets the node_id field.
func (e *Event) WithNode(nodeID string) *Event {
	e.NodeID = nodeID
	return e
}

// WithData sets the data field from any struct/map.
// Marshals to JSON internally.
func (e *Event) WithData(data any) *Event {
	if data == nil {
		e.Data = nil
		return e
	}
	bytes, err := json.Marshal(data)
	if err != nil {
		// Store error as data (fail gracefully)
		e.Data = json.RawMessage(`{"_marshal_error":"` + err.Error() + `"}`)
		return e
	}
	e.Data = bytes
	return e
}

// WithRawData sets data directly from raw JSON bytes.
func (e *Event) WithRawData(data json.RawMessage) *Event {
	e.Data = data
	return e
}

// === Serialization ===

// Bytes serializes the event to JSON.
func (e *Event) Bytes() ([]byte, error) {
	return json.Marshal(e)
}

// ParseData unmarshals the Data field into the provided struct.
func (e *Event) ParseData(dest any) error {
	if e.Data == nil {
		return nil
	}
	return json.Unmarshal(e.Data, dest)
}

// === Validation ===

// Validate checks if the event has required fields.
// Validation is level-aware:
// - Level 1 (Workflow Catalog): Only requires client_id and workflow_id (no run_id)
// - Level 2 (Run Lifecycle): Requires client_id, workflow_id, and run_id
// - Level 3 (Plugin Lifecycle): Requires client_id, workflow_id, run_id, and node_id
func (e *Event) Validate() error {
	if e.ID == "" {
		return ErrMissingEventID
	}
	if e.Type == "" {
		return ErrMissingEventType
	}
	if e.ClientID == "" {
		return ErrMissingClientID
	}

	// Level-based validation
	switch e.Type {
	case TypeWorkflowPublished, TypeWorkflowUnpublished, TypeWorkflowHealth:
		if e.WorkflowID == "" {
			return ErrMissingWorkflowID
		}

	case TypeRunStarted, TypeRunEnded:
		if e.WorkflowID == "" {
			return ErrMissingWorkflowID
		}
		if e.RunID == "" {
			return ErrMissingRunID
		}

	case TypePluginStarted, TypePluginEnded:
		if e.WorkflowID == "" {
			return ErrMissingWorkflowID
		}
		if e.RunID == "" {
			return ErrMissingRunID
		}
		if e.NodeID == "" {
			return ErrMissingNodeID
		}
	}

	return nil
}

// === Data Schema Structs ===
// These are documented structs for Data payloads.
// Event producers use these to create type-safe data.
// Event consumers use these to parse data.

// RunStartedData is the data payload for run.started events.
type RunStartedData struct {
	// TODO: Add fields here
}

// RunEndedData is the data payload for run.ended events.
type RunEndedData struct {
	// TODO: Add fields here
}

// PluginStartedData is the data payload for plugin.started events.
type PluginStartedData struct {
	// TODO: Add fields here
}

// PluginEndedData is the data payload for plugin.ended events.
type PluginEndedData struct {
	// TODO: Add fields here
}

// WorkflowHealthData is the data payload for workflow.health.changed events.
type WorkflowHealthData struct {
	// TODO: Add fields here
}
