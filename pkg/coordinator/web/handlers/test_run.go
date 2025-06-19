package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/erigontech/assertoor/pkg/coordinator/db"
	"github.com/erigontech/assertoor/pkg/coordinator/types"
	"github.com/erigontech/assertoor/pkg/coordinator/web/api"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type TestRunPage struct {
	RunID        uint64         `json:"runId"`
	TestID       string         `json:"testId"`
	Name         string         `json:"name"`
	IsStarted    bool           `json:"started"`
	IsCompleted  bool           `json:"completed"`
	StartTime    int64          `json:"start_time"`
	StopTime     int64          `json:"stop_time"`
	Timeout      int64          `json:"timeout"` // milliseconds
	Status       string         `json:"status"`
	IsSecTrimmed bool           `json:"is_sec_trimmed"`
	Tasks        []*TestRunTask `json:"tasks"`
}

type TestRunTask struct {
	Index       uint64               `json:"index"`
	ParentIndex uint64               `json:"parent_index"`
	Name        string               `json:"name"`
	Title       string               `json:"title"`
	IsStarted   bool                 `json:"started"`
	IsCompleted bool                 `json:"completed"`
	StartTime   int64                `json:"start_time"`
	StopTime    int64                `json:"stop_time"`
	Timeout     int64                `json:"timeout"` // milliseconds
	RunTime     int64                `json:"runtime"` // milliseconds
	Status      string               `json:"status"`
	Result      string               `json:"result"`
	ResultError string               `json:"result_error"`
	ResultFiles []*TestRunTaskResult `json:"result_files"`
}

type TestRunTaskResult struct {
	Type  string `json:"type"`
	Index uint64 `json:"index"`
	Name  string `json:"name"`
	Size  uint64 `json:"size"`
	URL   string `json:"url"`
}

type TestRunTaskLog struct {
	Time    time.Time         `json:"time"`
	Level   uint64            `json:"level"`
	Message string            `json:"msg"`
	DataLen uint64            `json:"datalen"`
	Data    map[string]string `json:"data"`
}

// Test will return the "test" page using a go template
func (fh *FrontendHandler) TestRun(w http.ResponseWriter, r *http.Request) {
	urlArgs := r.URL.Query()
	if urlArgs.Has("json") {
		fh.TestRunData(w, r)
		return
	}

	templateFiles := LayoutTemplateFiles
	templateFiles = append(templateFiles,
		"test_run/test_run.html",
		"sidebar/sidebar.html",
	)
	pageTemplate := fh.templates.GetTemplate(templateFiles...)
	data := fh.initPageData(r, "test", "/", "Test ", templateFiles)

	vars := mux.Vars(r)

	var pageData *TestRunPage

	runID, pageError := strconv.ParseUint(vars["runId"], 10, 64)
	if pageError == nil {
		pageData, pageError = fh.getTestRunPageData(runID)
		data.Data = pageData
	}

	if pageError != nil {
		fh.HandlePageError(w, r, pageError)
		return
	}

	data.ShowSidebar = true
	data.SidebarData = fh.getSidebarData(pageData.TestID)

	w.Header().Set("Content-Type", "text/html")

	if fh.handleTemplateError(w, r, "test_run.go", "Test Run", pageTemplate.ExecuteTemplate(w, "layout", data)) != nil {
		return // an error has occurred and was processed
	}
}

func (fh *FrontendHandler) TestRunData(w http.ResponseWriter, r *http.Request) {
	var pageData *TestRunPage

	vars := mux.Vars(r)

	runID, pageError := strconv.ParseUint(vars["runId"], 10, 64)
	if pageError == nil {
		pageData, pageError = fh.getTestRunPageData(runID)
	}

	if pageError != nil {
		fh.HandlePageError(w, r, pageError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	err := json.NewEncoder(w).Encode(pageData)
	if err != nil {
		logrus.WithError(err).Error("error encoding test data")

		//nolint:gocritic // ignore
		http.Error(w, "Internal server error", http.StatusServiceUnavailable)
	}
}

func (fh *FrontendHandler) getTestRunPageData(runID uint64) (*TestRunPage, error) {
	test := fh.coordinator.GetTestByRunID(runID)
	if test == nil {
		return nil, fmt.Errorf("test not found")
	}

	pageData := &TestRunPage{
		RunID:        runID,
		TestID:       test.TestID(),
		Name:         test.Name(),
		StartTime:    test.StartTime().UnixMilli(),
		StopTime:     test.StopTime().UnixMilli(),
		Timeout:      test.Timeout().Milliseconds(),
		Status:       string(test.Status()),
		IsSecTrimmed: fh.securityTrimmed,
	}

	switch test.Status() {
	case types.TestStatusPending:
	case types.TestStatusRunning:
		pageData.IsStarted = true
	case types.TestStatusSuccess:
		pageData.IsStarted = true
		pageData.IsCompleted = true
	case types.TestStatusFailure:
		pageData.IsStarted = true
		pageData.IsCompleted = true
	case types.TestStatusSkipped:
	case types.TestStatusAborted:
		pageData.IsStarted = true
		pageData.IsCompleted = true
	}

	// get result headers
	resultHeaderMap := map[uint64][]db.TaskResultHeader{}

	if !fh.securityTrimmed {
		resultHeaders, err := fh.coordinator.Database().GetAllTaskResultHeaders(runID)
		if err != nil {
			logrus.WithError(err).Warnf("Failed to get result headers for run %d", runID)
		} else {
			for _, header := range resultHeaders {
				resultHeaderMap[header.TaskID] = append(resultHeaderMap[header.TaskID], header)
			}
		}
	}

	taskScheduler := test.GetTaskScheduler()
	if taskScheduler != nil && taskScheduler.GetTaskCount() > 0 {
		taskMap := make(map[uint64]*TestRunTask)

		allTasks := taskScheduler.GetAllTasks()
		cleanupTasks := taskScheduler.GetAllCleanupTasks()
		allTasks = append(allTasks, cleanupTasks...)

		for _, task := range allTasks {
			taskState := taskScheduler.GetTaskState(task)
			taskStatus := taskState.GetTaskStatus()

			taskData := &TestRunTask{
				Index:       uint64(taskState.Index()),
				ParentIndex: uint64(taskState.ParentIndex()),
				Name:        taskState.Name(),
				Title:       taskState.Title(),
				IsStarted:   taskStatus.IsStarted,
				IsCompleted: taskStatus.IsStarted && !taskStatus.IsRunning,
				StartTime:   taskStatus.StartTime.UnixMilli(),
				StopTime:    taskStatus.StopTime.UnixMilli(),
				Timeout:     taskState.Timeout().Milliseconds(),
				RunTime:     time.Since(taskStatus.StartTime).Milliseconds(),
				Status:      "complete",
				Result:      "unknown",
				ResultError: "",
				ResultFiles: nil,
			}

			switch {
			case !taskStatus.IsStarted:
				taskData.Status = api.TaskStatusPending
			case taskStatus.IsRunning:
				taskData.Status = api.TaskStatusRunning
				taskData.RunTime = time.Since(taskStatus.StartTime).Round(1 * time.Millisecond).Milliseconds()
			default:
				taskData.Status = api.TaskStatusComplete
				taskData.RunTime = taskStatus.StopTime.Sub(taskStatus.StartTime).Round(1 * time.Millisecond).Milliseconds()
			}

			switch taskStatus.Result {
			case types.TaskResultNone:
				taskData.Result = api.TaskResultNone
			case types.TaskResultSuccess:
				taskData.Result = api.TaskResultSuccess
			case types.TaskResultFailure:
				taskData.Result = api.TaskResultFailure
			}

			if taskStatus.Error != nil {
				taskData.ResultError = taskStatus.Error.Error()
			}

			if !fh.securityTrimmed && len(resultHeaderMap[taskData.Index]) > 0 {
				taskData.ResultFiles = make([]*TestRunTaskResult, len(resultHeaderMap[taskData.Index]))
				for i, header := range resultHeaderMap[taskData.Index] {
					resName := header.Name
					if resName == "" {
						resName = fmt.Sprintf("%v-%v", header.Type, header.Index)
					}

					taskData.ResultFiles[i] = &TestRunTaskResult{
						Type:  header.Type,
						Index: header.Index,
						Name:  resName,
						Size:  header.Size,
						URL:   fmt.Sprintf("/api/v1/test_run/%v/task/%v/result/%v/%v", runID, taskData.Index, header.Type, header.Index),
					}
				}
			}

			pageData.Tasks = append(pageData.Tasks, taskData)
			taskMap[taskData.Index] = taskData
		}
	}

	return pageData, nil
}
