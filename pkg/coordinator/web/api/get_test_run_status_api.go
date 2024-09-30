package api

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// GetTestRunStatus godoc
// @Id getTestRunStatus
// @Summary Get test run status by run ID
// @Tags TestRun
// @Description Returns the run status with given ID.
// @Produce json
// @Param runId path string true "ID of the test run to get the status for"
// @Success 200 {object} Response{data=string} "Success"
// @Failure 400 {object} Response "Failure"
// @Failure 500 {object} Response "Server Error"
// @Router /api/v1/test_run/{runId}/status [get]
func (ah *APIHandler) GetTestRunStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", contentTypeJSON)

	vars := mux.Vars(r)

	runID, err := strconv.ParseUint(vars["runId"], 10, 64)
	if err != nil {
		ah.sendErrorResponse(w, r.URL.String(), "invalid runId provided", http.StatusBadRequest)
		return
	}

	testInstance := ah.coordinator.GetTestByRunID(runID)
	if testInstance == nil {
		ah.sendErrorResponse(w, r.URL.String(), "test run not found", http.StatusNotFound)
		return
	}

	ah.sendOKResponse(w, r.URL.String(), string(testInstance.Status()))
}
