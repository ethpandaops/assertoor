package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/jmoiron/sqlx"
)

// dashboardConfigStateKey is the `assertoor_state` key under which
// the dashboard config blob is stored. Using the existing KV store
// avoids a dedicated table / migration just to hold one row.
const dashboardConfigStateKey = "dashboard_config"

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
	var raw json.RawMessage

	_, err := ah.coordinator.Database().GetAssertoorState(dashboardConfigStateKey, &raw)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNoContent)

			return
		}

		w.Header().Set("Content-Type", contentTypeJSON)
		ah.sendErrorResponse(w, r.URL.String(), "failed to load dashboard config", http.StatusInternalServerError)

		return
	}

	if len(raw) == 0 {
		w.WriteHeader(http.StatusNoContent)

		return
	}

	w.Header().Set("Content-Type", contentTypeJSON)
	_, _ = w.Write(raw)
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

	// Validate "is this JSON" up-front — the KV store stores opaque
	// strings, so guarding against garbage is our job.
	if !json.Valid(body) {
		ah.sendErrorResponse(w, r.URL.String(), "body is not valid JSON", http.StatusBadRequest)

		return
	}

	// Persist as json.RawMessage so SetAssertoorState's MarshalJSON
	// step preserves the bytes verbatim (RawMessage.MarshalJSON is
	// the identity). That way the column ends up holding the actual
	// dashboard JSON rather than a double-encoded string.
	raw := json.RawMessage(body)

	err = ah.coordinator.Database().RunTransaction(func(tx *sqlx.Tx) error {
		return ah.coordinator.Database().SetAssertoorState(tx, dashboardConfigStateKey, raw)
	})
	if err != nil {
		ah.sendErrorResponse(w, r.URL.String(), "failed to save dashboard config", http.StatusInternalServerError)

		return
	}

	ah.sendOKResponse(w, r.URL.String(), nil)
}
