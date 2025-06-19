package api

import (
	"net/http"

	"github.com/erigontech/assertoor/pkg/coordinator/types"
	"github.com/gorilla/mux"
)

type GetTestResponse struct {
	ID         string              `json:"id"`
	Source     string              `json:"source"`
	Name       string              `json:"name"`
	Timeout    uint64              `json:"timeout"`
	Config     map[string]any      `json:"config"`
	ConfigVars map[string]string   `json:"configVars"`
	Schedule   *types.TestSchedule `json:"schedule"`
}

// GetTest godoc
// @Id getTest
// @Summary Get test definition by test ID
// @Tags Test
// @Description Returns the test definition with given ID.
// @Produce json
// @Param testId path string true "ID of the test definition to get details for"
// @Success 200 {object} Response{data=GetTestResponse} "Success"
// @Failure 400 {object} Response "Failure"
// @Failure 500 {object} Response "Server Error"
// @Router /api/v1/test/{testId} [get]
func (ah *APIHandler) GetTest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", contentTypeJSON)

	vars := mux.Vars(r)
	if vars["testId"] == "" {
		ah.sendErrorResponse(w, r.URL.String(), "testId missing", http.StatusBadRequest)
		return
	}

	var testDescriptor types.TestDescriptor

	for _, testDescr := range ah.coordinator.TestRegistry().GetTestDescriptors() {
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

	testConfig := testDescriptor.Config()

	ah.sendOKResponse(w, r.URL.String(), &GetTestResponse{
		ID:         testDescriptor.ID(),
		Source:     testDescriptor.Source(),
		Name:       testConfig.Name,
		Timeout:    uint64(testConfig.Timeout.Seconds()),
		Config:     testConfig.Config,
		ConfigVars: testConfig.ConfigVars,
		Schedule:   testConfig.Schedule,
	})
}
