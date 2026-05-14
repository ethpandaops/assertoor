package api

import (
	"net/http"
)

// GetPlaybookLibrary godoc
// @Id getPlaybookLibrary
// @Summary Get the published playbook library index
// @Tags PlaybookLibrary
// @Description Returns the index of playbooks available from the configured upstream
// @Description (default: ethpandaops/assertoor master). The response includes the
// @Description resolved base URL used to fetch individual playbooks and the folder
// @Description metadata harvested from `_header.yaml` files.
// @Description The index is cached server-side; a stale copy is served if a refresh fails.
// @Produce json
// @Success 200 {object} Response{data=playbooklibrary.IndexResponse} "Success"
// @Failure 404 {object} Response "Library disabled"
// @Failure 500 {object} Response "Server Error"
// @Router /api/v1/playbook_library [get]
func (ah *APIHandler) GetPlaybookLibrary(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", contentTypeJSON)

	svc := ah.coordinator.PlaybookLibrary()
	if svc == nil || !svc.Enabled() {
		ah.sendErrorResponse(w, r.URL.String(), "playbook library is disabled", http.StatusNotFound)
		return
	}

	resp, err := svc.GetIndex(r.Context())
	if err != nil {
		ah.sendErrorResponse(w, r.URL.String(), err.Error(), http.StatusInternalServerError)
		return
	}

	ah.sendOKResponse(w, r.URL.String(), resp)
}

// GetPlaybookLibraryCheckResponse is returned by the check endpoint.
type GetPlaybookLibraryCheckResponse struct {
	State       string `json:"state"`
	RemoteID    string `json:"remote_id"`
	RemoteName  string `json:"remote_name"`
	RemoteURL   string `json:"remote_url"`
	RemoteYaml  string `json:"remote_yaml"`
	LocalTestID string `json:"local_test_id,omitempty"`
	LocalName   string `json:"local_name,omitempty"`
	LocalYaml   string `json:"local_yaml,omitempty"`
}

// GetPlaybookLibraryCheck godoc
// @Id getPlaybookLibraryCheck
// @Summary Compare a library playbook against the locally-registered copy
// @Tags PlaybookLibrary
// @Description Fetches the remote YAML for the given library file and compares it
// @Description against any locally registered test with the same id. The response
// @Description tells the caller whether to register fresh (absent), skip registration
// @Description and run the existing test (same), or warn before overwriting (different).
// @Description Requires authentication because the response includes local YAML which
// @Description may contain sensitive configuration.
// @Produce json
// @Param file query string true "Library file path (e.g. stable/stability-check.yaml)"
// @Success 200 {object} Response{data=GetPlaybookLibraryCheckResponse} "Success"
// @Failure 400 {object} Response "Bad Request"
// @Failure 401 {object} Response "Unauthorized"
// @Failure 404 {object} Response "Library disabled or file not in index"
// @Failure 500 {object} Response "Server Error"
// @Router /api/v1/playbook_library/check [get]
func (ah *APIHandler) GetPlaybookLibraryCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", contentTypeJSON)

	if !ah.checkAuth(r) {
		ah.sendUnauthorizedResponse(w, r.URL.String())
		return
	}

	svc := ah.coordinator.PlaybookLibrary()
	if svc == nil || !svc.Enabled() {
		ah.sendErrorResponse(w, r.URL.String(), "playbook library is disabled", http.StatusNotFound)
		return
	}

	file := r.URL.Query().Get("file")
	if file == "" {
		ah.sendErrorResponse(w, r.URL.String(), "file query parameter is required", http.StatusBadRequest)
		return
	}

	result, remoteYaml, err := svc.Check(r.Context(), file)
	if err != nil {
		ah.sendErrorResponse(w, r.URL.String(), err.Error(), http.StatusInternalServerError)
		return
	}

	ah.sendOKResponse(w, r.URL.String(), &GetPlaybookLibraryCheckResponse{
		State:       string(result.State),
		RemoteID:    result.RemoteID,
		RemoteName:  result.RemoteName,
		RemoteURL:   result.RemoteURL,
		RemoteYaml:  remoteYaml,
		LocalTestID: result.LocalTestID,
		LocalName:   result.LocalName,
		LocalYaml:   result.LocalSource,
	})
}
