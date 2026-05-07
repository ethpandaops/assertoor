package tysmhookactivation

import (
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ethpandaops/assertoor/pkg/helper"
	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/ethpandaops/assertoor/pkg/vars"
	"github.com/sirupsen/logrus"
)

const testToken = "test-token"

func newTestTask(cfg *Config) *Task {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	return &Task{
		ctx: &types.TaskContext{
			Outputs:        vars.NewVariables(nil),
			SetResult:      func(types.TaskResult) {},
			ReportProgress: func(float64, string) {},
		},
		config: *cfg,
		logger: logger,
	}
}

func boolPtr(b bool) *bool { return &b }

func validBaseConfig() Config {
	return Config{
		Endpoint:    "http://example.invalid",
		Hook:        "blob-mutator",
		Enabled:     boolPtr(true),
		ConfigPatch: map[string]interface{}{"k": "v"},
		Duration:    helper.Duration{Duration: time.Minute},
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(c *Config)
		wantErr string
	}{
		{
			name:   "valid",
			mutate: func(*Config) {},
		},
		{
			name:    "missing endpoint",
			mutate:  func(c *Config) { c.Endpoint = "" },
			wantErr: "endpoint is required",
		},
		{
			name:    "missing hook",
			mutate:  func(c *Config) { c.Hook = "" },
			wantErr: "hook is required",
		},
		{
			name:    "zero duration",
			mutate:  func(c *Config) { c.Duration = helper.Duration{} },
			wantErr: "duration must be greater than 0",
		},
		{
			name: "neither enabled nor configPatch",
			mutate: func(c *Config) {
				c.Enabled = nil
				c.ConfigPatch = nil
			},
			wantErr: "at least one of enabled or configPatch must be set",
		},
		{
			name: "only enabled is sufficient",
			mutate: func(c *Config) {
				c.ConfigPatch = nil
			},
		},
		{
			name: "only configPatch is sufficient",
			mutate: func(c *Config) {
				c.Enabled = nil
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := validBaseConfig()
			tc.mutate(&cfg)

			err := cfg.Validate()

			switch {
			case tc.wantErr == "" && err != nil:
				t.Fatalf("unexpected error: %v", err)
			case tc.wantErr != "" && err == nil:
				t.Fatalf("expected error containing %q, got nil", tc.wantErr)
			case tc.wantErr != "" && !strings.Contains(err.Error(), tc.wantErr):
				t.Fatalf("error %q does not contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestExecute_HappyPath(t *testing.T) {
	var (
		capturedAuth   string
		capturedBody   activationRequest
		capturedMethod string
		capturedPath   string
	)

	expiry := time.Now().Add(2 * time.Minute).UTC().Round(time.Second)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		capturedMethod = r.Method
		capturedPath = r.URL.Path

		if err := json.NewDecoder(r.Body).Decode(&capturedBody); err != nil {
			t.Errorf("decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(activationResponse{
			ID:        "act_abc",
			Hook:      "blob-mutator",
			CreatedAt: time.Now().UTC(),
			ExpiresAt: expiry,
		})
	}))
	defer srv.Close()

	task := newTestTask(&Config{
		Endpoint:    srv.URL,
		AuthToken:   testToken,
		Hook:        "blob-mutator",
		Enabled:     boolPtr(true),
		ConfigPatch: map[string]interface{}{"mutationProbability": 1.0},
		Duration:    helper.Duration{Duration: 90 * time.Second},
		Replace:     true,
	})

	if err := task.Execute(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedMethod != http.MethodPost {
		t.Errorf("method: got %q, want %q", capturedMethod, http.MethodPost)
	}

	if capturedPath != "/tysm/v1/activations" {
		t.Errorf("path: got %q, want /tysm/v1/activations", capturedPath)
	}

	if want := "Bearer " + testToken; capturedAuth != want {
		t.Errorf("auth header: got %q, want %q", capturedAuth, want)
	}

	if capturedBody.Hook != "blob-mutator" {
		t.Errorf("body.hook: got %q, want blob-mutator", capturedBody.Hook)
	}

	if capturedBody.Duration != "1m30s" {
		t.Errorf("body.duration: got %q, want 1m30s", capturedBody.Duration)
	}

	if !capturedBody.Replace {
		t.Errorf("body.replace: got false, want true")
	}

	if capturedBody.Enabled == nil || !*capturedBody.Enabled {
		t.Errorf("body.enabled: got %v, want pointer to true", capturedBody.Enabled)
	}

	probAny, ok := capturedBody.ConfigPatch["mutationProbability"]
	if !ok {
		t.Fatalf("body.config_patch missing mutationProbability key")
	}

	prob, ok := probAny.(float64)
	if !ok {
		t.Fatalf("body.config_patch.mutationProbability: got %T, want float64", probAny)
	}

	if math.Abs(prob-1.0) > 0 {
		t.Errorf("body.config_patch.mutationProbability: got %v, want 1.0", prob)
	}

	if got := task.ctx.Outputs.GetVar("activation_id"); got != "act_abc" {
		t.Errorf("output activation_id: got %v, want act_abc", got)
	}

	if got, want := task.ctx.Outputs.GetVar("expires_at"), expiry.Format(time.RFC3339); got != want {
		t.Errorf("output expires_at: got %v, want %v", got, want)
	}

	if got := task.ctx.Outputs.GetVar("hook"); got != "blob-mutator" {
		t.Errorf("output hook: got %v, want blob-mutator", got)
	}
}

func TestExecute_NoAuthHeader_WhenTokenEmpty(t *testing.T) {
	var capturedAuth string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(activationResponse{ID: "id", ExpiresAt: time.Now()})
	}))
	defer srv.Close()

	task := newTestTask(&Config{
		Endpoint: srv.URL,
		Hook:     "blob-mutator",
		Enabled:  boolPtr(true),
		Duration: helper.Duration{Duration: time.Minute},
	})

	if err := task.Execute(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedAuth != "" {
		t.Errorf("expected no Authorization header when AuthToken is empty, got %q", capturedAuth)
	}
}

func TestExecute_ServerErrors(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       interface{}
		wantSubstr string
	}{
		{
			name:       "401 unauthorized",
			status:     http.StatusUnauthorized,
			body:       errorResponse{Error: "missing bearer"},
			wantSubstr: "HTTP 401: missing bearer",
		},
		{
			name:       "400 with details",
			status:     http.StatusBadRequest,
			body:       errorResponse{Error: "hook is not runtime_reconfigurable", Details: "metrics"},
			wantSubstr: "HTTP 400: hook is not runtime_reconfigurable (metrics)",
		},
		{
			name:       "409 conflict",
			status:     http.StatusConflict,
			body:       errorResponse{Error: "activation already exists for hook"},
			wantSubstr: "HTTP 409: activation already exists for hook",
		},
		{
			name:       "500 with non-JSON body",
			status:     http.StatusInternalServerError,
			body:       "internal boom",
			wantSubstr: "HTTP 500: internal boom",
		},
		{
			name:       "503 with empty body",
			status:     http.StatusServiceUnavailable,
			body:       "",
			wantSubstr: "HTTP 503",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)

				switch v := tc.body.(type) {
				case string:
					_, _ = io.WriteString(w, v)
				default:
					_ = json.NewEncoder(w).Encode(tc.body)
				}
			}))
			defer srv.Close()

			task := newTestTask(&Config{
				Endpoint: srv.URL,
				Hook:     "blob-mutator",
				Enabled:  boolPtr(true),
				Duration: helper.Duration{Duration: time.Minute},
			})

			err := task.Execute(context.Background())

			switch {
			case err == nil:
				t.Fatalf("expected error containing %q, got nil", tc.wantSubstr)
			case !strings.Contains(err.Error(), tc.wantSubstr):
				t.Fatalf("error %q does not contain %q", err.Error(), tc.wantSubstr)
			}

			if got := task.ctx.Outputs.GetVar("activation_id"); got != nil {
				t.Errorf("expected no activation_id output on failure, got %v", got)
			}
		})
	}
}

func TestExecute_201WithEmptyID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(activationResponse{ID: "", ExpiresAt: time.Now()})
	}))
	defer srv.Close()

	task := newTestTask(&Config{
		Endpoint: srv.URL,
		Hook:     "blob-mutator",
		Enabled:  boolPtr(true),
		Duration: helper.Duration{Duration: time.Minute},
	})

	err := task.Execute(context.Background())

	switch {
	case err == nil:
		t.Fatalf("expected error, got nil")
	case !strings.Contains(err.Error(), "no activation id"):
		t.Fatalf("error %q does not contain 'no activation id'", err.Error())
	}
}

func TestExecute_ConnectionRefused(t *testing.T) {
	// Bind a server, capture its URL, then close it so the connection is
	// guaranteed to be refused at the captured port.
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL

	srv.Close()

	task := newTestTask(&Config{
		Endpoint: url,
		Hook:     "blob-mutator",
		Enabled:  boolPtr(true),
		Duration: helper.Duration{Duration: time.Minute},
	})

	err := task.Execute(context.Background())

	switch {
	case err == nil:
		t.Fatalf("expected error, got nil")
	case !strings.Contains(err.Error(), "POST "+url+"/tysm/v1/activations"):
		t.Fatalf("error %q missing expected POST URL", err.Error())
	}
}

func TestExecute_TrimsTrailingSlash(t *testing.T) {
	var capturedPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(activationResponse{ID: "act_x", ExpiresAt: time.Now()})
	}))
	defer srv.Close()

	task := newTestTask(&Config{
		Endpoint: srv.URL + "/",
		Hook:     "blob-mutator",
		Enabled:  boolPtr(true),
		Duration: helper.Duration{Duration: time.Minute},
	})

	if err := task.Execute(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedPath != "/tysm/v1/activations" {
		t.Errorf("path: got %q, want /tysm/v1/activations", capturedPath)
	}
}
