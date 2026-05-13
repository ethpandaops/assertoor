package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/ethpandaops/assertoor/pkg/web/static"
	"github.com/sirupsen/logrus"
)

// SPAHandler serves a React single-page application.
// It serves static files when they exist, otherwise falls back to index.html
// for client-side routing.
type SPAHandler struct {
	logger      logrus.FieldLogger
	staticFS    http.FileSystem
	indexHTML   []byte
	contentType string
}

// RuntimeConfig is injected into the served index.html as a nested global
// `window.ethpandaops.assertoor.config` so the SPA can read it
// synchronously at boot — no extra round-trip to the backend.
type RuntimeConfig struct {
	AuthProviderURL string `json:"authProviderURL"`
}

// NewSPAHandler creates a new SPA handler. The runtimeConfig is encoded
// into a small <script> block injected into the <head> of index.html
// before any other script runs.
func NewSPAHandler(logger logrus.FieldLogger, runtimeConfig RuntimeConfig) (*SPAHandler, error) {
	fs := http.FS(static.FS)

	// Pre-load index.html for faster serving
	indexFile, err := static.FS.Open("index.html")
	if err != nil {
		return nil, err
	}
	defer indexFile.Close()

	indexHTML, err := io.ReadAll(indexFile)
	if err != nil {
		return nil, err
	}

	indexHTML, err = injectHead(indexHTML, runtimeConfig)
	if err != nil {
		return nil, err
	}

	return &SPAHandler{
		logger:      logger,
		staticFS:    fs,
		indexHTML:   indexHTML,
		contentType: "text/html; charset=utf-8",
	}, nil
}

// injectHead inserts a runtime-config <script> immediately before
// </head>. The runtime config JSON is HTML-safe (encoding/json escapes
// <, >, & by default), so values can't break out of the script tag.
func injectHead(html []byte, cfg RuntimeConfig) ([]byte, error) {
	payload, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshal runtime config: %w", err)
	}

	injected := []byte(
		"<script>" +
			"window.ethpandaops=window.ethpandaops||{};" +
			"window.ethpandaops.assertoor=window.ethpandaops.assertoor||{};" +
			"window.ethpandaops.assertoor.config=" + string(payload) + ";" +
			"</script>",
	)

	headClose := []byte("</head>")

	idx := bytes.Index(html, headClose)
	if idx < 0 {
		return append(injected, html...), nil
	}

	out := make([]byte, 0, len(html)+len(injected))
	out = append(out, html[:idx]...)
	out = append(out, injected...)
	out = append(out, html[idx:]...)

	return out, nil
}

// ServeHTTP implements http.Handler.
func (h *SPAHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Clean the path
	urlPath := path.Clean(r.URL.Path)
	if !strings.HasPrefix(urlPath, "/") {
		urlPath = "/" + urlPath
	}

	// Try to serve static file
	if h.serveStaticFile(w, r, urlPath) {
		return
	}

	// Fall back to index.html for SPA routing
	h.serveIndex(w, r)
}

// serveStaticFile attempts to serve a static file. Returns true if successful.
func (h *SPAHandler) serveStaticFile(w http.ResponseWriter, r *http.Request, urlPath string) bool {
	// Don't serve index.html directly - let SPA handle root
	if urlPath == "/" || urlPath == "/index.html" {
		return false
	}

	// Check if the file exists
	f, err := h.staticFS.Open(urlPath)
	if err != nil {
		return false
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil || stat.IsDir() {
		return false
	}

	// Check if file implements ReadSeeker
	rs, ok := f.(io.ReadSeeker)
	if !ok {
		return false
	}

	// Serve the static file
	http.ServeContent(w, r, urlPath, stat.ModTime(), rs)

	return true
}

// serveIndex serves the SPA index.html.
func (h *SPAHandler) serveIndex(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", h.contentType)
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(h.indexHTML); err != nil {
		h.logger.WithError(err).Error("failed to write index.html")
	}
}
