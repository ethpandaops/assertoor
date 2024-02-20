package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/gorilla/mux"
)

type GetTestRunResponse struct {
	RunID     uint64            `json:"run_id"`
	TestID    string            `json:"test_id"`
	Name      string            `json:"name"`
	Status    types.TestStatus  `json:"status"`
	StartTime int64             `json:"start_time"`
	StopTime  int64             `json:"stop_time"`
	Tasks     []*GetTestRunTask `json:"tasks"`
}

type GetTestRunTask struct {
	Index       uint64 `json:"index"`
	ParentIndex uint64 `json:"parent_index"`
	Name        string `json:"name"`
	Title       string `json:"title"`
	Started     bool   `json:"started"`
	Completed   bool   `json:"completed"`
	StartTime   int64  `json:"start_time"`
	StopTime    int64  `json:"stop_time"`
	Timeout     uint64 `json:"timeout"`
	RunTime     uint64 `json:"runtime"`
	Status      string `json:"status"`
	Result      string `json:"result"`
	ResultError string `json:"result_error"`
}

// GetTestRun godoc
// @Id getTestRun
// @Summary Get test run by run ID
// @Tags TestRun
// @Description Returns the run details with given ID. Includes a summary and a list of task with limited details
// @Produce json
// @Param runId path string true "ID of the test run to get details for"
// @Success 200 {object} Response{data=GetTestRunResponse} "Success"
// @Failure 400 {object} Response "Failure"
// @Failure 500 {object} Response "Server Error"
// @Router /api/v1/test_run/{runId} [get]
func (ah *APIHandler) GetTestRun(w http.ResponseWriter, r *http.Request) {
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

	response := &GetTestRunResponse{
		RunID:  testInstance.RunID(),
		TestID: testInstance.TestID(),
		Name:   testInstance.Name(),
		Status: testInstance.Status(),
		Tasks:  []*GetTestRunTask{},
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
			taskStatus := taskScheduler.GetTaskStatus(task)

			taskData := &GetTestRunTask{
				Index:       taskStatus.Index,
				ParentIndex: taskStatus.ParentIndex,
				Name:        task.Name(),
				Title:       task.Title(),
				Started:     taskStatus.IsStarted,
				Completed:   taskStatus.IsStarted && !taskStatus.IsRunning,
				Timeout:     uint64(task.Timeout().Seconds()),
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

			response.Tasks = append(response.Tasks, taskData)
		}
	}

	ah.sendOKResponse(w, r.URL.String(), response)
}
