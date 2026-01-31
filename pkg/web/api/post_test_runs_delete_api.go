package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"gopkg.in/yaml.v3"
)

type PostTestRunsDeleteRequest struct {
	TestRuns []uint64 `json:"test_runs"`
}

type PostTestRunsDeleteResponse struct {
	Deleted []uint64 `json:"deleted"`
	Errors  []string `json:"errors"`
}

// PostTestRunsDelete godoc
// @Id postTestRunsDelete
// @Summary Delete test runs
// @Tags TestRun
// @Description Deletes test runs
// @Produce json
// @Accept json,application/yaml
// @Param testConfig body PostTestRunsDeleteRequest true "Test configuration (json or yaml)"
// @Success 200 {object} Response{data=PostTestRunsDeleteResponse} "Success"
// @Failure 400 {object} Response "Failure"
// @Failure 500 {object} Response "Server Error"
// @Router /api/v1/test_runs/delete [post]
func (ah *APIHandler) PostTestRunsDelete(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", contentTypeJSON)

	// parse request body
	req := &PostTestRunsDeleteRequest{}

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

	deletedTests := make([]uint64, 0)
	errors := make([]string, 0)

	for _, runID := range req.TestRuns {
		err := ah.coordinator.DeleteTestRun(runID)
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed deleting test %v: %v", runID, err))
		} else {
			deletedTests = append(deletedTests, runID)
		}
	}

	ah.sendOKResponse(w, r.URL.String(), &PostTestRunsDeleteResponse{
		Deleted: deletedTests,
		Errors:  errors,
	})
}
