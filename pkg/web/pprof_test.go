package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
)

const (
	testToken = "valid-token"
	bodyPprof = "pprof"
	bodyGate  = "gate"
)

// acceptToken accepts testToken, bare or as a bearer header.
func acceptToken(tokenStr, _ string) *jwt.Token {
	if tokenStr == testToken || tokenStr == "Bearer "+testToken {
		return &jwt.Token{Valid: true}
	}

	return nil
}

func openMode(_, _ string) *jwt.Token {
	return &jwt.Token{Valid: true}
}

func TestPprofHandler(t *testing.T) {
	t.Parallel()

	pprofMux := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(bodyPprof))
	})
	gate := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(bodyGate))
	})

	tests := []struct {
		name       string
		checkToken tokenChecker
		gate       http.Handler
		method     string
		path       string
		header     http.Header
		cookie     string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "open mode passes without token",
			checkToken: openMode,
			gate:       gate,
			path:       "/debug/pprof/heap",
			wantStatus: http.StatusOK,
			wantBody:   bodyPprof,
		},
		{
			name:       "bearer header passes",
			checkToken: acceptToken,
			gate:       gate,
			path:       "/debug/pprof/heap",
			header:     http.Header{"Authorization": {"Bearer " + testToken}},
			wantStatus: http.StatusOK,
			wantBody:   bodyPprof,
		},
		{
			name:       "cookie passes",
			checkToken: acceptToken,
			gate:       gate,
			path:       "/debug/pprof/heap",
			cookie:     testToken,
			wantStatus: http.StatusOK,
			wantBody:   bodyPprof,
		},
		{
			name:       "invalid header is not rescued by cookie",
			checkToken: acceptToken,
			gate:       gate,
			path:       "/debug/pprof/heap",
			header:     http.Header{"Authorization": {"Bearer wrong"}},
			cookie:     testToken,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "tool without token gets 401",
			checkToken: acceptToken,
			gate:       gate,
			path:       "/debug/pprof/heap",
			header:     http.Header{"Accept": {"*/*"}},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "browser without token gets the gate",
			checkToken: acceptToken,
			gate:       gate,
			path:       "/debug/pprof/",
			header:     http.Header{"Accept": {"text/html,application/xhtml+xml"}},
			wantStatus: http.StatusOK,
			wantBody:   bodyGate,
		},
		{
			name:       "browser POST without token gets 401",
			checkToken: acceptToken,
			gate:       gate,
			method:     http.MethodPost,
			path:       "/debug/pprof/",
			header:     http.Header{"Accept": {"text/html"}},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "browser without gate gets 401",
			checkToken: acceptToken,
			path:       "/debug/pprof/",
			header:     http.Header{"Accept": {"text/html"}},
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := newPprofHandler(logrus.New(), tt.checkToken, tt.gate, pprofMux)

			method := tt.method
			if method == "" {
				method = http.MethodGet
			}

			req := httptest.NewRequest(method, tt.path, http.NoBody)
			for k, v := range tt.header {
				req.Header[k] = v
			}

			if tt.cookie != "" {
				req.AddCookie(&http.Cookie{Name: debugTokenCookie, Value: tt.cookie})
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			if tt.wantBody != "" && rec.Body.String() != tt.wantBody {
				t.Errorf("body = %q, want %q", rec.Body.String(), tt.wantBody)
			}

			if tt.wantStatus == http.StatusUnauthorized && rec.Header().Get("WWW-Authenticate") != "Bearer" {
				t.Errorf("WWW-Authenticate = %q, want Bearer", rec.Header().Get("WWW-Authenticate"))
			}
		})
	}
}
