package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type GetTestYamlResponse struct {
	Yaml   string `json:"yaml"`
	Source string `json:"source"`
}

// GetTestYaml godoc
// @Id getTestYaml
// @Summary Get test YAML source by test ID
// @Tags Test
// @Description Returns the full YAML source for a test definition. For API-registered tests, returns the stored YAML. For external tests referenced by file/URL, loads and returns the content from the original source. Requires authentication as YAML may contain sensitive configuration.
// @Produce json
// @Param testId path string true "ID of the test definition to get YAML for"
// @Success 200 {object} Response{data=GetTestYamlResponse} "Success"
// @Failure 400 {object} Response "Failure"
// @Failure 401 {object} Response "Unauthorized"
// @Failure 404 {object} Response "Not Found"
// @Failure 500 {object} Response "Server Error"
// @Router /api/v1/test/{testId}/yaml [get]
func (ah *APIHandler) GetTestYaml(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", contentTypeJSON)

	// Check authentication - YAML may contain sensitive configuration
	if !ah.checkAuth(r) {
		ah.sendUnauthorizedResponse(w, r.URL.String())
		return
	}

	vars := mux.Vars(r)
	if vars["testId"] == "" {
		ah.sendErrorResponse(w, r.URL.String(), "testId missing", http.StatusBadRequest)
		return
	}

	testID := vars["testId"]

	// Get test config from database
	testConfig, err := ah.coordinator.Database().GetTestConfig(testID)
	if err != nil {
		ah.sendErrorResponse(w, r.URL.String(), fmt.Sprintf("test not found: %v", err), http.StatusNotFound)
		return
	}

	var yamlContent string

	var source string

	// Determine YAML source type and load content
	switch {
	case testConfig.YamlSource != "":
		// Stored YAML source (API-registered tests)
		yamlContent = testConfig.YamlSource
		source = "database"

	case testConfig.Source != "" && testConfig.Source != "api-call":
		// Load from external file/URL
		var loadErr error

		yamlContent, loadErr = ah.loadExternalYaml(r.Context(), testConfig.Source)
		if loadErr != nil {
			ah.sendErrorResponse(w, r.URL.String(), fmt.Sprintf("failed to load external YAML: %v", loadErr), http.StatusInternalServerError)
			return
		}

		source = testConfig.Source

	default:
		// No YAML source available
		ah.sendErrorResponse(w, r.URL.String(), "no YAML source available for this test", http.StatusNotFound)
		return
	}

	response := &GetTestYamlResponse{
		Yaml:   yamlContent,
		Source: source,
	}

	ah.sendOKResponse(w, r.URL.String(), response)
}

// loadExternalYaml loads YAML content from a file path or URL.
func (ah *APIHandler) loadExternalYaml(ctx context.Context, source string) (string, error) {
	// Strip "external:" prefix if present (added by test registry)
	cleanSource := strings.TrimPrefix(source, "external:")

	// Check if source is a URL
	if strings.HasPrefix(cleanSource, "http://") || strings.HasPrefix(cleanSource, "https://") {
		client := &http.Client{Timeout: time.Second * 120}

		req, err := http.NewRequestWithContext(ctx, "GET", cleanSource, http.NoBody)
		if err != nil {
			return "", fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := client.Do(req)
		if err != nil {
			return "", fmt.Errorf("failed to fetch URL: %w", err)
		}

		defer func() {
			if closeErr := resp.Body.Close(); closeErr != nil {
				ah.coordinator.Logger().WithError(closeErr).Warn("failed to close response body")
			}
		}()

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("HTTP error: %v %v", resp.StatusCode, resp.Status)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to read response: %w", err)
		}

		return string(body), nil
	}

	// It's a local file path - read the file (this endpoint is auth-protected)
	body, err := os.ReadFile(cleanSource)
	if err != nil {
		return "", fmt.Errorf("cannot load local file %s: %w", cleanSource, err)
	}

	return string(body), nil
}
