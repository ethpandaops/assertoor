package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
)

type PostTestRunRequest struct {
	TestID         string         `json:"test_id"`
	Config         map[string]any `json:"config"`
	AllowDuplicate bool           `json:"allow_duplicate"`
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
// @Tags TestRun
// @Description Returns the test & run id of the scheduled test execution.
// @Produce json
// @Param runOptions body PostTestRunRequest true "Rest run options"
// @Success 200 {object} Response{data=PostTestRunResponse} "Success"
// @Failure 400 {object} Response "Failure"
// @Failure 500 {object} Response "Server Error"
// @Router /api/v1/test_run [post]
func (ah *APIHandler) PostTestRun(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// parse request body
	decoder := json.NewDecoder(r.Body)
	req := &PostTestRunRequest{}

	err := decoder.Decode(req)
	if err != nil {
		ah.sendErrorResponse(w, r.URL.String(), fmt.Sprintf("error decoding request body: %v", err), http.StatusBadRequest)
		return
	}

	// get test descriptor by test id
	var testDescriptor types.TestDescriptor

	for _, testDescr := range ah.coordinator.GetTestDescriptors() {
		if testDescr.Err() != nil {
			continue
		}

		if testDescr.ID() == req.TestID {
			testDescriptor = testDescr
			break
		}
	}

	if testDescriptor == nil {
		ah.sendErrorResponse(w, r.URL.String(), "test not found", http.StatusNotFound)
		return
	}

	// create test run
	testInstance, err := ah.coordinator.ScheduleTest(testDescriptor, req.Config, req.AllowDuplicate)
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
