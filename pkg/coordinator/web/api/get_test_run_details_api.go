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

type GetTestRunDetailsResponse struct {
	RunID     uint64                    `json:"run_id"`
	TestID    string                    `json:"test_id"`
	Name      string                    `json:"name"`
	Status    types.TestStatus          `json:"status"`
	StartTime int64                     `json:"start_time"`
	StopTime  int64                     `json:"stop_time"`
	Tasks     []*GetTestRunDetailedTask `json:"tasks"`
}

type GetTestRunDetailedTask struct {
	Index       uint64                       `json:"index"`
	ParentIndex uint64                       `json:"parent_index"`
	Name        string                       `json:"name"`
	Title       string                       `json:"title"`
	Started     bool                         `json:"started"`
	Completed   bool                         `json:"completed"`
	StartTime   int64                        `json:"start_time"`
	StopTime    int64                        `json:"stop_time"`
	Timeout     uint64                       `json:"timeout"`
	RunTime     uint64                       `json:"runtime"`
	Status      string                       `json:"status"`
	Result      string                       `json:"result"`
	ResultError string                       `json:"result_error"`
	Log         []*GetTestRunDetailedTaskLog `json:"log"`
	ConfigYaml  string                       `json:"config_yaml"`
	ResultYaml  string                       `json:"result_yaml"`
}

type GetTestRunDetailedTaskLog struct {
	Time    time.Time         `json:"time"`
	Level   uint64            `json:"level"`
	Message string            `json:"msg"`
	DataLen uint64            `json:"datalen"`
	Data    map[string]string `json:"data"`
}

// GetTestRunDetails godoc
// @Id getTestRunDetails
// @Summary Get detailed test run by run ID
// @Tags TestRun
// @Description Returns the run details with given ID. Includes a summary and a list of task with all details (incl. logs & task configurations)
// @Produce json
// @Param runId path string true "ID of the test run to get details for"
// @Success 200 {object} Response{data=GetTestRunDetailsResponse} "Success"
// @Failure 400 {object} Response "Failure"
// @Failure 500 {object} Response "Server Error"
// @Router /api/v1/test_run/{runId}/details [get]
func (ah *APIHandler) GetTestRunDetails(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", contentTypeJSON)

	vars := mux.Vars(r)

	runID, err := strconv.ParseUint(vars["runId"], 10, 64)
	if err != nil {
		ah.sendErrorResponse(w, r.URL.String(), "invalid runId provided", http.StatusBadRequest)
		return
	}

	testInstance := ah.coordinator.GetTestByRunID(runID)
	if testInstance == nil {
		ah.sendErrorResponse(w, r.URL.String(), "test run not found", http.StatusNotFound)
		return
	}

	response := &GetTestRunDetailsResponse{
		RunID:  testInstance.RunID(),
		TestID: testInstance.TestID(),
		Name:   testInstance.Name(),
		Status: testInstance.Status(),
		Tasks:  []*GetTestRunDetailedTask{},
	}

	if !testInstance.StartTime().IsZero() {
		response.StartTime = testInstance.StartTime().Unix()
	}

	if !testInstance.StopTime().IsZero() {
		response.StopTime = testInstance.StopTime().Unix()
	}

	taskScheduler := testInstance.GetTaskScheduler()
	if taskScheduler != nil && taskScheduler.GetTaskCount() > 0 {
		allTasks := taskScheduler.GetAllTasks()
		cleanupTasks := taskScheduler.GetAllCleanupTasks()
		allTasks = append(allTasks, cleanupTasks...)

		for _, task := range allTasks {
			taskState := taskScheduler.GetTaskState(task)
			taskStatus := taskState.GetTaskStatus()

			taskData := &GetTestRunDetailedTask{
				Index:       uint64(taskState.Index()),
				ParentIndex: uint64(taskState.ParentIndex()),
				Name:        taskState.Name(),
				Title:       taskState.Title(),
				Started:     taskStatus.IsStarted,
				Completed:   taskStatus.IsStarted && !taskStatus.IsRunning,
			}

			timeout := taskState.Timeout().Milliseconds()
			if timeout < 0 {
				taskData.Timeout = 0
			} else {
				taskData.Timeout = uint64(timeout)
			}

			switch {
			case !taskStatus.IsStarted:
				taskData.Status = TaskStatusPending
			case taskStatus.IsRunning:
				taskData.Status = TaskStatusRunning
				taskData.StartTime = taskStatus.StartTime.UnixMilli()

				duration := time.Since(taskStatus.StartTime).Round(1 * time.Millisecond).Milliseconds()
				if duration < 0 {
					taskData.RunTime = 0
				} else {
					taskData.RunTime = uint64(duration)
				}
			default:
				taskData.Status = TaskStatusComplete
				taskData.StartTime = taskStatus.StartTime.UnixMilli()
				taskData.StopTime = taskStatus.StopTime.UnixMilli()

				duration := taskStatus.StopTime.Sub(taskStatus.StartTime).Round(1 * time.Millisecond).Milliseconds()
				if duration < 0 {
					taskData.RunTime = 0
				} else {
					taskData.RunTime = uint64(duration)
				}
			}

			switch taskStatus.Result {
			case types.TaskResultNone:
				taskData.Result = TaskResultNone
			case types.TaskResultSuccess:
				taskData.Result = TaskResultSuccess
			case types.TaskResultFailure:
				taskData.Result = TaskResultFailure
			}

			if taskStatus.Error != nil {
				taskData.ResultError = taskStatus.Error.Error()
			}

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

			taskConfig, err := yaml.Marshal(taskState.Config())
			if err != nil {
				taskData.ConfigYaml = fmt.Sprintf("failed marshalling config: %v", err)
			} else {
				taskData.ConfigYaml = string(taskConfig)
			}

			taskResult, err := yaml.Marshal(taskState.GetTaskStatusVars().GetVarsMap(nil, false))
			if err != nil {
				taskData.ResultYaml = fmt.Sprintf("failed marshalling result: %v", err)
			} else {
				taskData.ResultYaml = string(taskResult)
			}

			response.Tasks = append(response.Tasks, taskData)
		}
	}

	ah.sendOKResponse(w, r.URL.String(), response)
}
