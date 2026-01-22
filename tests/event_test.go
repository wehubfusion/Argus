package tests

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/wehubfusion/Argus/pkg/event"
)

func TestNew(t *testing.T) {
	evt := event.New(event.TypeRunStarted)

	if evt.ID == "" {
		t.Error("Expected event ID to be generated")
	}
	if evt.Type != event.TypeRunStarted {
		t.Errorf("Expected type %s, got %s", event.TypeRunStarted, evt.Type)
	}
	if evt.Version != "v1" {
		t.Errorf("Expected version 'v1', got %s", evt.Version)
	}
	if evt.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}
}

func TestWithClient(t *testing.T) {
	evt := event.New(event.TypeRunStarted).WithClient("org_123")

	if evt.ClientID != "org_123" {
		t.Errorf("Expected client_id 'org_123', got %s", evt.ClientID)
	}
}

func TestWithWorkflow(t *testing.T) {
	evt := event.New(event.TypeRunStarted).WithWorkflow("wf_abc")

	if evt.WorkflowID != "wf_abc" {
		t.Errorf("Expected workflow_id 'wf_abc', got %s", evt.WorkflowID)
	}
}

func TestWithRun(t *testing.T) {
	evt := event.New(event.TypeRunStarted).WithRun("run_xyz")

	if evt.RunID != "run_xyz" {
		t.Errorf("Expected run_id 'run_xyz', got %s", evt.RunID)
	}
}

func TestWithNode(t *testing.T) {
	evt := event.New(event.TypePluginStarted).WithNode("node_1")

	if evt.NodeID != "node_1" {
		t.Errorf("Expected node_id 'node_1', got %s", evt.NodeID)
	}
}

func TestWithData(t *testing.T) {
	// Test WithData with a simple map
	data := map[string]interface{}{
		"key": "value",
		"num": 42,
	}

	evt := event.New(event.TypeRunEnded).WithData(data)

	if evt.Data == nil {
		t.Error("Expected data to be set")
	}

	// Verify data was marshaled to JSON
	if len(evt.Data) == 0 {
		t.Error("Expected data to be non-empty")
	}

	// Parse back and verify it's valid JSON
	var parsed map[string]interface{}
	if err := evt.ParseData(&parsed); err != nil {
		t.Fatalf("Failed to parse data: %v", err)
	}

	if parsed["key"] != "value" {
		t.Errorf("Expected key 'value', got %v", parsed["key"])
	}
}

func TestWithRawData(t *testing.T) {
	rawData := json.RawMessage(`{"status":"error","error":"test error"}`)
	evt := event.New(event.TypeRunEnded).WithRawData(rawData)

	if string(evt.Data) != string(rawData) {
		t.Error("Expected raw data to match")
	}
}

func TestWithDataNil(t *testing.T) {
	evt := event.New(event.TypeRunStarted).WithData(nil)

	if evt.Data != nil {
		t.Error("Expected data to be nil")
	}
}

func TestBytes(t *testing.T) {
	evt := event.New(event.TypeRunStarted).
		WithClient("org_123").
		WithWorkflow("wf_abc").
		WithRun("run_xyz")

	bytes, err := evt.Bytes()
	if err != nil {
		t.Fatalf("Failed to serialize event: %v", err)
	}

	if len(bytes) == 0 {
		t.Error("Expected serialized bytes to be non-empty")
	}

	// Verify it's valid JSON
	var decoded event.Event
	if err := json.Unmarshal(bytes, &decoded); err != nil {
		t.Fatalf("Failed to deserialize event: %v", err)
	}

	if decoded.ClientID != "org_123" {
		t.Errorf("Expected client_id 'org_123', got %s", decoded.ClientID)
	}
}

func TestParseData(t *testing.T) {
	// Test ParseData with a simple map
	data := map[string]interface{}{
		"field1": "value1",
		"field2": 123,
	}

	evt := event.New(event.TypePluginEnded).WithData(data)

	var parsed map[string]interface{}
	if err := evt.ParseData(&parsed); err != nil {
		t.Fatalf("Failed to parse data: %v", err)
	}

	if parsed["field1"] != "value1" {
		t.Errorf("Expected field1 'value1', got %v", parsed["field1"])
	}
	if parsed["field2"].(float64) != 123 {
		t.Errorf("Expected field2 123, got %v", parsed["field2"])
	}
}

func TestParseDataNil(t *testing.T) {
	evt := event.New(event.TypeRunStarted)

	var data map[string]interface{}
	if err := evt.ParseData(&data); err != nil {
		t.Errorf("Expected no error for nil data, got %v", err)
	}
}

func TestValidate_Level1_WorkflowPublished(t *testing.T) {
	evt := event.New(event.TypeWorkflowPublished).
		WithClient("org_123").
		WithWorkflow("wf_abc")

	if err := evt.Validate(); err != nil {
		t.Errorf("Expected valid event, got error: %v", err)
	}
}

func TestValidate_Level1_MissingWorkflowID(t *testing.T) {
	evt := event.New(event.TypeWorkflowPublished).
		WithClient("org_123")
	// Missing workflow_id

	if err := evt.Validate(); err != event.ErrMissingWorkflowID {
		t.Errorf("Expected ErrMissingWorkflowID, got %v", err)
	}
}

func TestValidate_Level2_RunStarted(t *testing.T) {
	evt := event.New(event.TypeRunStarted).
		WithClient("org_123").
		WithWorkflow("wf_abc").
		WithRun("run_xyz")

	if err := evt.Validate(); err != nil {
		t.Errorf("Expected valid event, got error: %v", err)
	}
}

func TestValidate_Level2_MissingRunID(t *testing.T) {
	evt := event.New(event.TypeRunStarted).
		WithClient("org_123").
		WithWorkflow("wf_abc")
	// Missing run_id

	if err := evt.Validate(); err != event.ErrMissingRunID {
		t.Errorf("Expected ErrMissingRunID, got %v", err)
	}
}

func TestValidate_Level3_PluginStarted(t *testing.T) {
	evt := event.New(event.TypePluginStarted).
		WithClient("org_123").
		WithWorkflow("wf_abc").
		WithRun("run_xyz").
		WithNode("node_1")

	if err := evt.Validate(); err != nil {
		t.Errorf("Expected valid event, got error: %v", err)
	}
}

func TestValidate_Level3_MissingNodeID(t *testing.T) {
	evt := event.New(event.TypePluginStarted).
		WithClient("org_123").
		WithWorkflow("wf_abc").
		WithRun("run_xyz")
	// Missing node_id

	if err := evt.Validate(); err != event.ErrMissingNodeID {
		t.Errorf("Expected ErrMissingNodeID, got %v", err)
	}
}

func TestValidate_MissingEventID(t *testing.T) {
	evt := event.New(event.TypeRunStarted)
	evt.ID = "" // Clear ID

	if err := evt.Validate(); err != event.ErrMissingEventID {
		t.Errorf("Expected ErrMissingEventID, got %v", err)
	}
}

func TestValidate_MissingEventType(t *testing.T) {
	evt := event.New(event.TypeRunStarted)
	evt.Type = "" // Clear type

	if err := evt.Validate(); err != event.ErrMissingEventType {
		t.Errorf("Expected ErrMissingEventType, got %v", err)
	}
}

func TestValidate_MissingClientID(t *testing.T) {
	evt := event.New(event.TypeRunStarted).
		WithWorkflow("wf_abc").
		WithRun("run_xyz")
	// Missing client_id

	if err := evt.Validate(); err != event.ErrMissingClientID {
		t.Errorf("Expected ErrMissingClientID, got %v", err)
	}
}

func TestFluentBuilder(t *testing.T) {
	// Test fluent builder pattern with data
	data := map[string]interface{}{
		"test": "data",
	}

	evt := event.New(event.TypePluginEnded).
		WithClient("org_123").
		WithWorkflow("wf_abc").
		WithRun("run_xyz").
		WithNode("node_1").
		WithData(data)

	if evt.ClientID != "org_123" {
		t.Error("ClientID not set correctly")
	}
	if evt.WorkflowID != "wf_abc" {
		t.Error("WorkflowID not set correctly")
	}
	if evt.RunID != "run_xyz" {
		t.Error("RunID not set correctly")
	}
	if evt.NodeID != "node_1" {
		t.Error("NodeID not set correctly")
	}

	// Verify data was set
	if evt.Data == nil {
		t.Error("Data not set correctly")
	}
}

func TestEventSerialization(t *testing.T) {
	now := time.Now()
	// Test serialization with data
	data := map[string]interface{}{
		"key": "value",
	}

	evt := event.New(event.TypeRunStarted)
	evt.Timestamp = now
	evt.WithClient("org_123").
		WithWorkflow("wf_abc").
		WithRun("run_xyz").
		WithData(data)

	bytes, err := evt.Bytes()
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	// Verify JSON structure
	var jsonMap map[string]interface{}
	if err := json.Unmarshal(bytes, &jsonMap); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if jsonMap["id"] == nil {
		t.Error("Missing 'id' field")
	}
	if jsonMap["type"] != event.TypeRunStarted {
		t.Errorf("Expected type %s, got %v", event.TypeRunStarted, jsonMap["type"])
	}
	if jsonMap["client_id"] != "org_123" {
		t.Errorf("Expected client_id 'org_123', got %v", jsonMap["client_id"])
	}
	if jsonMap["workflow_id"] != "wf_abc" {
		t.Errorf("Expected workflow_id 'wf_abc', got %v", jsonMap["workflow_id"])
	}
	if jsonMap["run_id"] != "run_xyz" {
		t.Errorf("Expected run_id 'run_xyz', got %v", jsonMap["run_id"])
	}
	if jsonMap["data"] == nil {
		t.Error("Missing 'data' field")
	}
}
