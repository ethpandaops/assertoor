package api

import (
	"net/http"

	"github.com/ethpandaops/assertoor/pkg/types"
)

type GetTestRunsResponse struct {
	RunID     uint64           `json:"run_id"`
	TestID    string           `json:"test_id"`
	Name      string           `json:"name"`
	Status    types.TestStatus `json:"status"`
	StartTime int64            `json:"start_time"`
	StopTime  int64            `json:"stop_time"`
}

// GetTestRuns godoc
// @Id getTestRuns
// @Summary Get list of test runs
// @Tags TestRun
// @Description Returns a list of all test runs.
// @Produce  json
// @Param  test_id query string false "Return test runs for this test ID only"
// @Success 200 {object} Response{data=[]GetTestRunsResponse} "Success"
// @Failure 400 {object} Response "Failure"
// @Failure 500 {object} Response "Server Error"
// @Router /api/v1/test_runs [get]
func (ah *APIHandler) GetTestRuns(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", contentTypeJSON)

	q := r.URL.Query()
	filterTestID := q.Get("test_id")
	testRuns := []*GetTestRunsResponse{}

	testInstances, _ := ah.coordinator.GetTestHistory(filterTestID, 0, 0, 100)

	for _, testInstance := range testInstances {
		testRun := &GetTestRunsResponse{
			RunID:  testInstance.RunID(),
			TestID: testInstance.TestID(),
			Name:   testInstance.Name(),
			Status: testInstance.Status(),
		}

		if !testInstance.StartTime().IsZero() {
			testRun.StartTime = testInstance.StartTime().Unix()
		}

		if !testInstance.StopTime().IsZero() {
			testRun.StopTime = testInstance.StopTime().Unix()
		}

		testRuns = append(testRuns, testRun)
	}

	ah.sendOKResponse(w, r.URL.String(), testRuns)
}
