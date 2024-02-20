package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/ethpandaops/assertoor/pkg/coordinator/web"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type TestRunPage struct {
	RunID       uint64         `json:"runId"`
	TestID      string         `json:"testId"`
	Name        string         `json:"name"`
	IsStarted   bool           `json:"started"`
	IsCompleted bool           `json:"completed"`
	StartTime   time.Time      `json:"start_time"`
	StopTime    time.Time      `json:"stop_time"`
	Timeout     time.Duration  `json:"timeout"`
	Status      string         `json:"status"`
	Tasks       []*TestRunTask `json:"tasks"`
}

type TestRunTask struct {
	Index       uint64            `json:"index"`
	ParentIndex uint64            `json:"parent_index"`
	IndentPx    uint64            `json:"indent_px"`
	Name        string            `json:"name"`
	Title       string            `json:"title"`
	IsStarted   bool              `json:"started"`
	IsCompleted bool              `json:"completed"`
	StartTime   time.Time         `json:"start_time"`
	StopTime    time.Time         `json:"stop_time"`
	Timeout     time.Duration     `json:"timeout"`
	HasTimeout  bool              `json:"has_timeout"`
	RunTime     time.Duration     `json:"runtime"`
	HasRunTime  bool              `json:"has_runtime"`
	Status      string            `json:"status"`
	Result      string            `json:"result"`
	ResultError string            `json:"result_error"`
	Log         []*TestRunTaskLog `json:"log"`
	ConfigYaml  string            `json:"config_yaml"`
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

	templateFiles := web.LayoutTemplateFiles
	templateFiles = append(templateFiles,
		"test_run/test_run.html",
		"sidebar/sidebar.html",
	)
	pageTemplate := web.GetTemplate(templateFiles...)
	data := web.InitPageData(r, "test", "/", "Test ", templateFiles)

	vars := mux.Vars(r)

	var pageData *TestRunPage

	runID, pageError := strconv.ParseInt(vars["runId"], 10, 64)
	if pageError == nil {
		pageData, pageError = fh.getTestRunPageData(runID)
		data.Data = pageData
	}

	if pageError != nil {
		web.HandlePageError(w, r, pageError)
		return
	}

	data.ShowSidebar = true
	data.SidebarData = fh.getSidebarData(pageData.TestID)

	w.Header().Set("Content-Type", "text/html")

	if web.HandleTemplateError(w, r, "test_run.go", "Test Run", "", pageTemplate.ExecuteTemplate(w, "layout", data)) != nil {
		return // an error has occurred and was processed
	}
}

func (fh *FrontendHandler) TestRunData(w http.ResponseWriter, r *http.Request) {
	var pageData *TestRunPage

	vars := mux.Vars(r)

	runID, pageError := strconv.ParseInt(vars["runId"], 10, 64)
	if pageError == nil {
		pageData, pageError = fh.getTestRunPageData(runID)
	}

	if pageError != nil {
		web.HandlePageError(w, r, pageError)
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

func (fh *FrontendHandler) getTestRunPageData(runID int64) (*TestRunPage, error) {
	test := fh.coordinator.GetTestByRunID(uint64(runID))
	if test == nil {
		return nil, fmt.Errorf("test not found")
	}

	pageData := &TestRunPage{
		RunID:     uint64(runID),
		TestID:    test.TestID(),
		Name:      test.Name(),
		StartTime: test.StartTime(),
		StopTime:  test.StopTime(),
		Timeout:   test.Timeout(),
		Status:    string(test.Status()),
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
	}

	taskScheduler := test.GetTaskScheduler()
	if taskScheduler != nil && taskScheduler.GetTaskCount() > 0 {
		indentationMap := map[uint64]int{}

		for _, task := range taskScheduler.GetAllTasks() {
			taskStatus := taskScheduler.GetTaskStatus(task)

			taskData := &TestRunTask{
				Index:       taskStatus.Index,
				ParentIndex: taskStatus.ParentIndex,
				Name:        task.Name(),
				Title:       task.Title(),
				IsStarted:   taskStatus.IsStarted,
				IsCompleted: taskStatus.IsStarted && !taskStatus.IsRunning,
				StartTime:   taskStatus.StartTime,
				StopTime:    taskStatus.StopTime,
				Timeout:     task.Timeout(),
				HasTimeout:  task.Timeout() > 0,
			}

			indentation := 0
			if taskData.ParentIndex > 0 {
				indentation = indentationMap[taskData.ParentIndex] + 1
			}

			indentationMap[taskData.Index] = indentation
			taskData.IndentPx = uint64(20 * indentation)

			switch {
			case !taskStatus.IsStarted:
				taskData.Status = "pending"
			case taskStatus.IsRunning:
				taskData.Status = "running"
				taskData.HasRunTime = true
				taskData.RunTime = time.Since(taskStatus.StartTime).Round(1 * time.Millisecond)
			default:
				taskData.Status = "complete"
				taskData.HasRunTime = true
				taskData.RunTime = taskStatus.StopTime.Sub(taskStatus.StartTime).Round(1 * time.Millisecond)
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
			taskData.Log = make([]*TestRunTaskLog, len(taskLog))

			for i, log := range taskLog {
				logData := &TestRunTaskLog{
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

			taskConfig, err := yaml.Marshal(task.Config())
			if err != nil {
				taskData.ConfigYaml = fmt.Sprintf("failed marshalling config: %v", err)
			} else {
				taskData.ConfigYaml = string(taskConfig)
			}

			pageData.Tasks = append(pageData.Tasks, taskData)
		}
	}

	return pageData, nil
}
