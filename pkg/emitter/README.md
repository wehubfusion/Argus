# pkg/emitter

Payload preparation helpers and high-level emitters for Argus node lifecycle events.

## Purpose

`emitter` sits between your business logic and `pkg/observer`. It decides whether a node
output or input goes inline or to blob storage, builds the monitoring blob path, and wraps
the full emit-a-node-event flow into two concrete types: `ArgusNodeEndEmitter` and
`ArgusNodeStartEmitter`.

## `PreparePayload`

```go
func PreparePayload(
    ctx     context.Context,
    data    []byte,
    pathCtx PathContext,
    uploader BlobUploader,
    opts    *PayloadOptions,
) (*event.Payload, error)
```

Applies the size threshold and returns either inline data or a blob reference.

| Condition | Result |
|---|---|
| `data` is empty | `nil, nil` |
| `len(data) <= MaxInlineBytes` or `uploader == nil` | `Payload{InlineData: data}` |
| `len(data) > MaxInlineBytes` and upload succeeds | `Payload{BlobReference: &BlobReference{URL, Size}}` |
| Upload fails | `Payload{InlineData: data}, error` — caller may log and continue |

`PathContext.IsInput` controls the blob path suffix: output paths end with `{node_id}.json`;
input paths end with `{node_id}-input.json`.

(`pkg/emitter/emitter.go:55`)

## `PathContext`

```go
type PathContext struct {
    ClientID   string
    ProjectID  string
    WorkflowID string
    RunID      string
    NodeID     string
    IsInput    bool  // true → input path suffix; false → output path suffix
}
```

All five ID fields are required. If any is empty, `BuildMonitoringPath` returns `""` and
`PreparePayload` returns an inline payload with an error.

## `PayloadOptions`

```go
type PayloadOptions struct {
    MaxInlineBytes int  // 0 means DefaultMaxInlineBytes (512 000 bytes = 500 KB)
}
```

Pass `nil` to use the default threshold.

## `DefaultMaxInlineBytes`

```go
const DefaultMaxInlineBytes = 512000
```

500 KB. Matches the thresholds used by Athena and Icarus resolver. (`pkg/emitter/emitter.go:36`)

## Blob path format

```
monitoring/{client_id}/{project_id}/{workflow_id}/{run_id}/{node_id}.json        (output)
monitoring/{client_id}/{project_id}/{workflow_id}/{run_id}/{node_id}-input.json  (input)
monitoring/{client_id}/{project_id}/{workflow_id}/{run_id}/{node_id}.csv         (CSV produce)
monitoring/{client_id}/{project_id}/{workflow_id}/{run_id}/{node_id}.xlsx        (XLSX produce)
```

(`pkg/emitter/producer_blob.go:18`)

## `BlobUploader` interface

```go
type BlobUploader interface {
    Upload(ctx context.Context, path string, data []byte, metadata map[string]string) (url string, size int64, err error)
}
```

Inject a custom implementation or use `AzureBlobUploader`. Passing `nil` forces inline storage
for all payloads regardless of size.

## `AzureBlobUploader`

```go
func NewAzureBlobUploader(connectionString, containerName string, logger *zap.Logger) (*AzureBlobUploader, error)
```

Creates a shared-key credential Azure Blob Storage client. Supports HTTP endpoints (Azurite)
for local development — detected automatically when the service URL starts with `http://`.

The container is created lazily on the first `Upload` call; if it already exists the error is
swallowed. (`pkg/emitter/azure_blob.go:108`)

## `ArgusNodeEndEmitter`

High-level wrapper that marshals the output, decides inline vs blob (with special handling for
CSV/XLSX produce artifacts), builds the `node.ended` event, and calls `observer.Emit`.

```go
func NewArgusNodeEndEmitter(obs observer.Observer, uploader BlobUploader, logger *zap.Logger) *ArgusNodeEndEmitter
```

```go
type NodeEndEmitParams struct {
    ClientID      string
    ProjectID     string
    WorkflowID    string
    RunID         string
    NodeID        string
    Label         string
    Output        interface{}   // marshaled to JSON
    HasError      bool
    ErrorMessage  string
    ContainsNodes []string
}
```

**CSV/XLSX produce artifact handling:** when the marshaled output is a JSON object with
`action: "produce"`, `fileExtension: ".csv"` or `".xlsx"`, and a valid base64 `encoded` field,
the raw decoded bytes are uploaded directly to a `.csv` or `.xlsx` blob path — bypassing the
500 KB threshold — so monitoring downloads are native files, not JSON wrappers. Falls back to
`PreparePayload` if the uploader is `nil` or the upload fails. (`pkg/emitter/emitter.go:168`)

**Nil-safe:** calling `EmitNodeEnd` on a nil receiver or with a nil observer is a no-op.

**Required fields:** `ClientID`, `WorkflowID`, `RunID`, `NodeID` must all be non-empty. Empty
required fields skip emission silently (DEBUG log). (`pkg/emitter/emitter.go:172`)

## `ArgusNodeStartEmitter`

```go
func NewArgusNodeStartEmitter(obs observer.Observer, uploader BlobUploader, logger *zap.Logger) *ArgusNodeStartEmitter
```

```go
type NodeStartEmitParams struct {
    ClientID   string
    ProjectID  string
    WorkflowID string
    RunID      string
    NodeID     string
    Label      string
    Input      []byte  // empty → no-op
}
```

Calls `PreparePayload` on the resolved input, then emits a `node.started` event. No special
produce handling — input is always treated as opaque bytes. Empty `Label` defaults to `NodeID`.
(`pkg/emitter/emitter.go:349`)

## Typical usage

```go
uploader, err := emitter.NewAzureBlobUploader(connStr, "monitoring", logger)
if err != nil { ... }

nodeEndEmitter := emitter.NewArgusNodeEndEmitter(obs, uploader, logger)

err = nodeEndEmitter.EmitNodeEnd(ctx, emitter.NodeEndEmitParams{
    ClientID:   "org_123",
    ProjectID:  "proj_abc",
    WorkflowID: "wf_xyz",
    RunID:      "run_1",
    NodeID:     "node_2",
    Label:      "Transform Data",
    Output:     myOutputStruct,
    HasError:   false,
})
// error is best-effort; log and continue
```

## See also

- [pkg/event](../event/README.md) — `Payload`, `BlobReference`, event type constants
- [pkg/observer](../observer/README.md) — how to emit events over NATS
- [Argus README](../../README.md) — architecture overview
