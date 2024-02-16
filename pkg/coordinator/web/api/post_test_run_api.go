package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/gorilla/mux"
)

type PostTestRunRequest struct {
	Config map[string]any `json:"config"`
}

type PostTestRunResponse struct {
	TestID string         `json:"test_id"`
	RunID  uint64         `json:"run_id"`
	Name   string         `json:"name"`
	Config map[string]any `json:"config"`
}

// PostTestRun godoc
// @Id postTestRun
// @Summary Schedule new test run by test ID
// @Tags Test
// @Description Returns the test & run id of the scheduled test execution.
// @Produce json
// @Param testId path string true "ID of the test definition to schedule a test run for"
// @Param runOptions body PostTestRunRequest true "Rest run options"
// @Success 200 {object} Response{data=PostTestRunResponse} "Success"
// @Failure 400 {object} Response "Failure"
// @Failure 500 {object} Response "Server Error"
// @Router /api/v1/test/{testId}/run [post]
func (ah *APIHandler) PostTestRun(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	if vars["testId"] == "" {
		ah.sendErrorResponse(w, r.URL.String(), "testId missing", http.StatusBadRequest)
		return
	}

	var testDescriptor types.TestDescriptor

	for _, testDescr := range ah.coordinator.GetTestDescriptors() {
		if testDescr.Err() != nil {
			continue
		}

		if testDescr.ID() == vars["testId"] {
			testDescriptor = testDescr
			break
		}
	}

	if testDescriptor == nil {
		ah.sendErrorResponse(w, r.URL.String(), "test not found", http.StatusNotFound)
		return
	}

	// check if test is already scheduled
	for _, testInstance := range ah.coordinator.GetTestQueue() {
		if testInstance.TestID() == testDescriptor.ID() {
			ah.sendErrorResponse(w, r.URL.String(), fmt.Sprintf("test already scheduled (run_id: %v)", testInstance.RunID()), http.StatusTooManyRequests)
			return
		}
	}

	// parse request body
	decoder := json.NewDecoder(r.Body)
	req := &PostTestRunRequest{}

	err := decoder.Decode(req)
	if err != nil {
		ah.sendErrorResponse(w, r.URL.String(), fmt.Sprintf("error decoding request body: %v", err), http.StatusBadRequest)
		return
	}

	// create test run
	testInstance, err := ah.coordinator.ScheduleTest(testDescriptor, req.Config)
	if err != nil {
		ah.sendErrorResponse(w, r.URL.String(), fmt.Sprintf("failed creating test: %v", err), http.StatusInternalServerError)
		return
	}

	ah.sendOKResponse(w, r.URL.String(), &PostTestRunResponse{
		TestID: testDescriptor.ID(),
		RunID:  testInstance.RunID(),
		Name:   testInstance.Name(),
		Config: testInstance.GetTestVariables().GetVarsMap(),
	})
}
