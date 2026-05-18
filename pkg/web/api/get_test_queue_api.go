package api

import (
	"net/http"

	"github.com/ethpandaops/assertoor/pkg/types"
)

type GetTestQueueResponse struct {
	Queue []GetTestQueueEntry `json:"queue"`
}

type GetTestQueueEntry struct {
	RunID  uint64           `json:"run_id"`
	TestID string           `json:"test_id"`
	Name   string           `json:"name"`
	Status types.TestStatus `json:"status"`
}

// GetTestQueue godoc
// @Id getTestQueue
// @Summary Get the current test runner queue
// @Tags TestRun
// @Description Returns the pending test queue in execution order
// @Description plus any currently running tests at the head. Used
// @Description by the StartTestModal's QueuePicker to let users
// @Description slot a new run at a chosen position.
// @Produce json
// @Success 200 {object} Response{data=GetTestQueueResponse} "Success"
// @Router /api/v1/test_queue [get]
func (ah *APIHandler) GetTestQueue(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", contentTypeJSON)

	resp := &GetTestQueueResponse{Queue: []GetTestQueueEntry{}}

	// Walk recent history to find running tests; the runner's queue
	// itself only contains pending entries (running tests have already
	// been popped).
	if running, _ := ah.coordinator.GetTestHistory("", 0, 0, 50); len(running) > 0 {
		for _, t := range running {
			if t.Status() == types.TestStatusRunning {
				resp.Queue = append(resp.Queue, GetTestQueueEntry{
					RunID:  t.RunID(),
					TestID: t.TestID(),
					Name:   t.Name(),
					Status: t.Status(),
				})
			}
		}
	}

	for _, t := range ah.coordinator.GetTestQueue() {
		resp.Queue = append(resp.Queue, GetTestQueueEntry{
			RunID:  t.RunID(),
			TestID: t.TestID(),
			Name:   t.Name(),
			Status: t.Status(),
		})
	}

	ah.sendOKResponse(w, r.URL.String(), resp)
}
