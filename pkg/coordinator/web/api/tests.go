package api

import (
	"net/http"

	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/gorilla/mux"
)

type GetTestsResponse struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Name   string `json:"name"`
}

// GetTests godoc
// @Summary Get list of test definitions
// @Tags Test
// @Description Returns the list of test definitions. These test definitions can be used to create new test runs and are supplied via the assertoor configuration.
// @Produce  json
// @Success 200 {object} Response{data=[]GetTestsResponse} "Success"
// @Failure 400 {object} Response "Failure"
// @Failure 500 {object} Response "Server Error"
// @Router /api/v1/tests [get]
func (ah *APIHandler) GetTests(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	tests := []*GetTestsResponse{}

	for _, testDescr := range ah.coordinator.GetTestDescriptors() {
		if testDescr.Err() != nil {
			continue
		}

		tests = append(tests, &GetTestsResponse{
			ID:     testDescr.ID(),
			Source: testDescr.Source(),
			Name:   testDescr.Config().Name,
		})
	}

	ah.sendOKResponse(w, r.URL.String(), tests)
}

type GetTestResponse struct {
	ID     string            `json:"id"`
	Source string            `json:"source"`
	Config *types.TestConfig `json:"config"`
}

// GetTests godoc
// @Summary Get test definition by test ID
// @Tags Test
// @Description Returns the test definition with given ID.
// @Produce  json
// @Param  testId path string true "ID of the test definition to get details for"
// @Success 200 {object} Response{data=GetTestResponse} "Success"
// @Failure 400 {object} Response "Failure"
// @Failure 500 {object} Response "Server Error"
// @Router /api/v1/test/{testId} [get]
func (ah *APIHandler) GetTest(w http.ResponseWriter, r *http.Request) {
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

	ah.sendOKResponse(w, r.URL.String(), &GetTestResponse{
		ID:     testDescriptor.ID(),
		Source: testDescriptor.Source(),
		Config: testDescriptor.Config(),
	})
}
