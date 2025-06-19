package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/erigontech/assertoor/pkg/coordinator/types"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v3"
)

// GetTestRunTaskDetails godoc
// @Id getTestRunTaskDetails
// @Summary Get detailed task of a given test run
// @Tags TestRun
// @Description Returns the task details with given run ID and task index. Includes full log, configuration and result variables (unless security trimmed).
// @Produce json
// @Param runId path string true "ID of the test run"
// @Param taskIndex path string true "Index of the task to get details for"
// @Success 200 {object} Response{data=GetTestRunDetailedTask} "Success"
// @Failure 400 {object} Response "Bad Request"
// @Failure 404 {object} Response "Not Found"
// @Failure 500 {object} Response "Server Error"
// @Router /api/v1/test_run/{runId}/task/{taskIndex}/details [get]
func (ah *APIHandler) GetTestRunTaskDetails(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", contentTypeJSON)

	vars := mux.Vars(r)

	runID, err := strconv.ParseUint(vars["runId"], 10, 64)
	if err != nil {
		ah.sendErrorResponse(w, r.URL.String(), "invalid runId provided", http.StatusBadRequest)
		return
	}

	taskIdx, err := strconv.ParseUint(vars["taskIndex"], 10, 64)
	if err != nil {
		ah.sendErrorResponse(w, r.URL.String(), "invalid taskIndex provided", http.StatusBadRequest)
		return
	}

	testInstance := ah.coordinator.GetTestByRunID(runID)
	if testInstance == nil {
		ah.sendErrorResponse(w, r.URL.String(), "test run not found", http.StatusNotFound)
		return
	}

	taskScheduler := testInstance.GetTaskScheduler()
	if taskScheduler == nil {
		ah.sendErrorResponse(w, r.URL.String(), "task scheduler not found", http.StatusNotFound)
		return
	}

	// Get task state by index
	taskState := taskScheduler.GetTaskState(types.TaskIndex(taskIdx))
	if taskState == nil {
		ah.sendErrorResponse(w, r.URL.String(), "task not found", http.StatusNotFound)
		return
	}

	taskStatus := taskState.GetTaskStatus()

	// build response similar to GetTestRunDetails logic
	taskData := &GetTestRunDetailedTask{
		Index:       uint64(taskState.Index()),
		ParentIndex: uint64(taskState.ParentIndex()),
		Name:        taskState.Name(),
		Title:       taskState.Title(),
		Started:     taskStatus.IsStarted,
		Completed:   taskStatus.IsStarted && !taskStatus.IsRunning,
		Timeout:     uint64(taskState.Timeout().Seconds()),
	}

	switch {
	case !taskStatus.IsStarted:
		taskData.Status = "pending"
	case taskStatus.IsRunning:
		taskData.Status = "running"
		taskData.StartTime = taskStatus.StartTime.UnixMilli()
		taskData.RunTime = uint64(time.Since(taskStatus.StartTime).Round(1 * time.Millisecond).Milliseconds()) //nolint:gosec // no overflow possible
	default:
		taskData.Status = "complete"
		taskData.StartTime = taskStatus.StartTime.UnixMilli()
		taskData.StopTime = taskStatus.StopTime.UnixMilli()
		taskData.RunTime = uint64(taskStatus.StopTime.Sub(taskStatus.StartTime).Round(1 * time.Millisecond).Milliseconds()) //nolint:gosec // no overflow possible
	}

	switch taskStatus.Result {
	case types.TaskResultNone:
		taskData.Result = "none"
	case types.TaskResultSuccess:
		taskData.Result = "success"
	case types.TaskResultFailure:
		taskData.Result = "failure"
	}

	if taskStatus.Error != nil {
		taskData.ResultError = taskStatus.Error.Error()
	}

	// logs (limit 100 entries)
	logCount := taskStatus.Logger.GetLogEntryCount()
	logStart := uint64(0)
	logLimit := uint64(100)

	if logCount > logLimit {
		logStart = logCount - logLimit
	}

	taskLog := taskStatus.Logger.GetLogEntries(logStart, logLimit)
	taskData.Log = make([]*GetTestRunDetailedTaskLog, len(taskLog))

	for i, log := range taskLog {
		logData := &GetTestRunDetailedTaskLog{
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

		taskData.Log[i] = logData
	}

	// config yaml
	if cfgData, err := yaml.Marshal(taskState.Config()); err == nil {
		taskData.ConfigYaml = string(cfgData)
	} else {
		taskData.ConfigYaml = fmt.Sprintf("failed marshalling config: %v", err)
	}

	// result yaml
	if resData, err := yaml.Marshal(taskState.GetTaskStatusVars().GetVarsMap(nil, false)); err == nil {
		taskData.ResultYaml = string(resData)
	} else {
		taskData.ResultYaml = fmt.Sprintf("failed marshalling result: %v", err)
	}

	ah.sendOKResponse(w, r.URL.String(), taskData)
}
