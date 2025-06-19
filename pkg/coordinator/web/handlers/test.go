package handlers

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"

	"github.com/erigontech/assertoor/pkg/coordinator/types"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type TestPageArgs struct {
	PageSize uint64 `json:"ps"`
	Page     uint64 `json:"p"`
}

type TestPage struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Source    string `json:"source"`
	Config    string `json:"config"`
	CanStart  bool   `json:"can_start"`
	CanCancel bool   `json:"can_cancel"`

	Tests          []*TestRunData `json:"tests"`
	TotalTests     uint64         `json:"total_tests"`
	FirstTestIndex uint64         `json:"first_test_index"`
	LastTestIndex  uint64         `json:"last_test_index"`

	TotalPages       uint64 `json:"total_pages"`
	PageSize         uint64 `json:"page_size"`
	CurrentPageIndex uint64 `json:"current_page_index"`
	PrevPageIndex    uint64 `json:"prev_page_index"`
	NextPageIndex    uint64 `json:"next_page_index"`
}

func (fh *FrontendHandler) parseTestPageArgs(r *http.Request) *TestPageArgs {
	urlArgs := r.URL.Query()
	pageArgs := &TestPageArgs{
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
func (fh *FrontendHandler) TestPage(w http.ResponseWriter, r *http.Request) {
	urlArgs := r.URL.Query()
	if urlArgs.Has("json") {
		fh.IndexData(w, r)
		return
	}

	templateFiles := LayoutTemplateFiles
	templateFiles = append(templateFiles,
		"test/test.html",
		"sidebar/sidebar.html",
		"test/test_runs.html",
	)
	pageTemplate := fh.templates.GetTemplate(templateFiles...)
	vars := mux.Vars(r)
	data := fh.initPageData(r, "test", "/", "Test", templateFiles)
	pageArgs := fh.parseTestPageArgs(r)

	var pageError error

	var pageData *TestPage

	pageData, pageError = fh.getTestPageData(vars["testId"], pageArgs)
	if pageError != nil {
		fh.HandlePageError(w, r, pageError)
		return
	}

	data.Data = pageData
	data.ShowSidebar = true
	data.SidebarData = fh.getSidebarData(pageData.ID)

	w.Header().Set("Content-Type", "text/html")

	if fh.handleTemplateError(w, r, "index.go", "Index", pageTemplate.ExecuteTemplate(w, "layout", data)) != nil {
		return // an error has occurred and was processed
	}
}

func (fh *FrontendHandler) TestPageData(w http.ResponseWriter, r *http.Request) {
	var pageData *TestPage

	var pageError error

	vars := mux.Vars(r)
	pageArgs := fh.parseTestPageArgs(r)
	pageData, pageError = fh.getTestPageData(vars["testId"], pageArgs)

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

func (fh *FrontendHandler) getTestPageData(testID string, pageArgs *TestPageArgs) (*TestPage, error) {
	var testDescriptor types.TestDescriptor

	for _, testDescr := range fh.coordinator.TestRegistry().GetTestDescriptors() {
		if testDescr.Err() != nil {
			continue
		}

		if testDescr.ID() == testID {
			testDescriptor = testDescr
			break
		}
	}

	if testDescriptor == nil {
		return nil, fmt.Errorf("unknown test id: %v", testID)
	}

	testConfig := testDescriptor.Config()
	pageData := &TestPage{
		ID:     testID,
		Name:   testConfig.Name,
		Source: testDescriptor.Source(),
	}

	if fh.isAPIEnabled && !fh.securityTrimmed {
		testCfgJSON, err := json.Marshal(testDescriptor.Vars().GetVarsMap(nil, false))
		if err != nil {
			return nil, fmt.Errorf("failed marshalling test vars: %v", err)
		}

		pageData.Config = string(testCfgJSON)
		pageData.CanStart = true
		pageData.CanCancel = true
	} else {
		pageData.Config = "null"
	}

	// test runs
	pageData.Tests = []*TestRunData{}

	pageOffset := (pageArgs.Page - 1) * pageArgs.PageSize
	testInstances, totalTests := fh.coordinator.GetTestHistory(testID, 0, pageOffset, pageArgs.PageSize)

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

	if pageData.CurrentPageIndex > 1 {
		pageData.PrevPageIndex = pageData.CurrentPageIndex - 1
	}

	if pageData.CurrentPageIndex < pageData.TotalPages {
		pageData.NextPageIndex = pageData.CurrentPageIndex + 1
	}

	return pageData, nil
}
