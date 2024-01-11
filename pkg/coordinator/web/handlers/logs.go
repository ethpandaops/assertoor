package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
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
	vars := mux.Vars(r)

	sinceTime, err := strconv.ParseInt(vars["since"], 10, 64)
	if err != nil {
		fmt.Printf("err: %v", err)

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

	taskLog := fh.coordinator.LogScope().GetLogEntriesSince(since)
	pageData.Log = make([]*LogsEntry, len(taskLog))

	for i, log := range taskLog {
		logData := &LogsEntry{
			TIdx:    log.Time.UnixNano(),
			Time:    log.Time,
			Level:   uint64(log.Level),
			Message: log.Message,
			Data:    map[string]string{},
			DataLen: uint64(len(log.Data)),
		}

		for dataKey, dataVal := range log.Data {
			logData.Data[dataKey] = fmt.Sprintf("%v", dataVal)
		}

		pageData.Log[i] = logData
	}

	return pageData
}
