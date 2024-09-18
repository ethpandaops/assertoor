package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/helper"
	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
)

type PostTestsRegisterExternalRequest struct {
	File       string                 `yaml:"file" json:"file"`
	Name       string                 `yaml:"name" json:"name,omitempty"`
	Timeout    uint64                 `yaml:"timeout" json:"timeout,omitempty"`
	Config     map[string]interface{} `yaml:"config" json:"config,omitempty"`
	ConfigVars map[string]string      `yaml:"configVars" json:"configVars,omitempty"`
	Schedule   *types.TestSchedule    `yaml:"schedule" json:"schedule,omitempty"`
}

type PostTestsRegisterExternalResponse struct {
	TestID string         `json:"test_id"`
	Name   string         `json:"name"`
	Config map[string]any `json:"config"`
}

// PostTestsRegisterExternal godoc
// @Id postTestsRegisterExternal
// @Summary Register new test via external test configuration
// @Tags Test
// @Description Returns the test id and name of the added test.
// @Produce json
// @Accept json
// @Param externalTestConfig body PostTestsRegisterExternalRequest true "Test configuration (json or yaml)"
// @Success 200 {object} Response{data=PostTestsRegisterExternalResponse} "Success"
// @Failure 400 {object} Response "Failure"
// @Failure 500 {object} Response "Server Error"
// @Router /api/v1/tests/register_external [post]
func (ah *APIHandler) PostTestsRegisterExternal(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// parse request body
	req := &PostTestsRegisterExternalRequest{}
	decoder := json.NewDecoder(r.Body)

	err := decoder.Decode(req)
	if err != nil {
		ah.sendErrorResponse(w, r.URL.String(), fmt.Sprintf("error decoding request body json: %v", err), http.StatusBadRequest)
		return
	}

	extTestCfg := &types.ExternalTestConfig{
		File:       req.File,
		Name:       req.Name,
		Timeout:    &helper.Duration{},
		Config:     req.Config,
		ConfigVars: req.ConfigVars,
		Schedule:   req.Schedule,
	}
	if req.Timeout > 0 {
		extTestCfg.Timeout = &helper.Duration{Duration: time.Duration(req.Timeout) * time.Second}
	}

	// add test descriptor
	testDescriptor, err := ah.coordinator.AddExternalTest(r.Context(), extTestCfg)
	if err != nil {
		ah.sendErrorResponse(w, r.URL.String(), fmt.Sprintf("failed adding test: %v", err), http.StatusInternalServerError)
		return
	}

	ah.sendOKResponse(w, r.URL.String(), &PostTestsRegisterResponse{
		TestID: testDescriptor.ID(),
		Name:   testDescriptor.Config().Name,
		Config: testDescriptor.Config().Config,
	})
}
