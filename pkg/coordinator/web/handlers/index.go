package handlers

import (
	"net/http"

	"github.com/ethpandaops/minccino/pkg/coordinator/web"
)

type IndexPage struct {
	ClientCount  uint64 `json:"client_count"`
	CLReadyCount uint64 `json:"cl_ready_count"`
	CLHeadSlot   uint64 `json:"cl_head_slot"`
	CLHeadRoot   []byte `json:"cl_head_root"`
	ELReadyCount uint64 `json:"el_ready_count"`
	ELHeadNumber uint64 `json:"el_head_number"`
	ELHeadHash   []byte `json:"el_head_hash"`
}

// Index will return the "index" page using a go template
func (fh *FrontendHandler) Index(w http.ResponseWriter, r *http.Request) {
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

func (fh *FrontendHandler) getIndexPageData() (*IndexPage, error) {
	pageData := &IndexPage{}

	allClients := fh.pool.GetAllClients()
	pageData.ClientCount = uint64(len(allClients))

	canonicalClFork := fh.pool.GetConsensusPool().GetCanonicalFork(2)
	if canonicalClFork != nil {
		pageData.CLReadyCount = uint64(len(canonicalClFork.ReadyClients))
		pageData.CLHeadSlot = uint64(canonicalClFork.Slot)
		pageData.CLHeadRoot = canonicalClFork.Root[:]
	}
	canonicalElFork := fh.pool.GetExecutionPool().GetCanonicalFork(2)
	if canonicalElFork != nil {
		pageData.ELReadyCount = uint64(len(canonicalElFork.ReadyClients))
		pageData.ELHeadNumber = uint64(canonicalElFork.Number)
		pageData.ELHeadHash = canonicalElFork.Hash[:]
	}

	return pageData, nil
}
