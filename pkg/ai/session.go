package ai

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// SessionStatus represents the current state of an AI chat session.
type SessionStatus string

const (
	SessionStatusPending    SessionStatus = "pending"
	SessionStatusStreaming  SessionStatus = "streaming"
	SessionStatusValidating SessionStatus = "validating"
	SessionStatusFixing     SessionStatus = "fixing"
	SessionStatusComplete   SessionStatus = "complete"
	SessionStatusError      SessionStatus = "error"
)

// Session represents an AI chat session with streaming support.
type Session struct {
	ID        string        `json:"id"`
	Status    SessionStatus `json:"status"`
	CreatedAt time.Time     `json:"createdAt"`
	UpdatedAt time.Time     `json:"updatedAt"`

	// Response data
	Response      string            `json:"response"`
	GeneratedYaml string            `json:"generatedYaml,omitempty"`
	Validation    *ValidationResult `json:"validation,omitempty"`

	// Token usage
	Usage struct {
		PromptTokens     int `json:"promptTokens"`
		CompletionTokens int `json:"completionTokens"`
		TotalTokens      int `json:"totalTokens"`
	} `json:"usage"`

	// Error information
	Error string `json:"error,omitempty"`

	// Fix attempt tracking
	FixAttempts         int  `json:"fixAttempts"`
	MaxFixAttempts      int  `json:"-"`
	WarningFixAttempted bool `json:"warningFixAttempted"`

	// Internal state
	mu           sync.RWMutex
	cancelFunc   context.CancelFunc
	subscribers  []chan *Session
	lastPollTime time.Time
}

// SessionManager manages AI chat sessions.
type SessionManager struct {
	sessions map[string]*Session
	mu       sync.RWMutex

	// Cleanup configuration
	sessionTTL    time.Duration
	cleanupTicker *time.Ticker
	stopCleanup   chan struct{}
}

// NewSessionManager creates a new session manager.
func NewSessionManager() *SessionManager {
	sm := &SessionManager{
		sessions:      make(map[string]*Session),
		sessionTTL:    10 * time.Minute,
		cleanupTicker: time.NewTicker(1 * time.Minute),
		stopCleanup:   make(chan struct{}),
	}

	go sm.cleanupLoop()

	return sm
}

// Stop stops the session manager cleanup loop.
func (sm *SessionManager) Stop() {
	close(sm.stopCleanup)
	sm.cleanupTicker.Stop()
}

func (sm *SessionManager) cleanupLoop() {
	for {
		select {
		case <-sm.cleanupTicker.C:
			sm.cleanup()
		case <-sm.stopCleanup:
			return
		}
	}
}

func (sm *SessionManager) cleanup() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()

	for id, session := range sm.sessions {
		session.mu.RLock()
		isComplete := session.Status == SessionStatusComplete || session.Status == SessionStatusError
		lastActivity := session.lastPollTime

		if lastActivity.IsZero() {
			lastActivity = session.UpdatedAt
		}

		session.mu.RUnlock()

		// Remove sessions that are complete and haven't been polled recently
		if isComplete && now.Sub(lastActivity) > sm.sessionTTL {
			delete(sm.sessions, id)
		}
	}
}

// CreateSession creates a new session and returns it.
func (sm *SessionManager) CreateSession(maxFixAttempts int) *Session {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session := &Session{
		ID:             uuid.New().String(),
		Status:         SessionStatusPending,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		MaxFixAttempts: maxFixAttempts,
		subscribers:    make([]chan *Session, 0),
	}

	sm.sessions[session.ID] = session

	return session
}

// GetSession retrieves a session by ID.
func (sm *SessionManager) GetSession(id string) *Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session := sm.sessions[id]
	if session != nil {
		session.mu.Lock()
		session.lastPollTime = time.Now()
		session.mu.Unlock()
	}

	return session
}

// DeleteSession removes a session.
func (sm *SessionManager) DeleteSession(id string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if session, ok := sm.sessions[id]; ok {
		session.mu.Lock()

		if session.cancelFunc != nil {
			session.cancelFunc()
		}

		// Close all subscriber channels
		for _, ch := range session.subscribers {
			close(ch)
		}

		session.mu.Unlock()
		delete(sm.sessions, id)
	}
}

// Session methods

// UpdateStatus updates the session status and notifies subscribers.
func (s *Session) UpdateStatus(status SessionStatus) {
	s.mu.Lock()
	s.Status = status
	s.UpdatedAt = time.Now()
	s.mu.Unlock()
	s.notifySubscribers()
}

// SetResponse sets the response content.
func (s *Session) SetResponse(response string) {
	s.mu.Lock()
	s.Response = response
	s.UpdatedAt = time.Now()
	s.mu.Unlock()
	s.notifySubscribers()
}

// AppendResponse appends to the response (for streaming).
func (s *Session) AppendResponse(chunk string) {
	s.mu.Lock()
	s.Response += chunk
	s.UpdatedAt = time.Now()
	s.mu.Unlock()
	s.notifySubscribers()
}

// SetGeneratedYaml sets the generated YAML.
func (s *Session) SetGeneratedYaml(yaml string) {
	s.mu.Lock()
	s.GeneratedYaml = yaml
	s.UpdatedAt = time.Now()
	s.mu.Unlock()
	s.notifySubscribers()
}

// SetValidation sets the validation result.
func (s *Session) SetValidation(validation *ValidationResult) {
	s.mu.Lock()
	s.Validation = validation
	s.UpdatedAt = time.Now()
	s.mu.Unlock()
	s.notifySubscribers()
}

// SetUsage sets the token usage.
func (s *Session) SetUsage(prompt, completion, total int) {
	s.mu.Lock()
	s.Usage.PromptTokens += prompt
	s.Usage.CompletionTokens += completion
	s.Usage.TotalTokens += total
	s.UpdatedAt = time.Now()
	s.mu.Unlock()
}

// SetError sets an error and marks the session as failed.
func (s *Session) SetError(err string) {
	s.mu.Lock()
	s.Error = err
	s.Status = SessionStatusError
	s.UpdatedAt = time.Now()
	s.mu.Unlock()
	s.notifySubscribers()
}

// Complete marks the session as complete.
func (s *Session) Complete() {
	s.mu.Lock()
	s.Status = SessionStatusComplete
	s.UpdatedAt = time.Now()
	s.mu.Unlock()
	s.notifySubscribers()
}

// IncrementFixAttempts increments the fix attempt counter.
func (s *Session) IncrementFixAttempts() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.FixAttempts++

	return s.FixAttempts
}

// CanRetryFix checks if another fix attempt is allowed.
func (s *Session) CanRetryFix() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.FixAttempts < s.MaxFixAttempts
}

// CanFixWarnings checks if a warning fix attempt is allowed (only 1 allowed).
func (s *Session) CanFixWarnings() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return !s.WarningFixAttempted
}

// SetWarningFixAttempted marks that a warning fix was attempted.
func (s *Session) SetWarningFixAttempted() {
	s.mu.Lock()
	s.WarningFixAttempted = true
	s.mu.Unlock()
}

// SetCancelFunc sets the cancel function for the session.
func (s *Session) SetCancelFunc(cancel context.CancelFunc) {
	s.mu.Lock()
	s.cancelFunc = cancel
	s.mu.Unlock()
}

// Subscribe returns a channel that receives session updates.
func (s *Session) Subscribe() <-chan *Session {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch := make(chan *Session, 10)
	s.subscribers = append(s.subscribers, ch)

	return ch
}

// Unsubscribe removes a subscriber channel.
func (s *Session) Unsubscribe(ch <-chan *Session) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, sub := range s.subscribers {
		if sub == ch {
			close(sub)

			s.subscribers = append(s.subscribers[:i], s.subscribers[i+1:]...)

			break
		}
	}
}

func (s *Session) notifySubscribers() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, ch := range s.subscribers {
		select {
		case ch <- s:
		default:
			// Channel full, skip
		}
	}
}

// GetSnapshot returns a copy of the session for safe reading.
func (s *Session) GetSnapshot() *Session {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return &Session{
		ID:                  s.ID,
		Status:              s.Status,
		CreatedAt:           s.CreatedAt,
		UpdatedAt:           s.UpdatedAt,
		Response:            s.Response,
		GeneratedYaml:       s.GeneratedYaml,
		Validation:          s.Validation,
		Usage:               s.Usage,
		Error:               s.Error,
		FixAttempts:         s.FixAttempts,
		WarningFixAttempted: s.WarningFixAttempted,
	}
}
