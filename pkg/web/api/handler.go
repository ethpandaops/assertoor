package api

import (
	"encoding/json"
	"net/http"

	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/ethpandaops/assertoor/pkg/web/auth"
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
	Status string `json:"status"`
	Data   any    `json:"data"`
}

//nolint:revive // ignore
type APIHandler struct {
	logger      logrus.FieldLogger
	coordinator types.Coordinator
	authHandler *auth.Handler
}

func NewAPIHandler(logger logrus.FieldLogger, coordinator types.Coordinator, authHandler *auth.Handler) *APIHandler {
	return &APIHandler{
		logger:      logger,
		coordinator: coordinator,
		authHandler: authHandler,
	}
}

// checkAuth verifies the Authorization header. In open mode (no auth
// provider configured) it always returns true.
func (ah *APIHandler) checkAuth(r *http.Request) bool {
	if ah.authHandler == nil || ah.authHandler.IsOpen() {
		return true
	}

	token := ah.authHandler.CheckAuthToken(r.Header.Get("Authorization"), auth.StripPort(r.Host))
	if token == nil || !token.Valid {
		return false
	}

	return true
}

// requireTestRegistry is checkAuth for the endpoints that change test
// definitions: it answers 401 without a valid login and 403 when the login
// lacks the test_registry feature, and returns false in both cases.
func (ah *APIHandler) requireTestRegistry(w http.ResponseWriter, r *http.Request) bool {
	if ah.authHandler == nil || ah.authHandler.IsOpen() {
		return true
	}

	token := ah.authHandler.CheckAuthToken(r.Header.Get("Authorization"), auth.StripPort(r.Host))
	if token == nil || !token.Valid {
		ah.sendUnauthorizedResponse(w, r.URL.String())
		return false
	}

	if !ah.authHandler.HasFeature(token, auth.FeatureTestRegistry) {
		ah.sendErrorResponse(w, r.URL.String(), "forbidden: this login lacks the assertoor feature "+auth.FeatureTestRegistry, http.StatusForbidden)
		return false
	}

	return true
}

func (ah *APIHandler) sendUnauthorizedResponse(w http.ResponseWriter, route string) {
	ah.sendErrorResponse(w, route, "unauthorized", http.StatusUnauthorized)
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

func (ah *APIHandler) sendOKResponse(w http.ResponseWriter, route string, data any) {
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
