package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/ethpandaops/assertoor/pkg/ai"
	"github.com/ethpandaops/assertoor/pkg/db"
	"github.com/ethpandaops/assertoor/pkg/web/auth"
	"github.com/ethpandaops/assertoor/pkg/web/types"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

// AIHandler handles AI-related API endpoints.
type AIHandler struct {
	config         *types.AIConfig
	client         *ai.OpenRouterClient
	database       *db.Database
	logger         logrus.FieldLogger
	authHandler    *auth.Handler
	disableAuth    bool
	sessionManager *ai.SessionManager
}

// NewAIHandler creates a new AI handler.
func NewAIHandler(
	config *types.AIConfig,
	database *db.Database,
	logger logrus.FieldLogger,
	authHandler *auth.Handler,
	disableAuth bool,
) *AIHandler {
	var client *ai.OpenRouterClient
	if config != nil && config.Enabled && config.OpenRouterKey != "" {
		client = ai.NewOpenRouterClient(config.OpenRouterKey)
	}

	return &AIHandler{
		config:         config,
		client:         client,
		database:       database,
		logger:         logger.WithField("module", "ai"),
		authHandler:    authHandler,
		disableAuth:    disableAuth,
		sessionManager: ai.NewSessionManager(),
	}
}

func (h *AIHandler) checkAuth(r *http.Request) bool {
	if h.disableAuth {
		return true
	}

	if h.authHandler == nil {
		return true
	}

	token := h.authHandler.CheckAuthToken(r.Header.Get("Authorization"))

	return token != nil && token.Valid
}

// AIConfigResponse is the response for GetConfig endpoint.
type AIConfigResponse struct {
	Enabled             bool     `json:"enabled"`
	DefaultModel        string   `json:"defaultModel"`
	AllowedModels       []string `json:"allowedModels"`
	ServerKeyConfigured bool     `json:"serverKeyConfigured"`
}

// GetConfig returns AI configuration (enabled, available models).
// GET /api/v1/ai/config
func (h *AIHandler) GetConfig(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", contentTypeJSON)

	response := AIConfigResponse{
		Enabled:             false,
		DefaultModel:        "",
		AllowedModels:       []string{},
		ServerKeyConfigured: false,
	}

	if h.config != nil && h.config.Enabled {
		response.Enabled = true
		response.DefaultModel = h.config.DefaultModel
		response.AllowedModels = h.config.AllowedModels
		response.ServerKeyConfigured = h.client != nil
	}

	h.sendOKResponse(w, "/api/v1/ai/config", response)
}

// SystemPromptResponse is the response for GetSystemPrompt endpoint.
type SystemPromptResponse struct {
	Prompt string `json:"prompt"`
}

// GetSystemPrompt returns the AI system prompt text.
// GET /api/v1/ai/system_prompt
func (h *AIHandler) GetSystemPrompt(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", contentTypeJSON)

	prompt := ai.BuildSystemPrompt()
	h.sendOKResponse(w, "/api/v1/ai/system_prompt", SystemPromptResponse{
		Prompt: prompt,
	})
}

// ValidateYamlRequest is the request body for ValidateYaml endpoint.
type ValidateYamlRequest struct {
	Yaml string `json:"yaml"`
}

// ValidateYaml validates AI-generated YAML and returns validation results.
// POST /api/v1/ai/validate
func (h *AIHandler) ValidateYaml(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", contentTypeJSON)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.sendErrorResponse(
			w, "/api/v1/ai/validate", "failed to read request body", http.StatusBadRequest,
		)

		return
	}

	var req ValidateYamlRequest

	err = json.Unmarshal(body, &req)
	if err != nil {
		h.sendErrorResponse(
			w, "/api/v1/ai/validate", "invalid request body", http.StatusBadRequest,
		)

		return
	}

	result := ai.ValidateGeneratedYaml(req.Yaml)
	h.sendOKResponse(w, "/api/v1/ai/validate", result)
}

// AIUsageResponse is the response for GetUsage endpoint.
type AIUsageResponse struct {
	LastDay   *db.AIUsageStats `json:"lastDay"`
	LastMonth *db.AIUsageStats `json:"lastMonth"`
}

// GetUsage returns token usage statistics.
// GET /api/v1/ai/usage
func (h *AIHandler) GetUsage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", contentTypeJSON)

	if !h.checkAuth(r) {
		h.sendUnauthorizedResponse(w, "/api/v1/ai/usage")
		return
	}

	now := time.Now()
	dayAgo := now.Add(-24 * time.Hour)
	monthAgo := now.Add(-30 * 24 * time.Hour)

	lastDay, err := h.database.GetUsageByDateRange(dayAgo, now)
	if err != nil {
		h.logger.WithError(err).Error("failed to get daily usage")

		lastDay = &db.AIUsageStats{}
	}

	lastMonth, err := h.database.GetUsageByDateRange(monthAgo, now)
	if err != nil {
		h.logger.WithError(err).Error("failed to get monthly usage")

		lastMonth = &db.AIUsageStats{}
	}

	h.sendOKResponse(w, "/api/v1/ai/usage", AIUsageResponse{
		LastDay:   lastDay,
		LastMonth: lastMonth,
	})
}

// AIChatRequest is the request body for Chat endpoint.
type AIChatRequest struct {
	Model       string           `json:"model"`
	Messages    []ai.ChatMessage `json:"messages"`
	TestName    string           `json:"testName"`
	CurrentYaml string           `json:"currentYaml,omitempty"`
}

// AIChatStartResponse is the response for starting a chat session.
type AIChatStartResponse struct {
	SessionID string `json:"sessionId"`
}

// Chat starts an AI chat session and returns the session ID.
// POST /api/v1/ai/chat
func (h *AIHandler) Chat(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", contentTypeJSON)

	if !h.checkAuth(r) {
		h.sendUnauthorizedResponse(w, "/api/v1/ai/chat")
		return
	}

	if h.client == nil {
		h.sendErrorResponse(w, "/api/v1/ai/chat", "AI is not enabled", http.StatusServiceUnavailable)
		return
	}

	// Parse request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.sendErrorResponse(w, "/api/v1/ai/chat", "failed to read request body", http.StatusBadRequest)
		return
	}

	var req AIChatRequest

	err = json.Unmarshal(body, &req)
	if err != nil {
		h.sendErrorResponse(w, "/api/v1/ai/chat", "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate model
	model := req.Model
	if model == "" {
		model = h.config.DefaultModel
	}

	if !h.isModelAllowed(model) {
		h.sendErrorResponse(w, "/api/v1/ai/chat", "model not allowed", http.StatusBadRequest)
		return
	}

	// Create session
	session := h.sessionManager.CreateSession(3) // Max 3 fix attempts

	// Start processing in background
	go h.processChat(session, model, req)

	// Return session ID immediately
	h.sendOKResponse(w, "/api/v1/ai/chat", AIChatStartResponse{
		SessionID: session.ID,
	})
}

// GetChatSession returns the current state of a chat session.
// GET /api/v1/ai/chat/{sessionId}
func (h *AIHandler) GetChatSession(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", contentTypeJSON)

	if !h.checkAuth(r) {
		h.sendUnauthorizedResponse(w, "/api/v1/ai/chat/{sessionId}")
		return
	}

	vars := mux.Vars(r)
	sessionID := vars["sessionId"]

	session := h.sessionManager.GetSession(sessionID)
	if session == nil {
		h.sendErrorResponse(w, "/api/v1/ai/chat/{sessionId}", "session not found", http.StatusNotFound)
		return
	}

	h.sendOKResponse(w, "/api/v1/ai/chat/{sessionId}", session.GetSnapshot())
}

// StreamChatSession streams updates for a chat session using SSE.
// GET /api/v1/ai/chat/{sessionId}/stream
func (h *AIHandler) StreamChatSession(w http.ResponseWriter, r *http.Request) {
	if !h.checkAuth(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	sessionID := vars["sessionId"]

	session := h.sessionManager.GetSession(sessionID)
	if session == nil {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Subscribe to session updates
	updateCh := session.Subscribe()
	defer session.Unsubscribe(updateCh)

	// Send initial state
	h.sendSSEEvent(w, flusher, "update", session.GetSnapshot())

	// Stream updates until session completes or client disconnects
	ctx := r.Context()

	for {
		select {
		case <-ctx.Done():
			return
		case update, ok := <-updateCh:
			if !ok {
				return
			}

			h.sendSSEEvent(w, flusher, "update", update.GetSnapshot())

			// Stop streaming when session is complete or errored
			if update.Status == ai.SessionStatusComplete || update.Status == ai.SessionStatusError {
				return
			}
		}
	}
}

func (h *AIHandler) sendSSEEvent(w http.ResponseWriter, flusher http.Flusher, event string, data any) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return
	}

	fmt.Fprintf(w, "event: %s\n", event)
	fmt.Fprintf(w, "data: %s\n\n", jsonData)
	flusher.Flush()
}

// processChat handles the AI chat processing in the background.
func (h *AIHandler) processChat(session *ai.Session, model string, req AIChatRequest) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	session.SetCancelFunc(cancel)
	session.UpdateStatus(ai.SessionStatusStreaming)

	// Build messages with system prompt
	messages := h.buildMessages(req.Messages, req.CurrentYaml)

	// Send request to OpenRouter with streaming
	chatReq := &ai.ChatRequest{
		Model:     model,
		Messages:  messages,
		MaxTokens: h.config.MaxTokens,
	}

	var fullResponse string

	chatResp, err := h.client.ChatStream(ctx, chatReq, func(chunk string) {
		fullResponse += chunk
		session.AppendResponse(chunk)
	})
	if err != nil {
		h.logger.WithError(err).Error("failed to get AI response")
		session.SetError("Failed to get AI response: " + err.Error())

		return
	}

	// Record token usage
	testName := req.TestName
	if testName == "" {
		testName = "new_test"
	}

	session.SetUsage(chatResp.Usage.PromptTokens, chatResp.Usage.CompletionTokens, chatResp.Usage.TotalTokens)

	err = h.database.RunTransaction(func(tx *sqlx.Tx) error {
		return h.database.RecordTokenUsage(
			tx,
			time.Now(),
			testName,
			model,
			int64(chatResp.Usage.PromptTokens),
			int64(chatResp.Usage.CompletionTokens),
		)
	})
	if err != nil {
		h.logger.WithError(err).Error("failed to record token usage")
	}

	// Extract and validate YAML
	session.UpdateStatus(ai.SessionStatusValidating)

	generatedYaml := extractYamlFromResponse(fullResponse)
	session.SetGeneratedYaml(generatedYaml)

	if generatedYaml != "" {
		validation := ai.ValidateGeneratedYaml(generatedYaml)
		session.SetValidation(validation)

		// If there are errors and we can retry, attempt to fix
		if !validation.Valid && session.CanRetryFix() {
			h.attemptFix(ctx, session, model, req, fullResponse, generatedYaml, validation, false)

			return
		}

		// If no errors but has warnings and we haven't tried to fix warnings yet, attempt once
		if validation.Valid && hasWarnings(validation) && session.CanFixWarnings() {
			h.attemptFix(ctx, session, model, req, fullResponse, generatedYaml, validation, true)

			return
		}
	}

	session.Complete()
}

// hasWarnings checks if a validation result has any warnings.
func hasWarnings(validation *ai.ValidationResult) bool {
	if validation == nil || validation.Issues == nil {
		return false
	}

	for _, issue := range validation.Issues {
		if issue.Type == "warning" {
			return true
		}
	}

	return false
}

// attemptFix attempts to fix validation errors/warnings by asking the AI to correct them.
// If warningsOnly is true, this is a single attempt to fix warnings (no retries).
func (h *AIHandler) attemptFix(
	ctx context.Context,
	session *ai.Session,
	model string,
	req AIChatRequest,
	originalResponse string,
	brokenYaml string,
	validation *ai.ValidationResult,
	warningsOnly bool,
) {
	session.UpdateStatus(ai.SessionStatusFixing)

	if warningsOnly {
		session.SetWarningFixAttempted()
	} else {
		session.IncrementFixAttempts()
	}

	// Build fix prompt
	fixPrompt := h.buildFixPrompt(brokenYaml, validation, warningsOnly)

	// Build messages for fix request (don't include in user's conversation)
	messages := h.buildMessages(req.Messages, req.CurrentYaml)
	messages = append(messages,
		ai.ChatMessage{
			Role:    "assistant",
			Content: originalResponse,
		},
		ai.ChatMessage{
			Role:    "user",
			Content: fixPrompt,
		},
	)

	chatReq := &ai.ChatRequest{
		Model:     model,
		Messages:  messages,
		MaxTokens: h.config.MaxTokens,
	}

	chatResp, err := h.client.Chat(ctx, chatReq)
	if err != nil {
		h.logger.WithError(err).Error("failed to get fix response")
		// Keep the original response with validation errors
		session.Complete()

		return
	}

	// Record token usage for fix attempt
	session.SetUsage(chatResp.Usage.PromptTokens, chatResp.Usage.CompletionTokens, chatResp.Usage.TotalTokens)

	testName := req.TestName
	if testName == "" {
		testName = "new_test"
	}

	err = h.database.RunTransaction(func(tx *sqlx.Tx) error {
		return h.database.RecordTokenUsage(
			tx,
			time.Now(),
			testName,
			model,
			int64(chatResp.Usage.PromptTokens),
			int64(chatResp.Usage.CompletionTokens),
		)
	})
	if err != nil {
		h.logger.WithError(err).Error("failed to record fix token usage")
	}

	// Extract fixed YAML
	fixResponse := ""
	if len(chatResp.Choices) > 0 {
		fixResponse = chatResp.Choices[0].Message.Content
	}

	fixedYaml := extractYamlFromResponse(fixResponse)
	if fixedYaml == "" {
		// No YAML in fix response, keep original
		session.Complete()

		return
	}

	// Validate fixed YAML
	fixValidation := ai.ValidateGeneratedYaml(fixedYaml)

	// Update session with fixed YAML
	session.SetGeneratedYaml(fixedYaml)
	session.SetValidation(fixValidation)

	// If this was a warning-only fix, we're done (only 1 attempt allowed)
	if warningsOnly {
		session.Complete()

		return
	}

	// If still has errors and can retry, try again
	if !fixValidation.Valid && session.CanRetryFix() {
		h.attemptFix(ctx, session, model, req, originalResponse, fixedYaml, fixValidation, false)

		return
	}

	// If errors are fixed but has warnings and haven't tried warning fix yet, try once
	if fixValidation.Valid && hasWarnings(fixValidation) && session.CanFixWarnings() {
		h.attemptFix(ctx, session, model, req, originalResponse, fixedYaml, fixValidation, true)

		return
	}

	session.Complete()
}

func (h *AIHandler) buildFixPrompt(brokenYaml string, validation *ai.ValidationResult, warningsOnly bool) string {
	var prompt string

	if warningsOnly {
		prompt = "The YAML you generated has validation warnings. Please review and fix the following issues if appropriate:\n\n"

		for _, issue := range validation.Issues {
			if issue.Type == "warning" {
				prompt += fmt.Sprintf("- %s: %s\n", issue.Path, issue.Message)
			}
		}
	} else {
		prompt = "The YAML you generated has validation errors. Please fix the following issues:\n\n"

		for _, issue := range validation.Issues {
			if issue.Type == "error" {
				prompt += fmt.Sprintf("- %s: %s\n", issue.Path, issue.Message)
			}
		}
	}

	prompt += "\nHere is the YAML:\n```yaml\n" + brokenYaml + "\n```\n\n"

	if warningsOnly {
		prompt += "Please provide an improved version of the YAML that addresses the warnings where appropriate. "
	} else {
		prompt += "Please provide a corrected version of the YAML that fixes all the errors. "
	}

	prompt += "Only output the corrected YAML in a code block, no explanations needed."

	return prompt
}

func (h *AIHandler) isModelAllowed(model string) bool {
	if len(h.config.AllowedModels) == 0 {
		return true
	}

	for _, allowed := range h.config.AllowedModels {
		if allowed == model {
			return true
		}
	}

	return false
}

func (h *AIHandler) buildMessages(userMessages []ai.ChatMessage, currentYaml string) []ai.ChatMessage {
	messages := make([]ai.ChatMessage, 0, len(userMessages)+2)

	// Add system prompt
	systemPrompt := ai.BuildSystemPrompt()
	messages = append(messages, ai.ChatMessage{
		Role:    "system",
		Content: systemPrompt,
	})

	// Add current YAML context if provided
	if currentYaml != "" {
		messages = append(messages, ai.ChatMessage{
			Role:    "system",
			Content: "The user is currently working on the following test configuration:\n\n```yaml\n" + currentYaml + "\n```",
		})
	}

	// Add user messages
	messages = append(messages, userMessages...)

	return messages
}

// extractYamlFromResponse extracts YAML code blocks from the AI response.
func extractYamlFromResponse(response string) string {
	// Match ```yaml ... ``` blocks
	re := regexp.MustCompile("(?s)```ya?ml\\s*\n(.*?)```")
	matches := re.FindAllStringSubmatch(response, -1)

	if len(matches) == 0 {
		return ""
	}

	// Return the first YAML block found
	return matches[0][1]
}

func (h *AIHandler) sendOKResponse(w http.ResponseWriter, route string, data any) {
	j := json.NewEncoder(w)
	response := &Response{
		Status: "OK",
		Data:   data,
	}

	err := j.Encode(response)
	if err != nil {
		h.logger.Errorf("error serializing json data for API %v route: %v", route, err)
	}
}

func (h *AIHandler) sendErrorResponse(w http.ResponseWriter, route, message string, errorcode int) {
	w.WriteHeader(errorcode)
	j := json.NewEncoder(w)
	response := &Response{Status: "ERROR: " + message}

	err := j.Encode(response)
	if err != nil {
		h.logger.Errorf("error serializing json error for API %v route: %v", route, err)
	}
}

func (h *AIHandler) sendUnauthorizedResponse(w http.ResponseWriter, route string) {
	h.sendErrorResponse(w, route, "unauthorized", http.StatusUnauthorized)
}
