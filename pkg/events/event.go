package events

import (
	"encoding/json"
	"time"
)

// EventType represents the type of event being published.
type EventType string

// Event types for test lifecycle.
const (
	EventTestStarted   EventType = "test.started"
	EventTestCompleted EventType = "test.completed"
	EventTestFailed    EventType = "test.failed"
)

// Event types for task lifecycle.
const (
	EventTaskCreated   EventType = "task.created"
	EventTaskStarted   EventType = "task.started"
	EventTaskProgress  EventType = "task.progress"
	EventTaskCompleted EventType = "task.completed"
	EventTaskFailed    EventType = "task.failed"
	EventTaskLog       EventType = "task.log"
)

// Event types for client updates.
const (
	EventClientHeadUpdate   EventType = "client.head_update"
	EventClientStatusUpdate EventType = "client.status_update"
)

// Event represents a single event in the system.
type Event struct {
	ID        uint64          `json:"id"`
	Type      EventType       `json:"type"`
	Timestamp time.Time       `json:"timestamp"`
	TestRunID uint64          `json:"testRunId,omitempty"`
	TaskIndex uint64          `json:"taskIndex,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
}

// TestStartedData contains data for test.started events.
type TestStartedData struct {
	TestID   string `json:"testId"`
	TestName string `json:"testName"`
}

// TestCompletedData contains data for test.completed events.
type TestCompletedData struct {
	TestID   string `json:"testId"`
	TestName string `json:"testName"`
	Status   string `json:"status"`
}

// TestFailedData contains data for test.failed events.
type TestFailedData struct {
	TestID   string `json:"testId"`
	TestName string `json:"testName"`
	Error    string `json:"error,omitempty"`
}

// TaskStartedData contains data for task.started events.
type TaskStartedData struct {
	TaskName  string `json:"taskName"`
	TaskTitle string `json:"taskTitle"`
	TaskID    string `json:"taskId,omitempty"`
}

// TaskProgressData contains data for task.progress events.
type TaskProgressData struct {
	TaskName  string  `json:"taskName"`
	TaskTitle string  `json:"taskTitle"`
	TaskID    string  `json:"taskId,omitempty"`
	Progress  float64 `json:"progress"`
	Message   string  `json:"message,omitempty"`
}

// TaskCompletedData contains data for task.completed events.
type TaskCompletedData struct {
	TaskName  string `json:"taskName"`
	TaskTitle string `json:"taskTitle"`
	TaskID    string `json:"taskId,omitempty"`
	Result    string `json:"result"`
}

// TaskFailedData contains data for task.failed events.
type TaskFailedData struct {
	TaskName  string `json:"taskName"`
	TaskTitle string `json:"taskTitle"`
	TaskID    string `json:"taskId,omitempty"`
	Error     string `json:"error,omitempty"`
}

// TaskLogData contains data for task.log events.
type TaskLogData struct {
	TaskName  string         `json:"taskName"`
	TaskID    string         `json:"taskId,omitempty"`
	Level     string         `json:"level"`
	Message   string         `json:"message"`
	Fields    map[string]any `json:"fields,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
}

// TaskCreatedData contains data for task.created events.
type TaskCreatedData struct {
	TaskName      string `json:"taskName"`
	TaskTitle     string `json:"taskTitle"`
	TaskID        string `json:"taskId,omitempty"`
	ParentIndex   uint64 `json:"parentIndex"`
	RunConcurrent bool   `json:"runConcurrent,omitempty"`
}

// ClientHeadUpdateData contains data for client.head_update events.
type ClientHeadUpdateData struct {
	ClientIndex  int    `json:"clientIndex"`
	ClientName   string `json:"clientName"`
	CLHeadSlot   uint64 `json:"clHeadSlot"`
	CLHeadRoot   string `json:"clHeadRoot"`
	ELHeadNumber uint64 `json:"elHeadNumber"`
	ELHeadHash   string `json:"elHeadHash"`
}

// ClientStatusUpdateData contains data for client.status_update events.
type ClientStatusUpdateData struct {
	ClientIndex int    `json:"clientIndex"`
	ClientName  string `json:"clientName"`
	CLStatus    string `json:"clStatus"`
	CLReady     bool   `json:"clReady"`
	ELStatus    string `json:"elStatus"`
	ELReady     bool   `json:"elReady"`
}

// NewEvent creates a new event with the given type and data.
func NewEvent(eventType EventType, testRunID, taskIndex uint64, data any) (*Event, error) {
	var rawData json.RawMessage

	if data != nil {
		var err error

		rawData, err = json.Marshal(data)
		if err != nil {
			return nil, err
		}
	}

	return &Event{
		Type:      eventType,
		Timestamp: time.Now(),
		TestRunID: testRunID,
		TaskIndex: taskIndex,
		Data:      rawData,
	}, nil
}

// NewCustomEvent creates a new event with a custom event type string.
func NewCustomEvent(eventType string, testRunID, taskIndex uint64, data any) (*Event, error) {
	return NewEvent(EventType(eventType), testRunID, taskIndex, data)
}
