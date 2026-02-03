package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"gopkg.in/yaml.v3"
)

type GetLogsResponse struct {
	Log []*LogEntry `json:"log"`
}

type LogEntry struct {
	LogIndex uint64            `json:"tidx"`
	Time     time.Time         `json:"time"`
	Level    uint64            `json:"level"`
	Message  string            `json:"msg"`
	DataLen  uint64            `json:"datalen"`
	Data     map[string]string `json:"data"`
}

// GetLogs godoc
// @Summary Get application logs
// @Description Returns application-level logs from the coordinator. Protected endpoint - requires authentication unless auth is disabled.
// @Tags Logs
// @Produce json
// @Param since path string true "Log index to start from (0 for all)"
// @Success 200 {object} Response{data=GetLogsResponse} "Success"
// @Failure 401 {object} Response "Unauthorized"
// @Failure 500 {object} Response "Server Error"
// @Router /api/v1/logs/{since} [get]
func (ah *APIHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Check authentication - logs are protected
	if !ah.checkAuth(r) {
		ah.sendUnauthorizedResponse(w, r.URL.Path)
		return
	}

	vars := mux.Vars(r)

	sinceTime, err := strconv.ParseUint(vars["since"], 10, 64)
	if err != nil {
		sinceTime = 0
	}

	response := ah.getLogsData(sinceTime)
	ah.sendOKResponse(w, r.URL.Path, response)
}

func (ah *APIHandler) getLogsData(since uint64) *GetLogsResponse {
	response := &GetLogsResponse{}

	logEntries := ah.coordinator.LogReader().GetLogEntries(since, 0)
	response.Log = make([]*LogEntry, len(logEntries))

	for i, log := range logEntries {
		logData := &LogEntry{
			LogIndex: log.LogIndex,
			Time:     time.Unix(0, log.LogTime*int64(time.Millisecond)),
			Level:    uint64(log.LogLevel),
			Message:  log.LogMessage,
			Data:     map[string]string{},
		}

		if log.LogFields != "" {
			err := yaml.Unmarshal([]byte(log.LogFields), &logData.Data)
			if err == nil {
				logData.DataLen = uint64(len(logData.Data))
			}
		}

		response.Log[i] = logData
	}

	return response
}
