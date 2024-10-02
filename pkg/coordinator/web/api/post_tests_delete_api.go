package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"gopkg.in/yaml.v3"
)

type PostTestsDeleteRequest struct {
	Tests []string `json:"tests"`
}

type PostTestsDeleteResponse struct {
	Deleted []string `json:"deleted"`
	Errors  []string `json:"errors"`
}

// PostTestsDelete godoc
// @Id postTestsDelete
// @Summary Delete tests
// @Tags Test
// @Description Deletes tests
// @Produce json
// @Accept json,application/yaml
// @Param testConfig body PostTestsDeleteRequest true "Test configuration (json or yaml)"
// @Success 200 {object} Response{data=PostTestsDeleteResponse} "Success"
// @Failure 400 {object} Response "Failure"
// @Failure 500 {object} Response "Server Error"
// @Router /api/v1/tests/delete [post]
func (ah *APIHandler) PostTestsDelete(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", contentTypeJSON)

	// parse request body
	req := &PostTestsDeleteRequest{}

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

	deletedTests := make([]string, 0)
	errors := make([]string, 0)

	for _, testID := range req.Tests {
		err := ah.coordinator.TestRegistry().DeleteTest(testID)
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed deleting test %v: %v", testID, err))
		} else {
			deletedTests = append(deletedTests, testID)
		}
	}

	ah.sendOKResponse(w, r.URL.String(), &PostTestsDeleteResponse{
		Deleted: deletedTests,
		Errors:  errors,
	})
}
