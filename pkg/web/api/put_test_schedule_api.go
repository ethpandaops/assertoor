package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/gorhill/cronexpr"
	"github.com/gorilla/mux"
)

// PutTestScheduleRequest is the body accepted by
// PUT /api/v1/test/{testId}/schedule. Setting `schedule` to null
// (i.e. omitting the field) clears the test's schedule entirely.
type PutTestScheduleRequest struct {
	// Schedule is the new schedule. May be null to clear.
	Schedule *types.TestSchedule `json:"schedule"`
}

// PutTestSchedule godoc
// @Id putTestSchedule
// @Summary Update a test's run schedule
// @Tags Test
// @Description Auth-required. Replaces the schedule (cron + startup
// @Description + skipQueue) for a registered test. Cron expressions
// @Description are validated up-front; invalid input rejects the
// @Description whole request without changing state.
// @Accept json
// @Produce json
// @Param testId path string true "Test ID"
// @Param body body PutTestScheduleRequest true "New schedule"
// @Success 200 {object} Response "Saved"
// @Failure 400 {object} Response "Bad Request"
// @Failure 401 {object} Response "Unauthorized"
// @Failure 404 {object} Response "Test not found"
// @Failure 500 {object} Response "Server Error"
// @Router /api/v1/test/{testId}/schedule [put]
func (ah *APIHandler) PutTestSchedule(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", contentTypeJSON)

	if !ah.checkAuth(r) {
		ah.sendUnauthorizedResponse(w, r.URL.String())
		return
	}

	vars := mux.Vars(r)
	testID := vars["testId"]

	if testID == "" {
		ah.sendErrorResponse(w, r.URL.String(), "missing testId", http.StatusBadRequest)
		return
	}

	// Confirm the test exists.
	var found bool

	for _, td := range ah.coordinator.TestRegistry().GetTestDescriptors() {
		if td.ID() == testID {
			found = true
			break
		}
	}

	if !found {
		ah.sendErrorResponse(w, r.URL.String(), "test not registered", http.StatusNotFound)
		return
	}

	req := &PutTestScheduleRequest{}
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		ah.sendErrorResponse(w, r.URL.String(), fmt.Sprintf("invalid JSON body: %v", err), http.StatusBadRequest)
		return
	}

	if err := ah.coordinator.TestRegistry().UpdateTestSchedule(testID, req.Schedule); err != nil {
		ah.sendErrorResponse(w, r.URL.String(), err.Error(), http.StatusBadRequest)
		return
	}

	ah.sendOKResponse(w, r.URL.String(), nil)
}

// GetTestNextRunResponse describes the upcoming firings of a test's
// cron schedule. `entries[].next` is a Unix timestamp; `expression`
// is the originating cron expression. Empty list means the test has
// no cron schedule.
type GetTestNextRunResponse struct {
	TestID  string                       `json:"test_id"`
	Entries []GetTestNextRunEntry        `json:"entries"`
	Earliest *GetTestNextRunEntryEarliest `json:"earliest,omitempty"`
}

type GetTestNextRunEntry struct {
	Expression string `json:"expression"`
	Next       int64  `json:"next"`
}

type GetTestNextRunEntryEarliest struct {
	Expression string `json:"expression"`
	Next       int64  `json:"next"`
}

// GetTestNextRun godoc
// @Id getTestNextRun
// @Summary Get the next planned cron firings for a test
// @Tags Test
// @Description Walks each cron expression on the test's schedule
// @Description and returns the next firing time per expression plus
// @Description the overall earliest one. Empty when the test has no
// @Description cron schedule.
// @Produce json
// @Param testId path string true "Test ID"
// @Success 200 {object} Response{data=GetTestNextRunResponse} "Success"
// @Failure 404 {object} Response "Test not found"
// @Router /api/v1/test/{testId}/next_run [get]
func (ah *APIHandler) GetTestNextRun(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", contentTypeJSON)

	vars := mux.Vars(r)
	testID := vars["testId"]

	var descriptor types.TestDescriptor

	for _, td := range ah.coordinator.TestRegistry().GetTestDescriptors() {
		if td.ID() == testID {
			descriptor = td
			break
		}
	}

	if descriptor == nil {
		ah.sendErrorResponse(w, r.URL.String(), "test not registered", http.StatusNotFound)
		return
	}

	resp := &GetTestNextRunResponse{TestID: testID, Entries: []GetTestNextRunEntry{}}

	cfg := descriptor.Config()
	if cfg.Schedule == nil || len(cfg.Schedule.Cron) == 0 {
		ah.sendOKResponse(w, r.URL.String(), resp)
		return
	}

	now := time.Now()

	var earliest *GetTestNextRunEntryEarliest

	for _, expr := range cfg.Schedule.Cron {
		parsed, err := cronexpr.Parse(expr)
		if err != nil {
			// Skip — clients can validate via PUT before saving.
			continue
		}

		next := parsed.Next(now)
		entry := GetTestNextRunEntry{Expression: expr, Next: next.Unix()}
		resp.Entries = append(resp.Entries, entry)

		if earliest == nil || next.Unix() < earliest.Next {
			earliest = &GetTestNextRunEntryEarliest{Expression: expr, Next: next.Unix()}
		}
	}

	resp.Earliest = earliest

	ah.sendOKResponse(w, r.URL.String(), resp)
}
