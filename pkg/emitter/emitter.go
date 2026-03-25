package emitter

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/wehubfusion/Argus/pkg/event"
	"github.com/wehubfusion/Argus/pkg/observer"
	"go.uber.org/zap"
)

// PathContext describes where a payload belongs in monitoring blob storage.
// Used to build deterministic blob paths for node output and input events.
type PathContext struct {
	ClientID   string
	ProjectID  string
	WorkflowID string
	RunID      string
	NodeID     string
	// IsInput indicates whether this payload is a node input (true) or output (false).
	IsInput bool
}

// BlobUploader uploads bytes to blob storage and returns the resulting URL.
// Use NewAzureBlobUploader for Azure Blob Storage, or inject a custom implementation
// (e.g. an adapter for Icarus storage.BlobStorageClient).
type BlobUploader interface {
	Upload(ctx context.Context, path string, data []byte, metadata map[string]string) (url string, size int64, err error)
}

// DefaultMaxInlineBytes is the default threshold (500KB) for inline payloads.
// Matches Athena and Icarus resolver defaults. Payloads larger than this are
// uploaded to blob storage and a BlobReference is returned.
const DefaultMaxInlineBytes = 512000

// PayloadOptions configures PreparePayload behavior.
type PayloadOptions struct {
	// MaxInlineBytes is the size threshold in bytes. Payloads larger than this
	// are uploaded to blob storage. Defaults to DefaultMaxInlineBytes when zero.
	MaxInlineBytes int
}

// PreparePayload applies the size threshold and returns either inline data or
// a blob reference for node output and input events. If data exceeds MaxInlineBytes
// and uploader is provided, it uploads to the monitoring path and returns a
// BlobReference. Otherwise returns InlineData.
//
// Use for both lifecycle payload directions; set PathContext.IsInput
// accordingly for correct blob path suffixes.
//
// Returns nil, nil when data is empty. Returns an error only when upload fails;
// callers may fall back to inline with logging.
func PreparePayload(ctx context.Context, data []byte, pathCtx PathContext, uploader BlobUploader, opts *PayloadOptions) (*event.Payload, error) {
	if len(data) == 0 {
		return nil, nil
	}

	maxInline := DefaultMaxInlineBytes
	if opts != nil && opts.MaxInlineBytes > 0 {
		maxInline = opts.MaxInlineBytes
	}

	// Below threshold or no uploader: return inline
	if len(data) <= maxInline || uploader == nil {
		return &event.Payload{
			InlineData:    data,
			BlobReference: nil,
		}, nil
	}

	blobPath := BuildMonitoringPath(pathCtx)
	if blobPath == "" {
		return &event.Payload{
			InlineData:    data,
			BlobReference: nil,
		}, fmt.Errorf("emitter: missing identifiers for blob path (client_id/project_id/workflow_id/run_id/node_id)")
	}

	metadata := map[string]string{
		"client_id":   pathCtx.ClientID,
		"project_id":  pathCtx.ProjectID,
		"workflow_id": pathCtx.WorkflowID,
		"run_id":      pathCtx.RunID,
		"node_id":     pathCtx.NodeID,
		"direction":   "output",
	}
	if pathCtx.IsInput {
		metadata["direction"] = "input"
	}

	url, size, err := uploader.Upload(ctx, blobPath, data, metadata)
	if err != nil {
		return &event.Payload{
			InlineData:    data,
			BlobReference: nil,
		}, fmt.Errorf("emitter: failed to upload monitoring blob: %w", err)
	}

	return &event.Payload{
		InlineData: nil,
		BlobReference: &event.BlobReference{
			URL:  url,
			Size: size,
		},
	}, nil
}

// BuildMonitoringPath returns the blob path for monitoring payloads.
// Mirrors Athena's path convention. Returns empty string if required
// identifiers are missing. For output: {node_id}.json; for input: {node_id}-input.json.
func BuildMonitoringPath(pathCtx PathContext) string {
	if pathCtx.ClientID == "" || pathCtx.ProjectID == "" || pathCtx.WorkflowID == "" || pathCtx.RunID == "" || pathCtx.NodeID == "" {
		return ""
	}

	suffix := pathCtx.NodeID + ".json"
	if pathCtx.IsInput {
		suffix = pathCtx.NodeID + "-input.json"
	}

	return fmt.Sprintf("monitoring/%s/%s/%s/%s/%s",
		pathCtx.ClientID,
		pathCtx.ProjectID,
		pathCtx.WorkflowID,
		pathCtx.RunID,
		suffix,
	)
}

// NodeEndEmitter emits node.ended observation events with output payload.
// Implementations are best-effort and must not panic.
type NodeEndEmitter interface {
	EmitNodeEnd(ctx context.Context, params NodeEndEmitParams) error
}

// NodeEndEmitParams contains all data needed to emit terminal node state.
type NodeEndEmitParams struct {
	ClientID      string
	ProjectID     string
	WorkflowID    string
	RunID         string
	NodeID        string
	Label         string
	Output        interface{}
	HasError      bool
	ErrorMessage  string
	ContainsNodes []string
}

// ArgusNodeEndEmitter implements NodeEndEmitter using Argus observer.
type ArgusNodeEndEmitter struct {
	observer observer.Observer
	uploader BlobUploader
	logger   *zap.Logger
}

// NewArgusNodeEndEmitter creates an emitter. observer and uploader may be nil; emission will no-op or fall back to inline.
func NewArgusNodeEndEmitter(
	obs observer.Observer,
	uploader BlobUploader,
	logger *zap.Logger,
) *ArgusNodeEndEmitter {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &ArgusNodeEndEmitter{
		observer: obs,
		uploader: uploader,
		logger:   logger,
	}
}

// EmitNodeEnd emits a node.ended event with output payload. Best-effort; logs errors, never panics.
func (e *ArgusNodeEndEmitter) EmitNodeEnd(ctx context.Context, params NodeEndEmitParams) error {
	if e == nil || e.observer == nil {
		return nil
	}
	if params.ClientID == "" || params.WorkflowID == "" || params.RunID == "" || params.NodeID == "" {
		e.logger.Debug("skipping node.ended emit due to missing context",
			zap.String("workflow_id", params.WorkflowID),
			zap.String("run_id", params.RunID),
			zap.String("node_id", params.NodeID),
		)
		return nil
	}

	// Marshal output as-is (no label wrapping)
	jsonBytes, err := json.Marshal(params.Output)
	if err != nil {
		e.logger.Warn("failed to marshal node output for observation",
			zap.String("node_id", params.NodeID),
			zap.Error(err),
		)
		return fmt.Errorf("emitter: marshal node.ended output (node_id=%s): %w", params.NodeID, err)
	}

	pathCtx := PathContext{
		ClientID:   params.ClientID,
		ProjectID:  params.ProjectID,
		WorkflowID: params.WorkflowID,
		RunID:      params.RunID,
		NodeID:     params.NodeID,
		IsInput:    false,
	}

	payload, prepErr := PreparePayload(ctx, jsonBytes, pathCtx, e.uploader, nil)
	if prepErr != nil {
		e.logger.Warn("PreparePayload failed, using inline fallback",
			zap.String("node_id", params.NodeID),
			zap.Error(prepErr),
		)
		// Fallback: inline
		payload = &event.Payload{
			InlineData:    jsonBytes,
			BlobReference: nil,
		}
	}

	if payload == nil {
		return nil
	}

	evt := event.New(event.TypeNodeEnded).
		WithClient(params.ClientID).
		WithWorkflow(params.WorkflowID).
		WithRun(params.RunID).
		WithNode(params.NodeID).
		WithData(&event.EndNode{
			WorkflowID:    params.WorkflowID,
			RunID:         params.RunID,
			ClientID:      params.ClientID,
			ProjectID:     params.ProjectID,
			NodeID:        params.NodeID,
			Label:         params.Label,
			EndedAt:       time.Now().UnixMilli(),
			Output:        payload,
			HasError:      params.HasError,
			ErrorMessage:  params.ErrorMessage,
			ContainsNodes: params.ContainsNodes,
		})

	if err := e.observer.Emit(ctx, evt); err != nil {
		e.logger.Warn("failed to emit node.ended observation event",
			zap.String("workflow_id", params.WorkflowID),
			zap.String("run_id", params.RunID),
			zap.String("node_id", params.NodeID),
			zap.Error(err),
		)
		return fmt.Errorf("emitter: emit node.ended observation (node_id=%s): %w", params.NodeID, err)
	}

	if prepErr != nil {
		// Event emitted successfully, but we fell back to inline due to payload preparation/upload failure.
		return fmt.Errorf("emitter: emitted node.ended with inline fallback (node_id=%s): %w", params.NodeID, prepErr)
	}
	return nil
}

// NodeStartEmitter emits node.started observation events with resolved payload.
// Implementations are best-effort and must not panic.
type NodeStartEmitter interface {
	EmitNodeStart(ctx context.Context, params NodeStartEmitParams) error
}

// NodeStartEmitParams contains all data needed to emit a node.started event.
type NodeStartEmitParams struct {
	ClientID   string
	ProjectID  string
	WorkflowID string
	RunID      string
	NodeID     string
	Label      string // Human-readable node label (e.g. from execution plan)
	Input      []byte
}

// ArgusNodeStartEmitter implements NodeStartEmitter using Argus observer.
type ArgusNodeStartEmitter struct {
	observer observer.Observer
	uploader BlobUploader
	logger   *zap.Logger
}

// NewArgusNodeStartEmitter creates an emitter. observer and uploader may be nil; emission will no-op or fall back to inline.
func NewArgusNodeStartEmitter(
	obs observer.Observer,
	uploader BlobUploader,
	logger *zap.Logger,
) *ArgusNodeStartEmitter {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &ArgusNodeStartEmitter{
		observer: obs,
		uploader: uploader,
		logger:   logger,
	}
}

// EmitNodeStart emits a node.started event with resolved payload. Best-effort; logs errors, never panics.
func (e *ArgusNodeStartEmitter) EmitNodeStart(ctx context.Context, params NodeStartEmitParams) error {
	if e == nil || e.observer == nil {
		return nil
	}
	if params.ClientID == "" || params.WorkflowID == "" || params.RunID == "" || params.NodeID == "" {
		e.logger.Debug("skipping node.started emit due to missing context",
			zap.String("workflow_id", params.WorkflowID),
			zap.String("run_id", params.RunID),
			zap.String("node_id", params.NodeID),
		)
		return nil
	}
	if len(params.Input) == 0 {
		return nil
	}

	pathCtx := PathContext{
		ClientID:   params.ClientID,
		ProjectID:  params.ProjectID,
		WorkflowID: params.WorkflowID,
		RunID:      params.RunID,
		NodeID:     params.NodeID,
		IsInput:    true,
	}

	payload, prepErr := PreparePayload(ctx, params.Input, pathCtx, e.uploader, nil)
	if prepErr != nil {
		e.logger.Warn("PreparePayload failed for node.started input payload, using inline fallback",
			zap.String("node_id", params.NodeID),
			zap.Error(prepErr),
		)
		payload = &event.Payload{
			InlineData:    params.Input,
			BlobReference: nil,
		}
	}

	if payload == nil {
		return nil
	}

	label := params.Label
	if label == "" {
		label = params.NodeID
	}
	evt := event.New(event.TypeNodeStarted).
		WithClient(params.ClientID).
		WithWorkflow(params.WorkflowID).
		WithRun(params.RunID).
		WithNode(params.NodeID).
		WithData(&event.StartNode{
			WorkflowID: params.WorkflowID,
			RunID:      params.RunID,
			ClientID:   params.ClientID,
			ProjectID:  params.ProjectID,
			NodeID:     params.NodeID,
			Label:      label,
			StartedAt:  time.Now().UnixMilli(),
			Input:      payload,
		})

	if err := e.observer.Emit(ctx, evt); err != nil {
		e.logger.Warn("failed to emit node.started observation event",
			zap.String("workflow_id", params.WorkflowID),
			zap.String("run_id", params.RunID),
			zap.String("node_id", params.NodeID),
			zap.Error(err),
		)
		return fmt.Errorf("emitter: emit node.started observation (node_id=%s): %w", params.NodeID, err)
	}
	if prepErr != nil {
		// Event emitted successfully, but we fell back to inline due to payload preparation/upload failure.
		return fmt.Errorf("emitter: emitted node.started with inline fallback (node_id=%s): %w", params.NodeID, prepErr)
	}
	return nil
}
