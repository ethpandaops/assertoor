package api

import (
	"net/http"
)

type GetTestsResponse struct {
	ID       string `json:"id"`
	Source   string `json:"source"`
	BasePath string `json:"basePath"`
	Name     string `json:"name"`
}

// GetTests godoc
// @Id getTests
// @Summary Get list of test definitions
// @Tags Test
// @Description Returns the list of test definitions. These test definitions can be used to create new test runs and are supplied via the assertoor configuration.
// @Produce  json
// @Success 200 {object} Response{data=[]GetTestsResponse} "Success"
// @Failure 400 {object} Response "Failure"
// @Failure 500 {object} Response "Server Error"
// @Router /api/v1/tests [get]
func (ah *APIHandler) GetTests(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", contentTypeJSON)

	tests := []*GetTestsResponse{}

	for _, testDescr := range ah.coordinator.TestRegistry().GetTestDescriptors() {
		if testDescr.Err() != nil {
			continue
		}

		tests = append(tests, &GetTestsResponse{
			ID:       testDescr.ID(),
			Source:   testDescr.Source(),
			BasePath: testDescr.BasePath(),
			Name:     testDescr.Config().Name,
		})
	}

	ah.sendOKResponse(w, r.URL.String(), tests)
}
