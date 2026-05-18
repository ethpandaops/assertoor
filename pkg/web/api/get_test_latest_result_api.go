package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/gorilla/mux"
)

// latestResultScanLimit caps how many recent runs we walk while looking
// for one that has a $ASSERTOOR_TEST_RESULT blob set. Keeps the
// endpoint cheap even on busy boxes.
const latestResultScanLimit = 20

// GetTestLatestResult godoc
// @Id getTestLatestResult
// @Summary Get the latest run-level result markdown for a test
// @Tags Test
// @Description Walks recent runs of the test (newest first) and returns
// @Description the first $ASSERTOOR_TEST_RESULT markdown blob found.
// @Description Returns 200 with metadata in headers (X-Run-Id, X-Run-Status,
// @Description X-Run-Start-Time) and the markdown as the body.
// @Description When ?meta=1 is set, returns a JSON envelope instead
// @Description (handy for tile fetchers that want one round-trip).
// @Produce text/markdown
// @Produce json
// @Param testId path string true "Test ID"
// @Param meta query string false "Set to 1 to receive a JSON envelope"
// @Success 200 {string} string "Markdown body"
// @Success 204 "No result available across recent runs"
// @Failure 400 {object} Response "Bad Request"
// @Failure 404 {object} Response "Test not found"
// @Router /api/v1/test/{testId}/latest_result [get]
func (ah *APIHandler) GetTestLatestResult(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	testID := vars["testId"]

	if testID == "" {
		w.Header().Set("Content-Type", contentTypeJSON)
		ah.sendErrorResponse(w, r.URL.String(), "missing testId", http.StatusBadRequest)

		return
	}

	runs, _ := ah.coordinator.GetTestHistory(testID, 0, 0, latestResultScanLimit)
	wantJSON := r.URL.Query().Get("meta") == "1"

	for _, run := range runs {
		result, err := ah.coordinator.Database().GetTestResult(run.RunID())
		if err != nil || result == nil || len(result.Data) == 0 {
			continue
		}

		startTime := int64(0)
		if !run.StartTime().IsZero() {
			startTime = run.StartTime().Unix()
		}

		stopTime := int64(0)
		if !run.StopTime().IsZero() {
			stopTime = run.StopTime().Unix()
		}

		if wantJSON {
			ah.sendOKResponse(w, r.URL.String(), &GetTestLatestResultResponse{
				RunID:     run.RunID(),
				Status:    run.Status(),
				StartTime: startTime,
				StopTime:  stopTime,
				Markdown:  string(result.Data),
			})

			return
		}

		w.Header().Set("X-Run-Id", strconv.FormatUint(run.RunID(), 10))
		w.Header().Set("X-Run-Status", string(run.Status()))

		if startTime > 0 {
			w.Header().Set("X-Run-Start-Time", strconv.FormatInt(startTime, 10))
		}

		if stopTime > 0 {
			w.Header().Set("X-Run-Stop-Time", strconv.FormatInt(stopTime, 10))
		}

		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		http.ServeContent(w, r, "result.md", time.Now(), bytes.NewReader(result.Data))

		return
	}

	if wantJSON {
		// Send an empty envelope (status OK, data null) so the UI can
		// distinguish "no result yet" from a network failure.
		w.Header().Set("Content-Type", contentTypeJSON)
		w.WriteHeader(http.StatusOK)

		_ = json.NewEncoder(w).Encode(&Response{Status: "OK"})

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetTestLatestResultResponse is the JSON envelope returned when the
// client passes ?meta=1.
type GetTestLatestResultResponse struct {
	RunID     uint64           `json:"run_id"`
	Status    types.TestStatus `json:"status"`
	StartTime int64            `json:"start_time"`
	StopTime  int64            `json:"stop_time"`
	Markdown  string           `json:"markdown"`
}
