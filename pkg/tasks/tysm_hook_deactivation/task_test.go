package tysmhookdeactivation

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr string
	}{
		{
			name: "valid",
			cfg: Config{
				Endpoint:     "http://example.invalid",
				ActivationID: "act_1",
			},
		},
		{
			name: "missing endpoint",
			cfg: Config{
				ActivationID: "act_1",
			},
			wantErr: "endpoint is required",
		},
		{
			name: "missing activation_id",
			cfg: Config{
				Endpoint: "http://example.invalid",
			},
			wantErr: "activation_id is required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()

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
		capturedMethod string
		capturedPath   string
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		capturedMethod = r.Method
		capturedPath = r.URL.Path

		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	task := newTestTask(&Config{
		Endpoint:     srv.URL,
		AuthToken:    testToken,
		ActivationID: "act_abc",
	})

	if err := task.Execute(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedMethod != http.MethodDelete {
		t.Errorf("method: got %q, want %q", capturedMethod, http.MethodDelete)
	}

	if capturedPath != "/tysm/v1/activations/act_abc" {
		t.Errorf("path: got %q, want /tysm/v1/activations/act_abc", capturedPath)
	}

	if want := "Bearer " + testToken; capturedAuth != want {
		t.Errorf("auth header: got %q, want %q", capturedAuth, want)
	}
}

func TestExecute_NoAuthHeader_WhenTokenEmpty(t *testing.T) {
	var capturedAuth string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")

		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	task := newTestTask(&Config{
		Endpoint:     srv.URL,
		ActivationID: "act_1",
	})

	if err := task.Execute(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedAuth != "" {
		t.Errorf("expected no Authorization header when AuthToken is empty, got %q", capturedAuth)
	}
}

func TestExecute_404_IgnoreNotFound_True(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(errorResponse{Error: "unknown activation"})
	}))
	defer srv.Close()

	task := newTestTask(&Config{
		Endpoint:       srv.URL,
		ActivationID:   "act_gone",
		IgnoreNotFound: true,
	})

	if err := task.Execute(context.Background()); err != nil {
		t.Fatalf("404 should be treated as success when ignoreNotFound is true; got %v", err)
	}
}

func TestExecute_404_IgnoreNotFound_False(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(errorResponse{Error: "unknown activation"})
	}))
	defer srv.Close()

	task := newTestTask(&Config{
		Endpoint:       srv.URL,
		ActivationID:   "act_gone",
		IgnoreNotFound: false,
	})

	err := task.Execute(context.Background())

	switch {
	case err == nil:
		t.Fatalf("expected error, got nil")
	case !strings.Contains(err.Error(), "HTTP 404"):
		t.Errorf("error %q missing HTTP 404", err.Error())
	case !strings.Contains(err.Error(), "unknown activation"):
		t.Errorf("error %q missing 'unknown activation'", err.Error())
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
			name:       "401",
			status:     http.StatusUnauthorized,
			body:       errorResponse{Error: "missing bearer"},
			wantSubstr: "HTTP 401: missing bearer",
		},
		{
			name:       "503 unavailable during shutdown",
			status:     http.StatusServiceUnavailable,
			body:       errorResponse{Error: "server is shutting down"},
			wantSubstr: "HTTP 503: server is shutting down",
		},
		{
			name:       "500 with non-JSON body",
			status:     http.StatusInternalServerError,
			body:       "boom",
			wantSubstr: "HTTP 500: boom",
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
				Endpoint:     srv.URL,
				ActivationID: "act_1",
			})

			err := task.Execute(context.Background())

			switch {
			case err == nil:
				t.Fatalf("expected error containing %q, got nil", tc.wantSubstr)
			case !strings.Contains(err.Error(), tc.wantSubstr):
				t.Errorf("error %q does not contain %q", err.Error(), tc.wantSubstr)
			}
		})
	}
}

func TestExecute_PathEscapesActivationID(t *testing.T) {
	// RawPath preserves the on-the-wire encoded form whenever it differs
	// from the decoded Path; that's what we want to assert against.
	var capturedRawPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRawPath = r.URL.RawPath

		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	// An activation id with characters that would otherwise corrupt the
	// URL — confirms url.PathEscape is in play. Note PathEscape leaves
	// '&' alone (a sub-delim valid in path segments).
	task := newTestTask(&Config{
		Endpoint:     srv.URL,
		ActivationID: "act/with spaces?and&stuff",
	})

	if err := task.Execute(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "/tysm/v1/activations/act%2Fwith%20spaces%3Fand&stuff"
	if capturedRawPath != want {
		t.Errorf("raw path: got %q, want %q", capturedRawPath, want)
	}
}

func TestExecute_TrimsTrailingSlash(t *testing.T) {
	var capturedPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path

		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	task := newTestTask(&Config{
		Endpoint:     srv.URL + "/",
		ActivationID: "act_x",
	})

	if err := task.Execute(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedPath != "/tysm/v1/activations/act_x" {
		t.Errorf("path: got %q, want /tysm/v1/activations/act_x", capturedPath)
	}
}

func TestExecute_ConnectionRefused(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL

	srv.Close()

	task := newTestTask(&Config{
		Endpoint:     url,
		ActivationID: "act_x",
	})

	err := task.Execute(context.Background())

	switch {
	case err == nil:
		t.Fatalf("expected error, got nil")
	case !strings.Contains(err.Error(), "DELETE "+url+"/tysm/v1/activations/act_x"):
		t.Errorf("error %q missing expected DELETE URL", err.Error())
	}
}
