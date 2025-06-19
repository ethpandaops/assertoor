package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/erigontech/assertoor/pkg/coordinator/db"
	"github.com/erigontech/assertoor/pkg/coordinator/types"
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
	Index       uint64                 `json:"index"`
	ParentIndex uint64                 `json:"parent_index"`
	Name        string                 `json:"name"`
	Title       string                 `json:"title"`
	Started     bool                   `json:"started"`
	Completed   bool                   `json:"completed"`
	StartTime   int64                  `json:"start_time"`
	StopTime    int64                  `json:"stop_time"`
	Timeout     uint64                 `json:"timeout"`
	RunTime     uint64                 `json:"runtime"`
	Status      string                 `json:"status"`
	Result      string                 `json:"result"`
	ResultFiles []GetTestRunTaskResult `json:"result_files"`
	ResultError string                 `json:"result_error"`
}

type GetTestRunTaskResult struct {
	Type  string `json:"type"`
	Index uint64 `json:"index"`
	Name  string `json:"name"`
	Size  uint64 `json:"size"`
	URL   string `json:"url"`
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

	// get result headers
	resultHeaderMap := map[uint64][]db.TaskResultHeader{}

	resultHeaders, err := ah.coordinator.Database().GetAllTaskResultHeaders(runID)
	if err != nil {
		ah.sendErrorResponse(w, r.URL.String(), "failed to get result headers", http.StatusInternalServerError)
		return
	}

	for _, header := range resultHeaders {
		resultHeaderMap[header.TaskID] = append(resultHeaderMap[header.TaskID], header)
	}

	taskScheduler := testInstance.GetTaskScheduler()
	if taskScheduler != nil && taskScheduler.GetTaskCount() > 0 {
		allTasks := taskScheduler.GetAllTasks()
		cleanupTasks := taskScheduler.GetAllCleanupTasks()
		allTasks = append(allTasks, cleanupTasks...)

		for _, task := range allTasks {
			taskState := taskScheduler.GetTaskState(task)
			taskStatus := taskState.GetTaskStatus()

			taskData := &GetTestRunTask{
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

			if len(resultHeaderMap[uint64(taskState.Index())]) > 0 {
				taskData.ResultFiles = make([]GetTestRunTaskResult, len(resultHeaderMap[uint64(taskState.Index())]))
				for i, header := range resultHeaderMap[uint64(taskState.Index())] {
					taskData.ResultFiles[i] = GetTestRunTaskResult{
						Type:  header.Type,
						Index: header.Index,
						Name:  header.Name,
						Size:  header.Size,
						URL:   fmt.Sprintf("/api/v1/test_run/%v/task/%v/result/%v/%v", runID, taskState.Index(), header.Type, header.Index),
					}
				}
			}

			response.Tasks = append(response.Tasks, taskData)
		}
	}

	ah.sendOKResponse(w, r.URL.String(), response)
}
