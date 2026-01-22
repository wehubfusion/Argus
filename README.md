# Argus

**Argus** is a lightweight Go SDK for emitting **workflow and execution observation events** to **NATS JetStream**.

It is designed to be used by orchestrators (such as Zeus) to publish **execution facts**—run lifecycle, node/plugin lifecycle, and execution payload references—without coupling producers to monitoring storage or UI concerns.

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

## Event Model

Argus emits structured observation events such as:

* Workflow run started / ended
* Node or plugin started / ended
* Execution payload available

Events are published to JetStream and consumed by monitoring or analytics services.

