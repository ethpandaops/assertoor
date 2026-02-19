package events

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
)

// AuthTokenChecker is a function that validates an authorization token.
type AuthTokenChecker func(tokenStr string) *jwt.Token

// SSEHandler handles Server-Sent Events connections.
type SSEHandler struct {
	logger         logrus.FieldLogger
	eventBus       *EventBus
	authChecker    AuthTokenChecker
	requireAuthLog bool
}

// NewSSEHandler creates a new SSE handler.
func NewSSEHandler(logger logrus.FieldLogger, eventBus *EventBus) *SSEHandler {
	return &SSEHandler{
		logger:         logger.WithField("component", "sse"),
		eventBus:       eventBus,
		authChecker:    nil,
		requireAuthLog: false,
	}
}

// NewSSEHandlerWithAuth creates a new SSE handler with authentication support.
func NewSSEHandlerWithAuth(logger logrus.FieldLogger, eventBus *EventBus, authChecker AuthTokenChecker, requireAuthLog bool) *SSEHandler {
	return &SSEHandler{
		logger:         logger.WithField("component", "sse"),
		eventBus:       eventBus,
		authChecker:    authChecker,
		requireAuthLog: requireAuthLog,
	}
}

// HandleGlobalStream handles the global event stream endpoint.
func (h *SSEHandler) HandleGlobalStream(w http.ResponseWriter, r *http.Request) {
	h.handleSSE(w, r, nil)
}

// HandleTestRunStream handles the per-test event stream endpoint.
func (h *SSEHandler) HandleTestRunStream(w http.ResponseWriter, r *http.Request, testRunID uint64) {
	filter := CreateTestRunFilter(testRunID)
	h.handleSSE(w, r, filter)
}

// handleSSE is the common SSE handling logic.
func (h *SSEHandler) handleSSE(w http.ResponseWriter, r *http.Request, filter FilterFunc) {
	// Check if the client supports SSE
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	// Check authentication for log events
	isAuthenticated := h.checkAuth(r)

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Accel-Buffering", "no")

	// Parse optional lastEventId for reconnection
	lastEventIDStr := r.Header.Get("Last-Event-ID")
	if lastEventIDStr == "" {
		lastEventIDStr = r.URL.Query().Get("lastEventId")
	}

	var lastEventID uint64

	if lastEventIDStr != "" {
		var err error

		lastEventID, err = strconv.ParseUint(lastEventIDStr, 10, 64)
		if err != nil {
			h.logger.WithError(err).Warn("invalid Last-Event-ID")
		}
	}

	// Subscribe to events
	sub := h.eventBus.Subscribe(filter)
	defer h.eventBus.Unsubscribe(sub)

	ctx := r.Context()

	h.logger.WithField("last_event_id", lastEventID).Debug("SSE client connected")

	// Send initial connection event
	h.sendEvent(w, flusher, &Event{
		ID:        0,
		Type:      "connected",
		Timestamp: time.Now(),
	})

	// Keep-alive ticker
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			h.logger.Debug("SSE client disconnected")
			return

		case event := <-sub.Channel():
			if event == nil {
				return
			}

			// Skip events before lastEventID for reconnection support
			if lastEventID > 0 && event.ID <= lastEventID {
				continue
			}

			// Filter out log events for unauthenticated clients
			if h.requireAuthLog && !isAuthenticated && event.Type == EventTaskLog {
				continue
			}

			h.sendEvent(w, flusher, event)

		case <-ticker.C:
			// Send keep-alive comment
			h.sendKeepAlive(w, flusher)
		}
	}
}

// checkAuth checks if the request has a valid authentication token.
func (h *SSEHandler) checkAuth(r *http.Request) bool {
	if h.authChecker == nil {
		return true // No auth checker, allow all
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		// Also check query parameter for SSE connections
		authHeader = r.URL.Query().Get("token")
		if authHeader != "" {
			authHeader = "Bearer " + authHeader
		}
	}

	if authHeader == "" {
		return false
	}

	token := h.authChecker(authHeader)

	return token != nil && token.Valid
}

// sendEvent sends an SSE event to the client.
func (h *SSEHandler) sendEvent(w http.ResponseWriter, flusher http.Flusher, event *Event) {
	data, err := json.Marshal(event)
	if err != nil {
		h.logger.WithError(err).Error("failed to marshal event")
		return
	}

	// Write event ID for reconnection support
	if event.ID > 0 {
		fmt.Fprintf(w, "id: %d\n", event.ID)
	}

	// Write event type
	fmt.Fprintf(w, "event: %s\n", event.Type)

	// Write data
	fmt.Fprintf(w, "data: %s\n\n", data)

	flusher.Flush()
}

// sendKeepAlive sends a keep-alive comment to prevent connection timeout.
func (h *SSEHandler) sendKeepAlive(w http.ResponseWriter, flusher http.Flusher) {
	fmt.Fprint(w, ": keep-alive\n\n")
	flusher.Flush()
}

// SSEMiddleware wraps an SSE handler with common functionality.
type SSEMiddleware struct {
	handler  *SSEHandler
	eventBus *EventBus
}

// NewSSEMiddleware creates a new SSE middleware.
func NewSSEMiddleware(logger logrus.FieldLogger, eventBus *EventBus) *SSEMiddleware {
	return &SSEMiddleware{
		handler:  NewSSEHandler(logger, eventBus),
		eventBus: eventBus,
	}
}

// GlobalStreamHandler returns an http.HandlerFunc for the global event stream.
func (m *SSEMiddleware) GlobalStreamHandler() http.HandlerFunc {
	return m.handler.HandleGlobalStream
}

// TestRunStreamHandler returns a function that creates handlers for test run streams.
func (m *SSEMiddleware) TestRunStreamHandler(testRunID uint64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m.handler.HandleTestRunStream(w, r, testRunID)
	}
}

// PublishTestStarted publishes a test started event.
func (eb *EventBus) PublishTestStarted(testRunID uint64, testID, testName string) {
	event, err := NewEvent(EventTestStarted, testRunID, 0, &TestStartedData{
		TestID:   testID,
		TestName: testName,
	})
	if err != nil {
		return
	}

	eb.Publish(event)
}

// PublishTestCompleted publishes a test completed event.
func (eb *EventBus) PublishTestCompleted(testRunID uint64, testID, testName, status string) {
	event, err := NewEvent(EventTestCompleted, testRunID, 0, &TestCompletedData{
		TestID:   testID,
		TestName: testName,
		Status:   status,
	})
	if err != nil {
		return
	}

	eb.Publish(event)
}

// PublishTestFailed publishes a test failed event.
func (eb *EventBus) PublishTestFailed(testRunID uint64, testID, testName, errMsg string) {
	event, err := NewEvent(EventTestFailed, testRunID, 0, &TestFailedData{
		TestID:   testID,
		TestName: testName,
		Error:    errMsg,
	})
	if err != nil {
		return
	}

	eb.Publish(event)
}

// PublishTaskStarted publishes a task started event.
func (eb *EventBus) PublishTaskStarted(
	testRunID, taskIndex uint64,
	taskName, taskTitle, taskID string,
) {
	event, err := NewEvent(EventTaskStarted, testRunID, taskIndex, &TaskStartedData{
		TaskName:  taskName,
		TaskTitle: taskTitle,
		TaskID:    taskID,
	})
	if err != nil {
		return
	}

	eb.Publish(event)
}

// PublishTaskProgress publishes a task progress event.
func (eb *EventBus) PublishTaskProgress(
	testRunID, taskIndex uint64,
	taskName, taskTitle, taskID string,
	progress float64,
	message string,
) {
	event, err := NewEvent(EventTaskProgress, testRunID, taskIndex, &TaskProgressData{
		TaskName:  taskName,
		TaskTitle: taskTitle,
		TaskID:    taskID,
		Progress:  progress,
		Message:   message,
	})
	if err != nil {
		return
	}

	eb.Publish(event)
}

// PublishTaskCompleted publishes a task completed event.
func (eb *EventBus) PublishTaskCompleted(
	testRunID, taskIndex uint64,
	taskName, taskTitle, taskID, result string,
) {
	event, err := NewEvent(EventTaskCompleted, testRunID, taskIndex, &TaskCompletedData{
		TaskName:  taskName,
		TaskTitle: taskTitle,
		TaskID:    taskID,
		Result:    result,
	})
	if err != nil {
		return
	}

	eb.Publish(event)
}

// PublishTaskFailed publishes a task failed event.
func (eb *EventBus) PublishTaskFailed(
	testRunID, taskIndex uint64,
	taskName, taskTitle, taskID, errMsg string,
) {
	event, err := NewEvent(EventTaskFailed, testRunID, taskIndex, &TaskFailedData{
		TaskName:  taskName,
		TaskTitle: taskTitle,
		TaskID:    taskID,
		Error:     errMsg,
	})
	if err != nil {
		return
	}

	eb.Publish(event)
}

// PublishTaskLog publishes a task log event.
func (eb *EventBus) PublishTaskLog(
	testRunID, taskIndex uint64,
	taskName, taskID, level, message string,
	fields map[string]any,
) {
	event, err := NewEvent(EventTaskLog, testRunID, taskIndex, &TaskLogData{
		TaskName:  taskName,
		TaskID:    taskID,
		Level:     level,
		Message:   message,
		Fields:    fields,
		Timestamp: time.Now(),
	})
	if err != nil {
		return
	}

	eb.Publish(event)
}

// PublishTaskCreated publishes a task created event.
func (eb *EventBus) PublishTaskCreated(
	testRunID, taskIndex uint64,
	taskName, taskTitle, taskID string,
	parentIndex uint64,
	runConcurrent bool,
) {
	event, err := NewEvent(EventTaskCreated, testRunID, taskIndex, &TaskCreatedData{
		TaskName:      taskName,
		TaskTitle:     taskTitle,
		TaskID:        taskID,
		ParentIndex:   parentIndex,
		RunConcurrent: runConcurrent,
	})
	if err != nil {
		return
	}

	eb.Publish(event)
}

// PublishClientHeadUpdate publishes a client head update event.
func (eb *EventBus) PublishClientHeadUpdate(
	clientIndex int,
	clientName string,
	clHeadSlot uint64,
	clHeadRoot string,
	elHeadNumber uint64,
	elHeadHash string,
) {
	event, err := NewEvent(EventClientHeadUpdate, 0, 0, &ClientHeadUpdateData{
		ClientIndex:  clientIndex,
		ClientName:   clientName,
		CLHeadSlot:   clHeadSlot,
		CLHeadRoot:   clHeadRoot,
		ELHeadNumber: elHeadNumber,
		ELHeadHash:   elHeadHash,
	})
	if err != nil {
		return
	}

	eb.Publish(event)
}

// PublishClientStatusUpdate publishes a client status update event.
func (eb *EventBus) PublishClientStatusUpdate(
	clientIndex int,
	clientName string,
	clStatus string,
	clReady bool,
	elStatus string,
	elReady bool,
) {
	event, err := NewEvent(EventClientStatusUpdate, 0, 0, &ClientStatusUpdateData{
		ClientIndex: clientIndex,
		ClientName:  clientName,
		CLStatus:    clStatus,
		CLReady:     clReady,
		ELStatus:    elStatus,
		ELReady:     elReady,
	})
	if err != nil {
		return
	}

	eb.Publish(event)
}

// HandleClientStream handles the client events stream endpoint.
func (h *SSEHandler) HandleClientStream(w http.ResponseWriter, r *http.Request) {
	// Filter for client events only
	filter := CreateEventTypeFilter(EventClientHeadUpdate, EventClientStatusUpdate)
	h.handleSSE(w, r, filter)
}
