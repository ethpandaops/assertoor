package handlers

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/db"
	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/sirupsen/logrus"
)

type RegistryPageArgs struct {
	PageSize uint64 `json:"ps"`
	Page     uint64 `json:"p"`
}

type RegistryPage struct {
	CanRegister bool `json:"can_register"`
	CanStart    bool `json:"can_start"`
	CanDelete   bool `json:"can_delete"`

	Tests          []*TestRegistryData `json:"tests"`
	TotalTests     uint64              `json:"total_tests"`
	FirstTestIndex uint64              `json:"first_test_index"`
	LastTestIndex  uint64              `json:"last_test_index"`

	TotalPages       uint64 `json:"total_pages"`
	PageSize         uint64 `json:"page_size"`
	CurrentPageIndex uint64 `json:"current_page_index"`
	PrevPageIndex    uint64 `json:"prev_page_index"`
	NextPageIndex    uint64 `json:"next_page_index"`
	LastPageRegistry uint64 `json:"last_page_index"`
}

type TestRegistryData struct {
	Index    uint64     `json:"index"`
	TestID   string     `json:"test_id"`
	Name     string     `json:"name"`
	Source   string     `json:"source"`
	Error    string     `json:"error"`
	Config   string     `json:"config"`
	RunCount int        `json:"run_count"`
	LastRun  *time.Time `json:"last_run"`
}

func (fh *FrontendHandler) parseRegistryPageArgs(r *http.Request) *RegistryPageArgs {
	urlArgs := r.URL.Query()
	pageArgs := &RegistryPageArgs{
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

// Registry will return the "registry" page using a go template
func (fh *FrontendHandler) Registry(w http.ResponseWriter, r *http.Request) {
	urlArgs := r.URL.Query()
	if urlArgs.Has("json") {
		fh.RegistryData(w, r)
		return
	}

	templateFiles := LayoutTemplateFiles
	templateFiles = append(templateFiles,
		"registry/registry.html",
		"sidebar/sidebar.html",
	)
	pageTemplate := fh.templates.GetTemplate(templateFiles...)
	data := fh.initPageData(r, "registry", "/", "Registry", templateFiles)
	pageArgs := fh.parseRegistryPageArgs(r)

	var pageError error
	data.Data, pageError = fh.getRegistryPageData(pageArgs)

	if pageError != nil {
		fh.HandlePageError(w, r, pageError)
		return
	}

	data.ShowSidebar = true
	data.SidebarData = fh.getSidebarData("*registry")

	w.Header().Set("Content-Type", "text/html")

	if fh.handleTemplateError(w, r, "registry.go", "Registry", pageTemplate.ExecuteTemplate(w, "layout", data)) != nil {
		return // an error has occurred and was processed
	}
}

func (fh *FrontendHandler) RegistryData(w http.ResponseWriter, r *http.Request) {
	var pageData *RegistryPage

	var pageError error

	pageArgs := fh.parseRegistryPageArgs(r)
	pageData, pageError = fh.getRegistryPageData(pageArgs)

	if pageError != nil {
		fh.HandlePageError(w, r, pageError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	err := json.NewEncoder(w).Encode(pageData)
	if err != nil {
		logrus.WithError(err).Error("error encoding registry data")

		//nolint:gocritic // ignore
		http.Error(w, "Internal server error", http.StatusServiceUnavailable)
	}
}

func (fh *FrontendHandler) getRegistryPageData(pageArgs *RegistryPageArgs) (*RegistryPage, error) {
	pageData := &RegistryPage{
		CanRegister: !fh.securityTrimmed,
		CanStart:    !fh.securityTrimmed,
		CanDelete:   !fh.securityTrimmed,
	}

	testDescriptors := fh.coordinator.TestRegistry().GetTestDescriptors()

	pageOffset := (pageArgs.Page - 1) * pageArgs.PageSize
	pageLimit := pageOffset + pageArgs.PageSize

	if pageLimit > uint64(len(testDescriptors)) {
		pageLimit = uint64(len(testDescriptors))
	}

	if pageOffset > uint64(len(testDescriptors)) {
		pageOffset = uint64(len(testDescriptors))
	}

	stats, err := fh.coordinator.Database().GetTestRunStats()
	if err != nil {
		return nil, err
	}

	runStatsMap := make(map[string]*db.TestRunStats)
	for _, stat := range stats {
		runStatsMap[stat.TestID] = stat
	}

	for idx, test := range testDescriptors[pageOffset:pageLimit] {
		runStats := runStatsMap[test.ID()]
		pageData.Tests = append(pageData.Tests, fh.getTestRegistryData(idx, test, runStats))
	}

	pageData.TotalTests = uint64(len(testDescriptors))
	pageData.TotalPages = uint64(math.Ceil(float64(pageData.TotalTests) / float64(pageArgs.PageSize)))
	pageData.CurrentPageIndex = pageArgs.Page
	pageData.PageSize = pageArgs.PageSize
	pageData.FirstTestIndex = (pageArgs.Page - 1) * pageArgs.PageSize
	pageData.LastTestIndex = pageData.FirstTestIndex + uint64(len(pageData.Tests)) - 1
	pageData.LastPageRegistry = pageData.TotalPages - 1

	if pageData.CurrentPageIndex > 1 {
		pageData.PrevPageIndex = pageData.CurrentPageIndex - 1
	}

	if pageData.CurrentPageIndex < pageData.TotalPages {
		pageData.NextPageIndex = pageData.CurrentPageIndex + 1
	}

	return pageData, nil
}

func (fh *FrontendHandler) getTestRegistryData(idx int, test types.TestDescriptor, runStats *db.TestRunStats) *TestRegistryData {
	testData := &TestRegistryData{
		Index:  uint64(idx),
		TestID: test.ID(),
		Source: test.Source(),
		Config: "null",
	}

	if testError := test.Err(); testError != nil {
		testData.Error = testError.Error()
	}

	if testConfig := test.Config(); testConfig != nil {
		testData.Name = testConfig.Name
	}

	if !fh.securityTrimmed && test.Vars() != nil {
		configJSON, err := json.Marshal(test.Vars().GetVarsMap(nil, true))
		if err == nil {
			testData.Config = string(configJSON)
		}
	}

	if runStats != nil {
		testData.RunCount = runStats.Count

		if runStats.LastRun > 0 {
			lastRun := time.UnixMilli(int64(runStats.LastRun))
			testData.LastRun = &lastRun
		}
	}

	return testData
}
