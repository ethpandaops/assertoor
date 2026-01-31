package api

import (
	"encoding/json"
	"net/http"

	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/sirupsen/logrus"
)

// @title Assertoor API Documentation
// @version 1.0
// @description API for querying information about Assertoor tests
// @tag.name Test
// @tag.description All endpoints related to test definitions
// @tag.name TestRun
// @tag.description All endpoints related to test runs

const contentTypeYAML = "application/yaml"

const contentTypeJSON = "application/json"

type Response struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data"`
}

//nolint:revive // ignore
type APIHandler struct {
	logger      logrus.FieldLogger
	coordinator types.Coordinator
}

func NewAPIHandler(logger logrus.FieldLogger, coordinator types.Coordinator) *APIHandler {
	return &APIHandler{
		logger:      logger,
		coordinator: coordinator,
	}
}

func (ah *APIHandler) sendErrorResponse(w http.ResponseWriter, route, message string, errorcode int) {
	w.WriteHeader(errorcode)
	j := json.NewEncoder(w)
	response := &Response{}
	response.Status = "ERROR: " + message

	err := j.Encode(response)
	if err != nil {
		ah.logger.Errorf("error serializing json error for API %v route: %v", route, err)
	}
}

func (ah *APIHandler) sendOKResponse(w http.ResponseWriter, route string, data interface{}) {
	j := json.NewEncoder(w)
	response := &Response{
		Status: "OK",
		Data:   data,
	}

	err := j.Encode(response)
	if err != nil {
		ah.logger.Errorf("error serializing json data for API %v route: %v", route, err)
	}
}
