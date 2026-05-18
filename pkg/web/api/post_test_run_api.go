package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ethpandaops/assertoor/pkg/types"
	"gopkg.in/yaml.v3"
)

type PostTestRunsScheduleRequest struct {
	TestID         string         `json:"test_id"`
	Config         map[string]any `json:"config"`
	AllowDuplicate bool           `json:"allow_duplicate"`

	// SkipQueue is the legacy boolean that drove "off-queue parallel
	// execution". Kept for backward compatibility — when Queue is
	// supplied it wins; otherwise we fall through to this field.
	//
	// Deprecated: use Queue with mode="immediate".
	SkipQueue bool `json:"skip_queue,omitempty"`

	// Queue is the modern (v2) way of expressing where the new test
	// should slot relative to the runner's pending queue. When unset
	// the request falls back to SkipQueue. See ScheduleQueueOption.
	Queue *ScheduleQueueOption `json:"queue,omitempty"`
}

// ScheduleQueueMode is the discriminator for ScheduleQueueOption.
type ScheduleQueueMode string

const (
	// ScheduleQueueModeImmediate runs the test off-queue (parallel,
	// no waiting). Equivalent to the legacy skip_queue=true.
	ScheduleQueueModeImmediate ScheduleQueueMode = "immediate"

	// ScheduleQueueModeEnd appends to the pending queue. Equivalent
	// to the legacy skip_queue=false default.
	ScheduleQueueModeEnd ScheduleQueueMode = "end"

	// ScheduleQueueModeAfter inserts just behind a specific run
	// already in the queue (AfterRunID).
	ScheduleQueueModeAfter ScheduleQueueMode = "after"
)

type ScheduleQueueOption struct {
	Mode ScheduleQueueMode `json:"mode"`
	// AfterRunID is honoured only when Mode == "after". The new test
	// is placed immediately after that run in the pending queue. If
	// the referenced run is no longer queued the request silently
	// falls back to "append to end".
	AfterRunID uint64 `json:"after_run_id,omitempty"`
}

type PostTestRunsScheduleResponse struct {
	TestID string         `json:"test_id"`
	RunID  uint64         `json:"run_id"`
	Name   string         `json:"name"`
	Config map[string]any `json:"config"`
}

// PostTestRunsSchedule godoc
// @Id postTestRunsSchedule
// @Summary Schedule new test run by test ID
// @Tags TestRun
// @Description Returns the test & run id of the scheduled test execution.
// @Produce json
// @Param runOptions body PostTestRunsScheduleRequest true "Rest run options"
// @Success 200 {object} Response{data=PostTestRunsScheduleResponse} "Success"
// @Failure 400 {object} Response "Failure"
// @Failure 500 {object} Response "Server Error"
// @Router /api/v1/test_runs/schedule [post]
func (ah *APIHandler) PostTestRunsSchedule(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", contentTypeJSON)

	if !ah.checkAuth(r) {
		ah.sendUnauthorizedResponse(w, r.URL.String())
		return
	}

	// parse request body
	req := &PostTestRunsScheduleRequest{}

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

	// get test descriptor by test id
	var testDescriptor types.TestDescriptor

	for _, testDescr := range ah.coordinator.TestRegistry().GetTestDescriptors() {
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

	// Build the scheduling options. The modern `Queue` field wins; if
	// absent we honour the legacy `skip_queue` boolean so older
	// integrations keep working.
	opts := types.ScheduleOptions{
		AllowDuplicate: req.AllowDuplicate,
	}

	if req.Queue != nil {
		switch req.Queue.Mode {
		case ScheduleQueueModeImmediate:
			opts.SkipQueue = true
		case ScheduleQueueModeEnd:
			opts.SkipQueue = false
		case ScheduleQueueModeAfter:
			opts.SkipQueue = false
			opts.AfterRunID = req.Queue.AfterRunID
		default:
			ah.sendErrorResponse(w, r.URL.String(),
				fmt.Sprintf("unknown queue mode %q (expected immediate|end|after)", req.Queue.Mode),
				http.StatusBadRequest)

			return
		}
	} else {
		opts.SkipQueue = req.SkipQueue
	}

	// create test run
	testInstance, err := ah.coordinator.ScheduleTestWithOptions(testDescriptor, req.Config, opts)
	if err != nil {
		ah.sendErrorResponse(w, r.URL.String(), fmt.Sprintf("failed creating test: %v", err), http.StatusInternalServerError)
		return
	}

	ah.sendOKResponse(w, r.URL.String(), &PostTestRunsScheduleResponse{
		TestID: testDescriptor.ID(),
		RunID:  testInstance.RunID(),
		Name:   testInstance.Name(),
		Config: testInstance.GetTestVariables().GetVarsMap(nil, false),
	})
}
