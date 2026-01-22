package event

import "errors"

var (
	// ErrMissingEventID is returned when event ID is missing
	ErrMissingEventID = errors.New("event_id is required")

	// ErrMissingEventType is returned when event type is missing
	ErrMissingEventType = errors.New("event type is required")

	// ErrMissingClientID is returned when client_id is missing
	ErrMissingClientID = errors.New("client_id is required")

	// ErrMissingWorkflowID is returned when workflow_id is missing
	ErrMissingWorkflowID = errors.New("workflow_id is required")

	// ErrMissingRunID is returned when run_id is missing
	ErrMissingRunID = errors.New("run_id is required")

	// ErrMissingNodeID is returned when node_id is missing (required for plugin events)
	ErrMissingNodeID = errors.New("node_id is required for plugin events")
)
