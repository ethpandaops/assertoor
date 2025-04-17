package handlers

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/sirupsen/logrus"
)

type IndexPageArgs struct {
	PageSize uint64 `json:"ps"`
	Page     uint64 `json:"p"`
}

type IndexPage struct {
	CanCancel      bool           `json:"can_cancel"`
	Tests          []*TestRunData `json:"tests"`
	TotalTests     uint64         `json:"total_tests"`
	FirstTestIndex uint64         `json:"first_test_index"`
	LastTestIndex  uint64         `json:"last_test_index"`

	TotalPages       uint64 `json:"total_pages"`
	PageSize         uint64 `json:"page_size"`
	CurrentPageIndex uint64 `json:"current_page_index"`
	PrevPageIndex    uint64 `json:"prev_page_index"`
	NextPageIndex    uint64 `json:"next_page_index"`
	LastPageIndex    uint64 `json:"last_page_index"`
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

func (fh *FrontendHandler) parseIndexPageArgs(r *http.Request) *IndexPageArgs {
	urlArgs := r.URL.Query()
	pageArgs := &IndexPageArgs{
		PageSize: 25,
		Page:     1,
	}

	if urlArgs.Has("ps") {
		val, err := strconv.ParseUint(urlArgs.Get("ps"), 10, 32)
		if err == nil && val > 0 && val <= 200 {
			pageArgs.PageSize = val
		}
	}

	if urlArgs.Has("p") {
		val, err := strconv.ParseUint(urlArgs.Get("p"), 10, 32)
		if err == nil && val > 0 {
			pageArgs.Page = val
		}
	}

	return pageArgs
}

// Index will return the "index" page using a go template
func (fh *FrontendHandler) Index(w http.ResponseWriter, r *http.Request) {
	urlArgs := r.URL.Query()
	if urlArgs.Has("json") {
		fh.IndexData(w, r)
		return
	}

	templateFiles := LayoutTemplateFiles
	templateFiles = append(templateFiles,
		"index/index.html",
		"sidebar/sidebar.html",
		"test/test_runs.html",
	)
	pageTemplate := fh.templates.GetTemplate(templateFiles...)
	data := fh.initPageData(r, "index", "/", "Index", templateFiles)
	pageArgs := fh.parseIndexPageArgs(r)

	var pageError error
	data.Data, pageError = fh.getIndexPageData(pageArgs)

	if pageError != nil {
		fh.HandlePageError(w, r, pageError)
		return
	}

	data.ShowSidebar = true
	data.SidebarData = fh.getSidebarData("*")

	w.Header().Set("Content-Type", "text/html")

	if fh.handleTemplateError(w, r, "index.go", "Index", pageTemplate.ExecuteTemplate(w, "layout", data)) != nil {
		return // an error has occurred and was processed
	}
}

func (fh *FrontendHandler) IndexData(w http.ResponseWriter, r *http.Request) {
	var pageData *IndexPage

	var pageError error

	pageArgs := fh.parseIndexPageArgs(r)
	pageData, pageError = fh.getIndexPageData(pageArgs)

	if pageError != nil {
		fh.HandlePageError(w, r, pageError)
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
func (fh *FrontendHandler) getIndexPageData(pageArgs *IndexPageArgs) (*IndexPage, error) {
	pageData := &IndexPage{
		CanCancel: fh.isAPIEnabled && !fh.securityTrimmed,
	}

	pageOffset := (pageArgs.Page - 1) * pageArgs.PageSize
	testInstances, totalTests := fh.coordinator.GetTestHistory("", 0, pageOffset, pageArgs.PageSize)

	for idx, test := range testInstances {
		idx64 := uint64(idx) //nolint:gosec // no overflow possible
		pageData.Tests = append(pageData.Tests, fh.getTestRunData(idx64, test))
	}

	pageData.TotalTests = totalTests
	pageData.TotalPages = uint64(math.Ceil(float64(totalTests) / float64(pageArgs.PageSize)))
	pageData.CurrentPageIndex = pageArgs.Page
	pageData.PageSize = pageArgs.PageSize
	pageData.FirstTestIndex = (pageArgs.Page - 1) * pageArgs.PageSize
	pageData.LastTestIndex = pageData.FirstTestIndex + uint64(len(pageData.Tests)) - 1
	pageData.LastPageIndex = pageData.TotalPages - 1

	if pageData.CurrentPageIndex > 1 {
		pageData.PrevPageIndex = pageData.CurrentPageIndex - 1
	}

	if pageData.CurrentPageIndex < pageData.TotalPages {
		pageData.NextPageIndex = pageData.CurrentPageIndex + 1
	}

	return pageData, nil
}

func (fh *FrontendHandler) getTestRunData(idx uint64, test types.Test) *TestRunData {
	testData := &TestRunData{
		RunID:      test.RunID(),
		TestID:     test.TestID(),
		Index:      idx,
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
		testData.TaskCount = taskScheduler.GetTaskCount()
	}

	return testData
}
