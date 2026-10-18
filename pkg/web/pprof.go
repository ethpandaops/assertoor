package web

import (
	"net/http"
	"strings"

	"github.com/ethpandaops/assertoor/pkg/web/auth"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
)

// pprofPathPrefix is the path the Go pprof handlers are mounted on.
const pprofPathPrefix = "/debug/pprof"

// debugTokenCookie carries the login token for the pprof pages in a
// browser. The web UI's gate page sets it from the session, scoped to
// pprofPathPrefix, SameSite=Strict and expiring with the token (at the
// latest after 30 minutes); see web-ui/src/pages/DebugLogin.tsx.
const debugTokenCookie = "assertoor_debug_token"

// tokenChecker validates a bearer token (with or without the "Bearer "
// prefix) for a request bound to host; see auth.Handler.CheckAuthToken.
type tokenChecker func(tokenStr, host string) *jwt.Token

// pprofHandler gates the pprof pages behind the login. A profile exposes
// the process memory (RPC credentials, private keys, ...) and costs CPU to
// collect, so unlike the rest of the UI it is never public once an auth
// provider is configured; in open mode every request passes.
//
// The login is taken from the Authorization header (tools: curl, go tool
// pprof) or from debugTokenCookie (a browser). A browser without a valid
// login gets the web UI instead, whose page for this path sets the cookie
// from the session and reloads, or asks for a login; everything else gets
// a 401. Every profile download is logged with the requesting user; the
// index page with its counts is not.
type pprofHandler struct {
	logger     logrus.FieldLogger
	checkToken tokenChecker
	gate       http.Handler // the web UI; nil when the frontend is disabled
	next       http.Handler
}

var _ http.Handler = (*pprofHandler)(nil)

// newPprofHandler wraps next (the pprof mux) in the login gate. gate is the
// web UI served to a browser without a login, or nil when there is none.
func newPprofHandler(logger logrus.FieldLogger, checkToken tokenChecker, gate, next http.Handler) http.Handler {
	return &pprofHandler{
		logger:     logger,
		checkToken: checkToken,
		gate:       gate,
		next:       next,
	}
}

// ServeHTTP implements http.Handler.
func (h *pprofHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tokenStr := r.Header.Get("Authorization")
	if tokenStr == "" {
		if cookie, err := r.Cookie(debugTokenCookie); err == nil {
			tokenStr = cookie.Value
		}
	}

	if token := h.checkToken(tokenStr, auth.StripPort(r.Host)); token != nil && token.Valid {
		h.serveProfile(w, r, token)

		return
	}

	if h.gate != nil && r.Method == http.MethodGet && strings.Contains(r.Header.Get("Accept"), "text/html") {
		w.Header().Set("Cache-Control", "no-store")
		h.gate.ServeHTTP(w, r)

		return
	}

	w.Header().Set("WWW-Authenticate", "Bearer")
	http.Error(w, "login required", http.StatusUnauthorized)
}

// serveProfile passes an authenticated request on to the pprof mux. A
// profile download (anything below the index page) is logged first.
func (h *pprofHandler) serveProfile(w http.ResponseWriter, r *http.Request, token *jwt.Token) {
	profile := strings.Trim(strings.TrimPrefix(r.URL.Path, pprofPathPrefix), "/")
	if profile != "" {
		h.logger.WithFields(logrus.Fields{
			"profile": profile,
			"query":   r.URL.RawQuery,
			"user":    auth.TokenSubject(token),
			"remote":  r.RemoteAddr,
		}).Info("pprof profile requested")
	}

	h.next.ServeHTTP(w, r)
}
