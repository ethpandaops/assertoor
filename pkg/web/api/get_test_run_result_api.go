package api

import (
	"bytes"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// GetTestRunResult godoc
// @Id getTestRunResult
// @Summary Get the run-level result markdown
// @Tags TestRun
// @Description Returns the markdown blob that tasks have collectively
// @Description written to $ASSERTOOR_TEST_RESULT during the run, if any.
// @Description Rendered by the UI as the run's prominent Result panel.
// @Produce text/markdown
// @Produce json
// @Param runId path string true "ID of the test run"
// @Success 200 {string} string "Markdown body"
// @Success 204 "No result set for this run"
// @Failure 400 {object} Response "Bad Request"
// @Failure 404 {object} Response "Test run not found"
// @Router /api/v1/test_run/{runId}/result [get]
func (ah *APIHandler) GetTestRunResult(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	runID, err := strconv.ParseUint(vars["runId"], 10, 64)
	if err != nil {
		w.Header().Set("Content-Type", contentTypeJSON)
		ah.sendErrorResponse(w, r.URL.String(), "invalid runId provided", http.StatusBadRequest)

		return
	}

	// Confirm the test run exists at all (returns 404 distinctly from
	// the "no result yet" 204 case).
	if testInstance := ah.coordinator.GetTestByRunID(runID); testInstance == nil {
		w.Header().Set("Content-Type", contentTypeJSON)
		ah.sendErrorResponse(w, r.URL.String(), "test run not found", http.StatusNotFound)

		return
	}

	result, err := ah.coordinator.Database().GetTestResult(runID)
	if err != nil {
		w.Header().Set("Content-Type", contentTypeJSON)
		ah.sendErrorResponse(w, r.URL.String(), "failed to fetch test result", http.StatusInternalServerError)

		return
	}

	if result == nil || len(result.Data) == 0 {
		w.WriteHeader(http.StatusNoContent)

		return
	}

	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	http.ServeContent(w, r, "result.md", time.Now(), bytes.NewReader(result.Data))
}
