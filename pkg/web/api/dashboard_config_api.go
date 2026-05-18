package api

import (
	"encoding/json"
	"io"
	"net/http"
)

// GetDashboardConfig godoc
// @Id getDashboardConfig
// @Summary Get the active dashboard config
// @Tags Dashboard
// @Description Returns the persisted dashboard layout (JSON blob).
// @Description Public read; mutations are auth-gated via PUT.
// @Description Returns 204 when no config has been saved yet — the
// @Description UI falls back to a built-in default in that case.
// @Produce json
// @Success 200 {object} json.RawMessage "Dashboard config JSON"
// @Success 204 "No config persisted yet"
// @Failure 500 {object} Response "Server Error"
// @Router /api/v1/dashboard_config [get]
func (ah *APIHandler) GetDashboardConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := ah.coordinator.Database().GetDashboardConfig()
	if err != nil {
		w.Header().Set("Content-Type", contentTypeJSON)
		ah.sendErrorResponse(w, r.URL.String(), "failed to load dashboard config", http.StatusInternalServerError)

		return
	}

	if cfg == nil || len(cfg.Data) == 0 {
		w.WriteHeader(http.StatusNoContent)

		return
	}

	w.Header().Set("Content-Type", contentTypeJSON)
	_, _ = w.Write(cfg.Data)
}

// PutDashboardConfig godoc
// @Id putDashboardConfig
// @Summary Replace the active dashboard config
// @Tags Dashboard
// @Description Auth-required. Persists the supplied JSON blob as the
// @Description active dashboard layout. The body is treated as
// @Description opaque — the server only checks that it parses as
// @Description valid JSON; semantic validation lives in the client.
// @Accept json
// @Produce json
// @Success 200 {object} Response "Saved"
// @Failure 400 {object} Response "Bad Request"
// @Failure 401 {object} Response "Unauthorized"
// @Failure 500 {object} Response "Server Error"
// @Router /api/v1/dashboard_config [put]
func (ah *APIHandler) PutDashboardConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", contentTypeJSON)

	if !ah.checkAuth(r) {
		ah.sendUnauthorizedResponse(w, r.URL.String())

		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		ah.sendErrorResponse(w, r.URL.String(), "failed to read body", http.StatusBadRequest)

		return
	}

	// Validate "is this JSON" — anything more strict belongs in the
	// client where the schema is defined.
	var probe any
	if err := json.Unmarshal(body, &probe); err != nil {
		ah.sendErrorResponse(w, r.URL.String(), "body is not valid JSON", http.StatusBadRequest)

		return
	}

	if err := ah.coordinator.Database().UpsertDashboardConfig(body); err != nil {
		ah.sendErrorResponse(w, r.URL.String(), "failed to save dashboard config", http.StatusInternalServerError)

		return
	}

	ah.sendOKResponse(w, r.URL.String(), nil)
}
