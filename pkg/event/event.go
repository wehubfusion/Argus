package event

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Event type constants
const (
	TypeWorkflowPublished = "workflow.published"
	TypeWorkflowStarted   = "workflow.started"
	TypeRunStarted        = "run.started"
	TypeRunEnded          = "run.ended"
	TypePluginStarted     = "plugin.started"
	TypePluginEnded       = "plugin.ended"
	// TypeNodeTriggered is a Level 3 node lifecycle event emitted when an
	// execution unit (and its embedded nodes) is dispatched by an orchestrator
	// such as Zeus. Payload: TriggerNode.
	TypeNodeTriggered = "node.triggered"
	// TypeNodeStarted is a Level 3 node lifecycle event emitted when a node
	// begins execution on a worker. Payload: StartNode.
	TypeNodeStarted = "node.started"
	// TypeNodeEnded is a Level 3 node lifecycle event emitted when a node
	// finishes execution on a worker. Payload: EndNode.
	TypeNodeEnded = "node.ended"
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
	Version   string    `json:"v"`         // Schema version; consumers should check before parsing Data and skip or log unknown versions
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
	case TypeWorkflowPublished:
		if e.WorkflowID == "" {
			return ErrMissingWorkflowID
		}

	case TypeWorkflowStarted, TypeRunStarted, TypeRunEnded:
		if e.WorkflowID == "" {
			return ErrMissingWorkflowID
		}
		if e.RunID == "" {
			return ErrMissingRunID
		}

	case TypePluginStarted, TypePluginEnded, TypeNodeTriggered, TypeNodeStarted, TypeNodeEnded:
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

// WorkflowPublish is the data payload for workflow.published events.
// client_id and workflow_id are on the event envelope; timestamp is event.Timestamp.
type WorkflowPublish struct {
	Action       string `json:"action"` // "publish" | "unpublish"
	Status       string `json:"status"` // "success" | "failed" | "started"
	StartedAt    int64  `json:"started_at"`
	EndedAt      int64  `json:"ended_at"`
	HasError     bool   `json:"has_error"`
	ErrorMessage string `json:"error_message,omitempty"`
}

type TriggerWorkflow struct {
	WorkflowID   string   `json:"workflow_id"`
	RunID        string   `json:"run_id"`
	ClientID     string   `json:"client_id"`
	Type         string   `json:"type"`
	Payload      *Payload `json:"payload"`
	StartedAt    int64    `json:"started_at"`
	EndedAt      int64    `json:"ended_at"`
	HasError     bool     `json:"has_error"`
	ErrorMessage string   `json:"error_message,omitempty"`
}

// TriggerInfo provides context about what initiated a run (e.g., http/manual) and payload details.
type TriggerInfo struct {
	Type     string `json:"type"`               // "http" | "manual" | "kafka" | ...
	HasData  bool   `json:"has_data"`           // Whether trigger had any payload
	DataSize int    `json:"data_size"`          // Inline payload size (bytes) when applicable
	BlobURL  string `json:"blob_url,omitempty"` // Blob URL if payload was stored externally
}

// RunStartedData is the data payload for run.started events emitted when execution begins.
// It is distinct from TriggerWorkflow (also sent with run.started) which represents trigger enqueueing.
type RunStartedData struct {
	TotalNodes  int          `json:"total_nodes"`
	TriggerInfo *TriggerInfo `json:"trigger_info,omitempty"`
}

// RunEndedData is the data payload for run.ended events.
type RunEndedData struct {
	Status       string `json:"status"` // "completed" | "failed" | "partial" | ...
	TotalNodes   int    `json:"total_nodes"`
	SuccessNodes int    `json:"success_nodes"`
	FailedNodes  int    `json:"failed_nodes"`
	SkippedNodes int    `json:"skipped_nodes"`
	QueueLength  int    `json:"queue_length"`
}

type StartWorkflow struct {
	WorkflowID string `json:"workflow_id"`
	RunID      string `json:"run_id"`
	ClientID   string `json:"client_id"`
	StartedAt  int64  `json:"started_at"`
}

type EndWorkflow struct {
	WorkflowID   string `json:"workflow_id"`
	RunID        string `json:"run_id"`
	ClientID     string `json:"client_id"`
	EndedAt      int64  `json:"ended_at"`
	HasError     bool   `json:"has_error"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// TriggerNode is the data payload for node.triggered events.
// It captures when an execution unit (and optionally its embedded nodes)
// is dispatched by an orchestrator. The identifying fields
// (client_id, workflow_id, run_id, node_id) are also present on the
// Event envelope; they are duplicated here for convenience for consumers.
type TriggerNode struct {
	WorkflowID string   `json:"workflow_id"`
	RunID      string   `json:"run_id"`
	ClientID   string   `json:"client_id"`
	NodeID     string   `json:"node_id"`
	Label      string   `json:"label,omitempty"` // Human-readable node label from execution plan (e.g. "ESR", "Append to User.csv")
	Type       string   `json:"type"`
	Payload    *Payload `json:"payload"`
	StartedAt  int64    `json:"started_at"`
}

type StartNode struct {
	WorkflowID string   `json:"workflow_id"`
	RunID      string   `json:"run_id"`
	ClientID   string   `json:"client_id"`
	NodeID     string   `json:"node_id"`
	StartedAt  int64    `json:"started_at"`
	Input      *Payload `json:"input"`
}
type EndNode struct {
	WorkflowID   string   `json:"workflow_id"`
	RunID        string   `json:"run_id"`
	ClientID     string   `json:"client_id"`
	NodeID       string   `json:"node_id"`
	Label        string   `json:"label,omitempty"`      // Human-readable node label from execution plan (e.g. "ESR", "Auth Rule")
	StartedAt    int64    `json:"started_at,omitempty"` // Unix ms when node execution started (optional; derived from result timestamp − duration when available)
	EndedAt      int64    `json:"ended_at"`
	Output       *Payload `json:"output"`
	HasError     bool     `json:"has_error"`
	ErrorMessage string   `json:"error_message,omitempty"`
}

// BlobRef represents a blob reference for observation payloads (used by plugin events).
type BlobRef struct {
	URL       string `json:"url"`
	SizeBytes int64  `json:"size_bytes"`
}

// PayloadInfo represents inline or blob payloads for plugin lifecycle events.
// InlineData is raw JSON so UIs can display it without base64 decoding.
type PayloadInfo struct {
	InlineData    json.RawMessage `json:"inline_data,omitempty"`
	BlobReference *BlobRef        `json:"blob_reference,omitempty"`
}

// PluginStartedData is the data payload for plugin.started events.
type PluginStartedData struct {
	ExecutionID    string       `json:"execution_id"`
	PluginType     string       `json:"plugin_type"`
	Label          string       `json:"label"`
	ExecutionOrder int          `json:"execution_order"`
	StartedAt      int64        `json:"started_at"`
	InputPayload   *PayloadInfo `json:"input_payload,omitempty"`
}

// PluginEndedData is the data payload for plugin.ended events.
type PluginEndedData struct {
	ExecutionID   string       `json:"execution_id"`
	Status        string       `json:"status"` // "success" | "failed" | "skipped"
	EndedAt       int64        `json:"ended_at"`
	OutputPayload *PayloadInfo `json:"output_payload,omitempty"`
	HasError      bool         `json:"has_error"`
	ErrorMessage  string       `json:"error_message,omitempty"`
}

type Payload struct {
	InlineData    []byte         `json:"inline_data"`
	BlobReference *BlobReference `json:"blob_reference"`
}

type BlobReference struct {
	URL  string `json:"url"`
	Size int64  `json:"size"`
}
