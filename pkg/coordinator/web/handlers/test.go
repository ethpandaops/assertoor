package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/ethpandaops/minccino/pkg/coordinator/types"
	"github.com/ethpandaops/minccino/pkg/coordinator/web"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type TestPage struct {
	Index       uint64          `json:"index"`
	Name        string          `json:"name"`
	IsStarted   bool            `json:"started"`
	IsCompleted bool            `json:"completed"`
	StartTime   time.Time       `json:"start_time"`
	StopTime    time.Time       `json:"stop_time"`
	Timeout     time.Duration   `json:"timeout"`
	Status      string          `json:"status"`
	Tasks       []*TestPageTask `json:"tasks"`
}

type TestPageTask struct {
	Index       uint64        `json:"index"`
	Name        string        `json:"name"`
	Title       string        `json:"title"`
	IsStarted   bool          `json:"started"`
	IsCompleted bool          `json:"completed"`
	StartTime   time.Time     `json:"start_time"`
	StopTime    time.Time     `json:"stop_time"`
	Timeout     time.Duration `json:"timeout"`
	HasTimeout  bool          `json:"has_timeout"`
	RunTime     time.Duration `json:"runtime"`
	HasRunTime  bool          `json:"has_runtime"`
	Status      string        `json:"status"`
	Result      string        `json:"result"`
	ResultError string        `json:"result_error"`
}

// Test will return the "test" page using a go template
func (fh *FrontendHandler) Test(w http.ResponseWriter, r *http.Request) {
	urlArgs := r.URL.Query()
	if urlArgs.Has("json") {
		fh.TestData(w, r)
		return
	}

	var templateFiles = append(web.LayoutTemplateFiles,
		"test/test.html",
	)
	var pageTemplate = web.GetTemplate(templateFiles...)
	data := web.InitPageData(w, r, "test", "/", "Test ", templateFiles)

	vars := mux.Vars(r)
	testIdx, pageError := strconv.ParseInt(vars["testIdx"], 10, 64)
	if pageError == nil {
		data.Data, pageError = fh.getTestPageData(testIdx)
	}
	if pageError != nil {
		web.HandlePageError(w, r, pageError)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	if web.HandleTemplateError(w, r, "test.go", "Test", "", pageTemplate.ExecuteTemplate(w, "layout", data)) != nil {
		return // an error has occurred and was processed
	}
}

func (fh *FrontendHandler) TestData(w http.ResponseWriter, r *http.Request) {
	var pageData *TestPage
	vars := mux.Vars(r)
	testIdx, pageError := strconv.ParseInt(vars["testIdx"], 10, 64)
	if pageError == nil {
		pageData, pageError = fh.getTestPageData(testIdx)
	}
	if pageError != nil {
		web.HandlePageError(w, r, pageError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(pageData)
	if err != nil {
		logrus.WithError(err).Error("error encoding test data")
		http.Error(w, "Internal server error", http.StatusServiceUnavailable)
	}
}

func (fh *FrontendHandler) getTestPageData(testIdx int64) (*TestPage, error) {
	var test types.Test
	allTests := fh.coordinator.GetTests()
	for idx := range allTests {
		if int64(idx) == testIdx {
			test = allTests[idx]
			break
		}
	}
	if test == nil {
		return nil, fmt.Errorf("Test not found")
	}

	pageData := &TestPage{
		Index:       uint64(testIdx),
		Name:        test.Name(),
		IsStarted:   test.Status() != types.TestStatusPending,
		IsCompleted: test.Status() > types.TestStatusRunning,
		StartTime:   test.StartTime(),
		StopTime:    test.StopTime(),
		Timeout:     test.Timeout(),
	}
	switch test.Status() {
	case types.TestStatusPending:
		pageData.Status = "pending"
	case types.TestStatusRunning:
		pageData.Status = "running"
	case types.TestStatusSuccess:
		pageData.Status = "success"
	case types.TestStatusFailure:
		pageData.Status = "failure"
	}

	taskScheduler := test.GetTaskScheduler()

	for _, task := range taskScheduler.GetAllTasks() {
		taskStatus := taskScheduler.GetTaskStatus(task)

		taskData := &TestPageTask{
			Index:       taskStatus.Index,
			Name:        task.Name(),
			Title:       task.Title(),
			IsStarted:   taskStatus.IsStarted,
			IsCompleted: taskStatus.IsStarted && !taskStatus.IsRunning,
			StartTime:   taskStatus.StartTime,
			StopTime:    taskStatus.StopTime,
			Timeout:     task.Timeout(),
			HasTimeout:  task.Timeout() > 0,
		}
		if !taskStatus.IsStarted {
			taskData.Status = "pending"
		} else if taskStatus.IsRunning {
			taskData.Status = "running"
			taskData.HasRunTime = true
			taskData.RunTime = time.Since(taskStatus.StartTime)
		} else {
			taskData.Status = "complete"
			taskData.HasRunTime = true
			taskData.RunTime = taskStatus.StopTime.Sub(taskStatus.StartTime)
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

		pageData.Tasks = append(pageData.Tasks, taskData)
	}

	return pageData, nil
}
