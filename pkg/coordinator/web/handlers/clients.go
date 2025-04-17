package handlers

import (
	"net/http"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/clients"
	"github.com/ethpandaops/assertoor/pkg/coordinator/clients/consensus"
	"github.com/ethpandaops/assertoor/pkg/coordinator/clients/execution"
)

type ClientsPage struct {
	Clients     []*ClientsPageClient `json:"clients"`
	ClientCount uint64               `json:"client_count"`
}

type ClientsPageClient struct {
	Index         int       `json:"index"`
	Name          string    `json:"name"`
	CLVersion     string    `json:"cl_version"`
	CLType        int64     `json:"cl_type"`
	CLHeadSlot    uint64    `json:"cl_head_slot"`
	CLHeadRoot    []byte    `json:"cl_head_root"`
	CLStatus      string    `json:"cl_status"`
	CLLastRefresh time.Time `json:"cl_refresh"`
	CLLastError   string    `json:"cl_error"`
	CLIsReady     bool      `json:"cl_ready"`
	ELVersion     string    `json:"el_version"`
	ELType        int64     `json:"el_type"`
	ELHeadNumber  uint64    `json:"el_head_number"`
	ELHeadHash    []byte    `json:"el_head_hash"`
	ELStatus      string    `json:"el_status"`
	ELLastRefresh time.Time `json:"el_refresh"`
	ELLastError   string    `json:"el_error"`
	ELIsReady     bool      `json:"el_ready"`
}

// Clients will return the "clients" page using a go template
func (fh *FrontendHandler) Clients(w http.ResponseWriter, r *http.Request) {
	templateFiles := LayoutTemplateFiles
	templateFiles = append(templateFiles,
		"clients/clients.html",
	)

	pageTemplate := fh.templates.GetTemplate(templateFiles...)
	data := fh.initPageData(r, "clients", "/clients", "Clients", templateFiles)

	var pageError error
	data.Data, pageError = fh.getClientsPageData()

	if pageError != nil {
		fh.HandlePageError(w, r, pageError)
		return
	}

	w.Header().Set("Content-Type", "text/html")

	if fh.handleTemplateError(w, r, "clients.go", "Clients", pageTemplate.ExecuteTemplate(w, "layout", data)) != nil {
		return // an error has occurred and was processed
	}
}

//nolint:unparam // ignore
func (fh *FrontendHandler) getClientsPageData() (*ClientsPage, error) {
	pageData := &ClientsPage{
		Clients: []*ClientsPageClient{},
	}

	// get clients
	for _, client := range fh.coordinator.ClientPool().GetAllClients() {
		clientData := fh.getClientsPageClientData(client)
		pageData.Clients = append(pageData.Clients, clientData)
	}

	pageData.ClientCount = uint64(len(pageData.Clients))

	return pageData, nil
}

func (fh *FrontendHandler) getClientsPageClientData(client *clients.PoolClient) *ClientsPageClient {
	clientPool := fh.coordinator.ClientPool()
	headSlot, headRoot := client.ConsensusClient.GetLastHead()
	blockNum, blockHash := client.ExecutionClient.GetLastHead()
	clientData := &ClientsPageClient{
		Index:         int(client.ConsensusClient.GetIndex()),
		Name:          client.ConsensusClient.GetName(),
		CLVersion:     client.ConsensusClient.GetVersion(),
		CLType:        int64(client.ConsensusClient.GetClientType()),
		CLHeadSlot:    uint64(headSlot),
		CLHeadRoot:    headRoot[:],
		CLLastRefresh: client.ConsensusClient.GetLastEventTime(),
		CLIsReady:     clientPool.GetConsensusPool().GetCanonicalFork(2).IsClientReady(client.ConsensusClient),
		ELVersion:     client.ExecutionClient.GetVersion(),
		ELType:        int64(client.ExecutionClient.GetClientType()),
		ELHeadNumber:  blockNum,
		ELHeadHash:    blockHash[:],
		ELLastRefresh: client.ExecutionClient.GetLastEventTime(),
		ELIsReady:     clientPool.GetExecutionPool().GetCanonicalFork(2).IsClientReady(client.ExecutionClient),
	}

	if lastError := client.ConsensusClient.GetLastError(); lastError != nil {
		clientData.CLLastError = lastError.Error()
	}

	switch client.ConsensusClient.GetStatus() {
	case consensus.ClientStatusOffline:
		clientData.CLStatus = "offline"
	case consensus.ClientStatusOnline:
		clientData.CLStatus = "online"
	case consensus.ClientStatusOptimistic:
		clientData.CLStatus = "optimistic"
	case consensus.ClientStatusSynchronizing:
		clientData.CLStatus = "synchronizing"
	}

	if lastError := client.ExecutionClient.GetLastError(); lastError != nil {
		clientData.ELLastError = lastError.Error()
	}

	switch client.ExecutionClient.GetStatus() {
	case execution.ClientStatusOffline:
		clientData.ELStatus = "offline"
	case execution.ClientStatusOnline:
		clientData.ELStatus = "online"
	case execution.ClientStatusSynchronizing:
		clientData.ELStatus = "synchronizing"
	}

	return clientData
}
