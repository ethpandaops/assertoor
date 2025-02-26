package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/noku-team/assertoor/pkg/coordinator/helper"
	"github.com/noku-team/assertoor/pkg/coordinator/types"
	"gopkg.in/yaml.v3"
)

type PostTestsRegisterRequest struct {
	ID           string                 `yaml:"id" json:"id"`
	Name         string                 `yaml:"name" json:"name"`
	Timeout      string                 `yaml:"timeout" json:"timeout"`
	Config       map[string]interface{} `yaml:"config" json:"config"`
	ConfigVars   map[string]string      `yaml:"configVars" json:"configVars"`
	Tasks        []helper.RawMessage    `yaml:"tasks" json:"tasks"`
	CleanupTasks []helper.RawMessage    `yaml:"cleanupTasks" json:"cleanupTasks"`
	Schedule     *types.TestSchedule    `yaml:"schedule" json:"schedule"`
}

type PostTestsRegisterResponse struct {
	TestID string         `json:"test_id"`
	Name   string         `json:"name"`
	Config map[string]any `json:"config"`
}

// PostTestsRegister godoc
// @Id postTestsRegister
// @Summary Register new test via yaml configuration
// @Tags Test
// @Description Returns the test id and name of the added test.
// @Produce json
// @Accept json,application/yaml
// @Param testConfig body PostTestsRegisterRequest true "Test configuration (json or yaml)"
// @Success 200 {object} Response{data=PostTestsRegisterResponse} "Success"
// @Failure 400 {object} Response "Failure"
// @Failure 500 {object} Response "Server Error"
// @Router /api/v1/tests/register [post]
func (ah *APIHandler) PostTestsRegister(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", contentTypeJSON)

	// parse request body
	req := &PostTestsRegisterRequest{}

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

	if req.ID == "" {
		ah.sendErrorResponse(w, r.URL.String(), "test id missing or empty", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		ah.sendErrorResponse(w, r.URL.String(), "test name missing or empty", http.StatusBadRequest)
		return
	}

	if len(req.Tasks) == 0 {
		ah.sendErrorResponse(w, r.URL.String(), "test must have 1 or more tasks", http.StatusBadRequest)
		return
	}

	testConfig := &types.TestConfig{
		ID:           req.ID,
		Name:         req.Name,
		Timeout:      helper.Duration{},
		Config:       req.Config,
		ConfigVars:   req.ConfigVars,
		Tasks:        req.Tasks,
		CleanupTasks: req.CleanupTasks,
		Schedule:     req.Schedule,
	}

	if req.Timeout != "" {
		err := testConfig.Timeout.Unmarshal(req.Timeout)
		if err != nil {
			ah.sendErrorResponse(w, r.URL.String(), fmt.Sprintf("failed decoding timeout: %v", err), http.StatusBadRequest)
			return
		}
	}

	// add test descriptor
	testDescriptor, err := ah.coordinator.TestRegistry().AddLocalTest(testConfig)
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
