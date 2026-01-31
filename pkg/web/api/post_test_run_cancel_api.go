package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"gopkg.in/yaml.v3"
)

type PostTestRunCancelRequest struct {
	TestID      string `json:"test_id"`
	SkipCleanup bool   `json:"skip_cleanup"`
}

type PostTestRunCancelResponse struct {
	TestID string `json:"test_id"`
	RunID  uint64 `json:"run_id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

// PostTestRunCancel godoc
// @Id postTestRunCancel
// @Summary Cancel test run by test ID
// @Tags TestRun
// @Description Returns the test/run id & status of the cancelled test.
// @Produce json
// @Param runId path string true "ID of the test run to cancel"
// @Param cancelOptions body PostTestRunCancelRequest true "Test cancellation options"
// @Success 200 {object} Response{data=PostTestRunCancelResponse} "Success"
// @Failure 400 {object} Response "Failure"
// @Failure 500 {object} Response "Server Error"
// @Router /api/v1/test_run/{runId}/cancel [post]
func (ah *APIHandler) PostTestRunCancel(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", contentTypeJSON)

	// parse request body
	req := &PostTestRunCancelRequest{}

	if r.Header.Get("Content-Type") == contentTypeYAML {
		decoder := yaml.NewDecoder(r.Body)

		err := decoder.Decode(req)
		if err != nil {
			ah.sendErrorResponse(w, r.URL.String(), fmt.Sprintf("error decoding request body yaml: %v", err), http.StatusBadRequest)
			return
		}
	} else {
		decoder := json.NewDecoder(r.Body)

		err := decoder.Decode(req)
		if err != nil {
			ah.sendErrorResponse(w, r.URL.String(), fmt.Sprintf("error decoding request body json: %v", err), http.StatusBadRequest)
			return
		}
	}

	// get test run by id
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

	// check if test ID matches
	if testInstance.TestID() != req.TestID {
		ah.sendErrorResponse(w, r.URL.String(), "test id does not match", http.StatusNotFound)
		return
	}

	// cancel test run
	testInstance.AbortTest(req.SkipCleanup)

	ah.sendOKResponse(w, r.URL.String(), &PostTestRunCancelResponse{
		TestID: testInstance.TestID(),
		RunID:  testInstance.RunID(),
		Name:   testInstance.Name(),
		Status: string(testInstance.Status()),
	})
}
