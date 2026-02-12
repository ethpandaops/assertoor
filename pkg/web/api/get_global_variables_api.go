package api

import (
	"net/http"
)

// GetGlobalVariablesResponse contains the names of all configured global variables.
type GetGlobalVariablesResponse struct {
	Names []string `json:"names"`
}

// GetGlobalVariables godoc
// @Id getGlobalVariables
// @Summary Get global variable names
// @Tags Config
// @Description Returns the names of all configured global variables. Values are not included as they may contain sensitive data.
// @Produce json
// @Success 200 {object} Response{data=GetGlobalVariablesResponse} "Success"
// @Failure 401 {object} Response "Unauthorized"
// @Router /api/v1/global_variables [get]
func (ah *APIHandler) GetGlobalVariables(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", contentTypeJSON)

	// Require authentication - variable names can hint at infrastructure details
	if !ah.checkAuth(r) {
		ah.sendUnauthorizedResponse(w, r.URL.String())
		return
	}

	globalVars := ah.coordinator.GlobalVariables()
	varsMap := globalVars.GetVarsMap(nil, true)

	names := make([]string, 0, len(varsMap))
	for name := range varsMap {
		names = append(names, name)
	}

	ah.sendOKResponse(w, r.URL.String(), &GetGlobalVariablesResponse{
		Names: names,
	})
}
