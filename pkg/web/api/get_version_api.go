package api

import (
	"net/http"
	"strings"

	"github.com/ethpandaops/assertoor/pkg/buildinfo"
)

// GetVersionResponse contains build version information for the running Assertoor binary.
type GetVersionResponse struct {
	Version string `json:"version"`
	Release string `json:"release"`
}

// GetVersion godoc
// @Id getVersion
// @Summary Get build version information
// @Tags Config
// @Description Returns the build version (commit hash) and release of the running Assertoor binary.
// @Produce json
// @Success 200 {object} Response{data=GetVersionResponse} "Success"
// @Router /api/v1/version [get]
func (ah *APIHandler) GetVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", contentTypeJSON)

	// ldflags inject values wrapped in literal quotes (see Makefile); strip them for a clean display.
	ah.sendOKResponse(w, r.URL.String(), &GetVersionResponse{
		Version: strings.Trim(buildinfo.BuildVersion, `"`),
		Release: strings.Trim(buildinfo.BuildRelease, `"`),
	})
}
