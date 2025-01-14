package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type TestRunPage struct {
	RunID        uint64         `json:"runId"`
	TestID       string         `json:"testId"`
	Name         string         `json:"name"`
	IsStarted    bool           `json:"started"`
	IsCompleted  bool           `json:"completed"`
	StartTime    time.Time      `json:"start_time"`
	StopTime     time.Time      `json:"stop_time"`
	Timeout      time.Duration  `json:"timeout"`
	Status       string         `json:"status"`
	IsSecTrimmed bool           `json:"is_sec_trimmed"`
	Tasks        []*TestRunTask `json:"tasks"`
}

type TestRunTask struct {
	Index            uint64            `json:"index"`
	ParentIndex      uint64            `json:"parent_index"`
	GraphLevels      []uint64          `json:"graph_levels"`
	HasChildren      bool              `json:"has_children"`
	Name             string            `json:"name"`
	Title            string            `json:"title"`
	IsStarted        bool              `json:"started"`
	IsCompleted      bool              `json:"completed"`
	StartTime        time.Time         `json:"start_time"`
	StopTime         time.Time         `json:"stop_time"`
	Timeout          time.Duration     `json:"timeout"`
	HasTimeout       bool              `json:"has_timeout"`
	RunTime          time.Duration     `json:"runtime"`
	HasRunTime       bool              `json:"has_runtime"`
	CustomRunTime    time.Duration     `json:"custom_runtime"`
	HasCustomRunTime bool              `json:"has_custom_runtime"`
	Status           string            `json:"status"`
	Result           string            `json:"result"`
	ResultError      string            `json:"result_error"`
	Log              []*TestRunTaskLog `json:"log"`
	ConfigYaml       string            `json:"config_yaml"`
	ResultYaml       string            `json:"result_yaml"`
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

	runID, pageError := strconv.ParseInt(vars["runId"], 10, 64)
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

	runID, pageError := strconv.ParseInt(vars["runId"], 10, 64)
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

//nolint:gocyclo // ignore
func (fh *FrontendHandler) getTestRunPageData(runID int64) (*TestRunPage, error) {
	test := fh.coordinator.GetTestByRunID(uint64(runID))
	if test == nil {
		return nil, fmt.Errorf("test not found")
	}

	pageData := &TestRunPage{
		RunID:        uint64(runID),
		TestID:       test.TestID(),
		Name:         test.Name(),
		StartTime:    test.StartTime(),
		StopTime:     test.StopTime(),
		Timeout:      test.Timeout(),
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
	}

	taskScheduler := test.GetTaskScheduler()
	if taskScheduler != nil && taskScheduler.GetTaskCount() > 0 {
		indentationMap := map[uint64]int{}

		allTasks := taskScheduler.GetAllTasks()
		cleanupTasks := taskScheduler.GetAllCleanupTasks()
		allTasks = append(allTasks, cleanupTasks...)

		for idx, task := range allTasks {
			taskState := taskScheduler.GetTaskState(task)
			taskStatus := taskState.GetTaskStatus()

			taskData := &TestRunTask{
				Index:       uint64(taskState.Index()),
				ParentIndex: uint64(taskState.ParentIndex()),
				Name:        taskState.Name(),
				Title:       taskState.Title(),
				IsStarted:   taskStatus.IsStarted,
				IsCompleted: taskStatus.IsStarted && !taskStatus.IsRunning,
				StartTime:   taskStatus.StartTime,
				StopTime:    taskStatus.StopTime,
				Timeout:     taskState.Timeout(),
				HasTimeout:  taskState.Timeout() > 0,
				GraphLevels: []uint64{},
			}

			indentation := 0
			if taskData.ParentIndex > 0 {
				indentation = indentationMap[taskData.ParentIndex] + 1
			}

			indentationMap[taskData.Index] = indentation

			if indentation > 0 {
				for i := 0; i < indentation; i++ {
					taskData.GraphLevels = append(taskData.GraphLevels, 0)
				}

				taskData.GraphLevels[indentation-1] = 3

				for i := idx - 1; i >= 0; i-- {
					if pageData.Tasks[i].Index == taskData.ParentIndex {
						pageData.Tasks[i].HasChildren = true
						break
					}

					if len(pageData.Tasks[i].GraphLevels) < indentation {
						break
					}

					if pageData.Tasks[i].ParentIndex == taskData.ParentIndex {
						pageData.Tasks[i].GraphLevels[indentation-1] = 2
						break
					}

					pageData.Tasks[i].GraphLevels[indentation-1] = 1
				}
			}

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

			if !fh.securityTrimmed {
				logCount := taskStatus.Logger.GetLogEntryCount()
				logStart := 0
				logLimit := 100

				if logCount > logLimit {
					logStart = logCount - logLimit
				}

				taskLog := taskStatus.Logger.GetLogEntries(logStart, logLimit)
				taskData.Log = make([]*TestRunTaskLog, len(taskLog))

				for i, log := range taskLog {
					logData := &TestRunTaskLog{
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
					taskData.ConfigYaml = fmt.Sprintf("\n%v\n", string(taskConfig))
				}

				taskStatusVars := taskState.GetTaskStatusVars().GetVarsMap(nil, false)
				if taskOutput, ok := taskStatusVars["outputs"]; ok {
					if customRunTimeSecondsRaw, ok := taskOutput.(map[string]interface{})["customRunTimeSeconds"]; ok {
						customRunTime, ok := customRunTimeSecondsRaw.(float64)
						if ok {
							taskData.CustomRunTime = time.Duration(customRunTime * float64(time.Second))
							taskData.HasCustomRunTime = true
						}
					}
				}

				taskResult, err := yaml.Marshal(taskStatusVars)
				if err != nil {
					taskData.ResultYaml = fmt.Sprintf("failed marshalling result: %v", err)
				} else {
					refComment := ""

					if taskState.ID() != "" {
						scopeOwner := "root"
						if scopeOwnerID := taskState.GetScopeOwner(); scopeOwnerID != 0 {
							scopeOwner = fmt.Sprintf("task %v", scopeOwnerID)
						}

						refComment = fmt.Sprintf("# available from %v scope via `tasks.%v`:\n", scopeOwner, taskState.ID())
					}

					taskData.ResultYaml = fmt.Sprintf("\n%v%v\n", refComment, string(taskResult))
				}
			}

			pageData.Tasks = append(pageData.Tasks, taskData)
		}
	}

	return pageData, nil
}
