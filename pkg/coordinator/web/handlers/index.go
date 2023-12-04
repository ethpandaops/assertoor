package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/ethpandaops/minccino/pkg/coordinator/types"
	"github.com/ethpandaops/minccino/pkg/coordinator/web"
	"github.com/sirupsen/logrus"
)

type IndexPage struct {
	ClientCount  uint64           `json:"client_count"`
	CLReadyCount uint64           `json:"cl_ready_count"`
	CLHeadSlot   uint64           `json:"cl_head_slot"`
	CLHeadRoot   []byte           `json:"cl_head_root"`
	ELReadyCount uint64           `json:"el_ready_count"`
	ELHeadNumber uint64           `json:"el_head_number"`
	ELHeadHash   []byte           `json:"el_head_hash"`
	Tests        []*IndexPageTest `json:"tests"`
}

type IndexPageTest struct {
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

	var templateFiles = append(web.LayoutTemplateFiles,
		"index/index.html",
	)
	var pageTemplate = web.GetTemplate(templateFiles...)
	data := web.InitPageData(w, r, "index", "/", "Index", templateFiles)

	var pageError error
	data.Data, pageError = fh.getIndexPageData()
	if pageError != nil {
		web.HandlePageError(w, r, pageError)
		return
	}
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
		http.Error(w, "Internal server error", http.StatusServiceUnavailable)
	}
}

func (fh *FrontendHandler) getIndexPageData() (*IndexPage, error) {
	pageData := &IndexPage{}

	// client pool status
	clientPool := fh.coordinator.ClientPool()
	allClients := clientPool.GetAllClients()
	pageData.ClientCount = uint64(len(allClients))
	canonicalClFork := clientPool.GetConsensusPool().GetCanonicalFork(2)
	if canonicalClFork != nil {
		pageData.CLReadyCount = uint64(len(canonicalClFork.ReadyClients))
		pageData.CLHeadSlot = uint64(canonicalClFork.Slot)
		pageData.CLHeadRoot = canonicalClFork.Root[:]
	}
	canonicalElFork := clientPool.GetExecutionPool().GetCanonicalFork(2)
	if canonicalElFork != nil {
		pageData.ELReadyCount = uint64(len(canonicalElFork.ReadyClients))
		pageData.ELHeadNumber = uint64(canonicalElFork.Number)
		pageData.ELHeadHash = canonicalElFork.Hash[:]
	}

	// tasks list
	pageData.Tests = []*IndexPageTest{}
	for idx, test := range fh.coordinator.GetTests() {
		testData := &IndexPageTest{
			Index:       uint64(idx),
			Name:        test.Name(),
			IsStarted:   test.Status() != types.TestStatusPending,
			IsCompleted: test.Status() > types.TestStatusRunning,
			StartTime:   test.StartTime(),
			StopTime:    test.StopTime(),
			Timeout:     test.Timeout(),
			HasTimeout:  test.Timeout() > 0,
			TaskCount:   uint64(test.GetTaskScheduler().GetTaskCount()),
		}
		if testData.IsCompleted {
			testData.RunTime = testData.StopTime.Sub(testData.StartTime)
			testData.HasRunTime = true
		} else if testData.IsStarted {
			testData.RunTime = time.Since(testData.StartTime)
			testData.HasRunTime = true
		}
		switch test.Status() {
		case types.TestStatusPending:
			testData.Status = "pending"
		case types.TestStatusRunning:
			testData.Status = "running"
		case types.TestStatusSuccess:
			testData.Status = "success"
		case types.TestStatusFailure:
			testData.Status = "failure"
		}
		pageData.Tests = append(pageData.Tests, testData)
	}

	return pageData, nil
}
