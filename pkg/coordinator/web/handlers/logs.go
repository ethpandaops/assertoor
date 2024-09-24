package handlers

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type Logs struct {
	Log []*LogsEntry `json:"log"`
}

type LogsEntry struct {
	TIdx    int64             `json:"tidx"`
	Time    time.Time         `json:"time"`
	Level   uint64            `json:"level"`
	Message string            `json:"msg"`
	DataLen uint64            `json:"datalen"`
	Data    map[string]string `json:"data"`
}

func (fh *FrontendHandler) LogsData(w http.ResponseWriter, r *http.Request) {
	if fh.securityTrimmed {
		http.Error(w, "Not allowed", http.StatusForbidden)
		return
	}

	vars := mux.Vars(r)

	sinceTime, err := strconv.ParseInt(vars["since"], 10, 64)
	if err != nil {
		fmt.Printf("err: %v", err)

		sinceTime = 0
	}

	// Check bounds before converting to int
	if sinceTime < math.MinInt32 || sinceTime > math.MaxInt32 {
		fmt.Printf("timestamp out of int bounds: %v", sinceTime)
		sinceTime = 0
	}

	pageData := fh.getLogsPageData(sinceTime)

	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(pageData)
	if err != nil {
		logrus.WithError(err).Error("error encoding test data")

		//nolint:gocritic // ignore
		http.Error(w, "Internal server error", http.StatusServiceUnavailable)
	}
}

func (fh *FrontendHandler) getLogsPageData(since int64) *Logs {
	pageData := &Logs{}

	taskLog := fh.coordinator.LogReader().GetLogEntries(int(since), 0)
	pageData.Log = make([]*LogsEntry, len(taskLog))

	for i, log := range taskLog {
		logData := &LogsEntry{
			TIdx:    int64(log.LogIndex),
			Time:    time.Unix(0, log.LogTime*int64(time.Millisecond)),
			Level:   uint64(log.LogLevel),
			Message: log.LogMessage,
			Data:    map[string]string{},
		}

		if log.LogFields != "" {
			err := yaml.Unmarshal([]byte(log.LogFields), &logData.Data)
			if err == nil {
				logData.DataLen = uint64(len(logData.Data))
			}
		}

		pageData.Log[i] = logData
	}

	return pageData
}
