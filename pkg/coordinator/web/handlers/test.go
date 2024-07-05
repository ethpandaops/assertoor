package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/ethpandaops/assertoor/pkg/coordinator/web"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type TestPage struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Source string `json:"source"`
	Config string `json:"config"`

	Tests []*TestRunData `json:"tests"`
}

// Index will return the "index" page using a go template
func (fh *FrontendHandler) TestPage(w http.ResponseWriter, r *http.Request) {
	urlArgs := r.URL.Query()
	if urlArgs.Has("json") {
		fh.IndexData(w, r)
		return
	}

	templateFiles := web.LayoutTemplateFiles
	templateFiles = append(templateFiles,
		"test/test.html",
		"sidebar/sidebar.html",
		"test/test_runs.html",
	)
	pageTemplate := web.GetTemplate(templateFiles...)
	vars := mux.Vars(r)
	data := web.InitPageData(r, "test", "/", "Test", templateFiles)

	var pageError error

	var pageData *TestPage

	pageData, pageError = fh.getTestPageData(vars["testId"])
	if pageError != nil {
		web.HandlePageError(w, r, pageError)
		return
	}

	data.Data = pageData
	data.ShowSidebar = true
	data.SidebarData = fh.getSidebarData(pageData.ID)

	w.Header().Set("Content-Type", "text/html")

	if web.HandleTemplateError(w, r, "index.go", "Index", "", pageTemplate.ExecuteTemplate(w, "layout", data)) != nil {
		return // an error has occurred and was processed
	}
}

func (fh *FrontendHandler) TestPageData(w http.ResponseWriter, r *http.Request) {
	var pageData *TestPage

	var pageError error

	vars := mux.Vars(r)
	pageData, pageError = fh.getTestPageData(vars["testId"])

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

func (fh *FrontendHandler) getTestPageData(testID string) (*TestPage, error) {
	var testDescriptor types.TestDescriptor

	for _, testDescr := range fh.coordinator.GetTestDescriptors() {
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

	testCfgJSON, err := json.Marshal(testDescriptor.Vars().GetVarsMap(nil, false))
	if err != nil {
		return nil, fmt.Errorf("failed marshalling test vars: %v", err)
	}

	pageData.Config = string(testCfgJSON)

	// test runs
	pageData.Tests = []*TestRunData{}

	testInstances := append(fh.coordinator.GetTestHistory(), fh.coordinator.GetTestQueue()...)
	for idx, test := range testInstances {
		if test.TestID() != testID {
			continue
		}

		pageData.Tests = append(pageData.Tests, fh.getTestRunData(idx, test))
	}

	return pageData, nil
}
