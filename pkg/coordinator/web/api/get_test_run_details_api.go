package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
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
	w.Header().Set("Content-Type", "application/json")

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
		for _, task := range taskScheduler.GetAllTasks() {
			taskState := taskScheduler.GetTaskState(task)
			taskStatus := taskState.GetTaskStatus()

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
				taskData.StartTime = taskStatus.StartTime.Unix()
				taskData.RunTime = uint64(time.Since(taskStatus.StartTime).Round(1 * time.Millisecond).Milliseconds())
			default:
				taskData.Status = "complete"
				taskData.StartTime = taskStatus.StartTime.Unix()
				taskData.StopTime = taskStatus.StopTime.Unix()
				taskData.RunTime = uint64(taskStatus.StopTime.Sub(taskStatus.StartTime).Round(1 * time.Millisecond).Milliseconds())
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

			taskLog := taskStatus.Logger.GetLogEntries()
			taskData.Log = make([]*GetTestRunDetailedTaskLog, len(taskLog))

			for i, log := range taskLog {
				logData := &GetTestRunDetailedTaskLog{
					Time:    log.Time,
					Level:   uint64(log.Level),
					Message: log.Message,
					Data:    map[string]string{},
					DataLen: uint64(len(log.Data)),
				}

				for dataKey, dataVal := range log.Data {
					logData.Data[dataKey] = fmt.Sprintf("%v", dataVal)
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
