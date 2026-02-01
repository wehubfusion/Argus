# Argus

**Argus** is a lightweight Go SDK for emitting **workflow and execution observation events** to **NATS JetStream**.

It is designed to be used by orchestrators and workflow engines to publish **execution facts**—run lifecycle, node/plugin lifecycle, and execution payload references—without coupling producers to monitoring storage or UI concerns.

Argus focuses on **observation, not aggregation**.

---

## Purpose

Argus exists to:

* Capture **what happened** during workflow execution
* Emit **structured, durable events**
* Enable downstream systems (e.g. monitoring services) to build:

  * run history
  * execution timelines
  * health and analytics

Argus does **not** store data, compute metrics, or provide query APIs.

---

## Key Characteristics

* Event-based observation model
* Asynchronous, non-blocking emission
* Native NATS JetStream integration
* At-least-once delivery semantics
* Idempotent, replay-safe events
* Minimal public API
* Safe for use in orchestration runtimes

---

## Installation

```bash
go get github.com/wehubfusion/Argus
```

---

## Quick Start

```go
package main

import (
    "context"
    "log"

    "github.com/nats-io/nats.go"
    "github.com/wehubfusion/Argus/pkg/event"
    "github.com/wehubfusion/Argus/pkg/observer"
    "go.uber.org/zap"
)

func main() {
    // 1. Connect to NATS
    nc, err := nats.Connect("nats://localhost:4222")
    if err != nil {
        log.Fatal(err)
    }
    defer nc.Close()

    // 2. Get JetStream context
    js, err := nc.JetStream()
    if err != nil {
        log.Fatal(err)
    }

    // 3. Create observer
    logger, _ := zap.NewProduction()
    obs, err := observer.NewObserver(js, observer.DefaultOptions(), logger)
    if err != nil {
        log.Fatal(err)
    }
    defer obs.Close(context.Background())

    // 4. Emit events
    ctx := context.Background()
    evt := event.New(event.TypeRunStarted).
        WithClient("org_123").
        WithWorkflow("wf_abc").
        WithRun("run_xyz").
        WithData(map[string]interface{}{
            "trigger": "manual",
        })

    if err := obs.Emit(ctx, evt); err != nil {
        log.Printf("Failed to emit event: %v", err)
    }
}
```

---

## Event Model

Argus emits structured observation events organized by hierarchy:

### Event Types

**Level 1 - Workflow Catalog:**
* `workflow.published` - Workflow publish/unpublish lifecycle (use `Action` in data: `"publish"` | `"unpublish"`)

**Level 2 - Run Lifecycle:**
* `run.started` - Workflow run started
* `run.ended` - Workflow run completed

**Level 3 - Plugin Lifecycle:**
* `plugin.started` - Plugin/node execution started
* `plugin.ended` - Plugin/node execution completed

### Event Structure

All events follow a common structure:

```go
type Event struct {
    ID        string          // Unique event ID (for deduplication)
    Type      string          // Event type (e.g., "run.started")
    Version   string          // Schema version ("v1")
    Timestamp time.Time       // When event occurred
    
    ClientID   string         // Required: Client/organization ID
    WorkflowID string         // Optional: Workflow identifier
    RunID      string         // Optional: Run identifier
    NodeID     string         // Optional: Node/plugin identifier
    
    Data json.RawMessage      // Optional: Type-specific payload
}
```

### Field Requirements

* **Level 1 events**: Require `client_id` and `workflow_id`
* **Level 2 events**: Require `client_id`, `workflow_id`, and `run_id`
* **Level 3 events**: Require `client_id`, `workflow_id`, `run_id`, and `node_id`

---

## Usage

### Creating an Observer

```go
// With default options
obs, err := observer.NewObserver(js, observer.DefaultOptions(), logger)

// With custom options
opts := observer.DefaultOptions().
    WithBufferSize(5000).
    WithDropOnFull(false).
    WithStreamName("CUSTOM_STREAM").
    WithStreamMaxAge(7 * 24 * time.Hour).
    WithPublishTimeout(10 * time.Second)

obs, err := observer.NewObserver(js, opts, logger)
```

### Emitting Events

#### Basic Event

```go
evt := event.New(event.TypeRunStarted).
    WithClient("org_123").
    WithWorkflow("wf_abc").
    WithRun("run_xyz")

err := obs.Emit(ctx, evt)
```

#### Event with Data

```go
evt := event.New(event.TypeRunEnded).
    WithClient("org_123").
    WithWorkflow("wf_abc").
    WithRun("run_xyz").
    WithData(map[string]interface{}{
        "status":   "success",
        "duration": 1500,
    })

err := obs.Emit(ctx, evt)
```

#### Plugin Event (Level 3)

```go
evt := event.New(event.TypePluginStarted).
    WithClient("org_123").
    WithWorkflow("wf_abc").
    WithRun("run_xyz").
    WithNode("node_1").
    WithData(map[string]interface{}{
        "plugin_type": "http",
        "attempt":     1,
    })

err := obs.Emit(ctx, evt)
```

### Error Handling

```go
err := obs.Emit(ctx, evt)
switch err {
case nil:
    // Success - event queued
case observer.ErrBufferFull:
    // Buffer is full and DropOnFull=true
    log.Warn("Event dropped - buffer full")
case observer.ErrObserverClosed:
    // Observer is closed
    log.Error("Cannot emit - observer is closed")
default:
    // Context cancelled or other error
    log.Errorf("Emit failed: %v", err)
}
```

### Graceful Shutdown

```go
// Close observer (drains buffer and waits for worker)
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

if err := obs.Close(ctx); err != nil {
    log.Printf("Error closing observer: %v", err)
}
```

---

## Configuration

### Observer Options

```go
type Options struct {
    BufferSize      int           // Event buffer size (default: 1000)
    DropOnFull      bool          // Drop events when buffer full (default: true)
    StreamName      string        // JetStream stream name (default: "OBSERVATION")
    StreamMaxAge    time.Duration // Stream retention period (default: 30 days)
    StreamMaxMsgs   int64         // Max messages in stream (default: 1000000)
    PublishTimeout  time.Duration // Publish timeout (default: 5 seconds)
}
```

### Buffer Behavior

* **`DropOnFull = true`** (default): New events are dropped when buffer is full. Fast, but may lose events under high load.
* **`DropOnFull = false`**: `Emit()` blocks until buffer has space. Slower, but no event loss.

---

## Architecture

The Observer uses an asynchronous, buffered architecture:

1. **Emit**: Events are queued in a buffered channel (non-blocking)
2. **Worker**: Background goroutine processes events from the buffer
3. **Publisher**: Internal publisher handles NATS JetStream publishing with ACK/NACK handling
4. **Drain**: On close, remaining events are drained before shutdown

This design ensures:
* Non-blocking event emission
* Backpressure handling (drop or block)
* At-least-once delivery with deduplication
* Graceful shutdown with event draining

---

## Event Consumption

Events are published to NATS JetStream subjects:

* `OBSERVE.WORKFLOW.PUBLISHED` (includes Action: "publish" | "unpublish" in data)
* `OBSERVE.RUN.STARTED`
* `OBSERVE.RUN.ENDED`
* `OBSERVE.PLUGIN.STARTED`
* `OBSERVE.PLUGIN.ENDED`

Consumers can subscribe to these subjects to process events. The `pkg/event` package provides the event structure for parsing:

```go
import "github.com/wehubfusion/Argus/pkg/event"

// Parse received message
var evt event.Event
json.Unmarshal(msg.Data, &evt)

// Access event fields
fmt.Println(evt.Type, evt.ClientID, evt.WorkflowID)
```

---

## Best Practices

1. **Always close the observer**: Use `defer obs.Close(ctx)` to ensure graceful shutdown
2. **Handle errors**: Check `Emit()` return values, especially `ErrBufferFull`
3. **Use context timeouts**: Pass contexts with timeouts to `Emit()` and `Close()`
4. **Monitor buffer**: If seeing `ErrBufferFull`, consider increasing `BufferSize` or setting `DropOnFull = false`
5. **Event validation**: Events are automatically validated before publishing
6. **Idempotency**: Use unique event IDs for deduplication (auto-generated if not provided)
