package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/ethpandaops/assertoor/pkg/coordinator/web"
	"github.com/sirupsen/logrus"
)

type IndexPage struct {
	TestDescriptors []*IndexPageTestDescriptor `json:"test_descriptors"`
	Tests           []*TestRunData             `json:"tests"`
}

type IndexPageTestDescriptor struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Source string `json:"source"`
	Config string `json:"config"`
}

type TestRunData struct {
	RunID       uint64        `json:"run_id"`
	TestID      string        `json:"test_id"`
	Index       uint64        `json:"index"`
	Name        string        `json:"name"`
	IsStarted   bool          `json:"started"`
	IsCompleted bool          `json:"completed"`
	StartTime   time.Time     `json:"start_time"`
	StopTime    time.Time     `json:"stop_time"`
	Timeout     time.Duration `json:"timeout"`
	HasTimeout  bool          `json:"has_timeout"`
	RunTime     time.Duration `json:"runtime"`
	HasRunTime  bool          `json:"has_runtime"`
	Status      string        `json:"status"`
	TaskCount   uint64        `json:"task_count"`
}

// Index will return the "index" page using a go template
func (fh *FrontendHandler) Index(w http.ResponseWriter, r *http.Request) {
	urlArgs := r.URL.Query()
	if urlArgs.Has("json") {
		fh.IndexData(w, r)
		return
	}

	templateFiles := web.LayoutTemplateFiles
	templateFiles = append(templateFiles,
		"index/index.html",
		"sidebar/sidebar.html",
		"test/test_runs.html",
	)
	pageTemplate := web.GetTemplate(templateFiles...)
	data := web.InitPageData(r, "index", "/", "Index", templateFiles)

	var pageError error
	data.Data, pageError = fh.getIndexPageData()

	if pageError != nil {
		web.HandlePageError(w, r, pageError)
		return
	}

	data.ShowSidebar = true
	data.SidebarData = fh.getSidebarData("*")

	w.Header().Set("Content-Type", "text/html")

	if web.HandleTemplateError(w, r, "index.go", "Index", "", pageTemplate.ExecuteTemplate(w, "layout", data)) != nil {
		return // an error has occurred and was processed
	}
}

func (fh *FrontendHandler) IndexData(w http.ResponseWriter, r *http.Request) {
	var pageData *IndexPage

	var pageError error
	pageData, pageError = fh.getIndexPageData()

	if pageError != nil {
		web.HandlePageError(w, r, pageError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	err := json.NewEncoder(w).Encode(pageData)
	if err != nil {
		logrus.WithError(err).Error("error encoding index data")

		//nolint:gocritic // ignore
		http.Error(w, "Internal server error", http.StatusServiceUnavailable)
	}
}

//nolint:unparam // ignore
func (fh *FrontendHandler) getIndexPageData() (*IndexPage, error) {
	pageData := &IndexPage{}

	// tasks list
	pageData.Tests = []*TestRunData{}

	testInstances := append(fh.coordinator.GetTestHistory(), fh.coordinator.GetTestQueue()...)
	for idx, test := range testInstances {
		pageData.Tests = append(pageData.Tests, fh.getTestRunData(idx, test))
	}

	return pageData, nil
}

func (fh *FrontendHandler) getTestRunData(idx int, test types.Test) *TestRunData {
	testData := &TestRunData{
		RunID:      test.RunID(),
		TestID:     test.TestID(),
		Index:      uint64(idx),
		Name:       test.Name(),
		StartTime:  test.StartTime(),
		StopTime:   test.StopTime(),
		Timeout:    test.Timeout(),
		HasTimeout: test.Timeout() > 0,
		Status:     string(test.Status()),
	}

	switch test.Status() {
	case types.TestStatusPending:
	case types.TestStatusRunning:
		testData.IsStarted = true
	case types.TestStatusSuccess:
		testData.IsStarted = true
		testData.IsCompleted = true
	case types.TestStatusFailure:
		testData.IsStarted = true
		testData.IsCompleted = true
	case types.TestStatusSkipped:
	case types.TestStatusAborted:
	}

	if testData.IsCompleted {
		testData.RunTime = testData.StopTime.Sub(testData.StartTime).Round(1 * time.Millisecond)
		testData.HasRunTime = true
	} else if testData.IsStarted {
		testData.RunTime = time.Since(testData.StartTime).Round(1 * time.Millisecond)
		testData.HasRunTime = true
	}

	if taskScheduler := test.GetTaskScheduler(); taskScheduler != nil {
		testData.TaskCount = uint64(taskScheduler.GetTaskCount())
	}

	return testData
}
