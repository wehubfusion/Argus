# Testing guide

## Running the tests

```bash
go test ./...
```

All tests are in the `tests/` package and run without external dependencies.

## Test packages

| File | What it covers |
|---|---|
| `tests/event_test.go` | `Event` struct construction, `Validate()` sentinels, `ParseData` round-trips |
| `tests/observer_test.go` | `NewObserver`, `Emit` happy path, `ErrObserverClosed` after `Close` |
| `tests/options_test.go` | `DefaultOptions()` field values, fluent builder methods |
| `tests/subjects_test.go` | `SubjectForEventType` returns correct subject for every known type |
| `tests/publisher_test.go` | Internal publisher: stream create-or-update, publish with MsgId |
| `tests/nats_mock_test.go` | `MockJetStream` implementation (shared helper, not a test file itself) |

## `MockJetStream`

`tests/nats_mock_test.go` provides `MockJetStream`, which implements `nats.JetStreamContext`
in memory. Use it to test observers without a running NATS server.

```go
js := tests.NewMockJetStream()

obs, err := observer.NewObserver(js, observer.DefaultOptions(), zap.NewNop())
require.NoError(t, err)

evt := event.New(event.TypeRunEnded).
    WithClient("org_1").
    WithWorkflow("wf_1").
    WithRun("run_1").
    WithData(&event.RunEndedData{Status: "completed"})

err = obs.Emit(context.Background(), evt)
require.NoError(t, err)

msgs := js.GetPublishedMessages()
require.Len(t, msgs, 1)
assert.Equal(t, event.SubjectRunEnded, msgs[0].Subject)
```

### Simulating publish errors

```go
js.SetPublishError(errors.New("broker unavailable"))
err = obs.Emit(ctx, evt)
require.Error(t, err)

js.SetPublishError(nil) // reset to success
```

### Inspecting captured messages

```go
msgs := js.GetPublishedMessages()
// Each PublishedMessage has Subject, Data (raw JSON), MsgID
var captured event.Event
json.Unmarshal(msgs[0].Data, &captured)
```

`ClearPublishedMessages()` resets the captured slice between sub-tests.

## Testing `ArgusNodeEndEmitter` with a stub uploader

`BlobUploader` is an interface, so you can inject a stub without Azure credentials:

```go
type stubUploader struct{}

func (s *stubUploader) Upload(_ context.Context, path string, data []byte, _ map[string]string) (string, int64, error) {
    return "https://stub/" + path, int64(len(data)), nil
}

js := tests.NewMockJetStream()
obs, _ := observer.NewObserver(js, observer.DefaultOptions(), zap.NewNop())
e := emitter.NewArgusNodeEndEmitter(obs, &stubUploader{}, zap.NewNop())

err := e.EmitNodeEnd(ctx, emitter.NodeEndEmitParams{
    ClientID:   "org_1",
    ProjectID:  "proj_1",
    WorkflowID: "wf_1",
    RunID:      "run_1",
    NodeID:     "node_1",
    Output:     map[string]any{"result": "ok"},
})
```

To force blob upload, pass a large output: generate a byte slice larger than
`emitter.DefaultMaxInlineBytes` (512 000 bytes).

## Testing with Azurite (local Azure Blob Storage)

For integration tests that exercise `AzureBlobUploader`:

```bash
docker run -p 10000:10000 mcr.microsoft.com/azure-storage/azurite azurite-blob --blobHost 0.0.0.0
```

```go
connStr := "DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;" +
    "AccountKey=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KbiE9mZbM9w==;" +
    "BlobEndpoint=http://127.0.0.1:10000/devstoreaccount1"

uploader, err := emitter.NewAzureBlobUploader(connStr, "monitoring", zap.NewNop())
```

The `AzureBlobUploader` detects the `http://` prefix and enables
`InsecureAllowCredentialWithHTTP` automatically. (`pkg/emitter/azure_blob.go:56`)

## Testing with a real NATS server

```bash
docker run -p 4222:4222 nats -js
```

```go
nc, _ := nats.Connect("nats://localhost:4222")
js, _ := nc.JetStream()
obs, _ := observer.NewObserver(js, observer.DefaultOptions(), zap.NewNop())
```

Use `observer.DefaultOptions().WithStreamMaxAge(time.Minute)` in tests to avoid accumulating
messages across test runs.
