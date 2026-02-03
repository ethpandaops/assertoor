package api

import (
	"encoding/hex"
	"net/http"
	"time"

	"github.com/ethpandaops/assertoor/pkg/clients/consensus"
	"github.com/ethpandaops/assertoor/pkg/clients/execution"
)

type GetClientsResponse struct {
	Clients     []*ClientResponse `json:"clients"`
	ClientCount int               `json:"client_count"`
}

type ClientResponse struct {
	Index         int    `json:"index"`
	Name          string `json:"name"`
	CLVersion     string `json:"cl_version"`
	CLType        int64  `json:"cl_type"`
	CLHeadSlot    uint64 `json:"cl_head_slot"`
	CLHeadRoot    string `json:"cl_head_root"`
	CLStatus      string `json:"cl_status"`
	CLLastRefresh string `json:"cl_refresh"`
	CLLastError   string `json:"cl_error"`
	CLIsReady     bool   `json:"cl_ready"`
	ELVersion     string `json:"el_version"`
	ELType        int64  `json:"el_type"`
	ELHeadNumber  uint64 `json:"el_head_number"`
	ELHeadHash    string `json:"el_head_hash"`
	ELStatus      string `json:"el_status"`
	ELLastRefresh string `json:"el_refresh"`
	ELLastError   string `json:"el_error"`
	ELIsReady     bool   `json:"el_ready"`
}

// GetClients godoc
// @Id getClients
// @Summary Get list of configured clients
// @Tags Client
// @Description Returns the list of configured consensus and execution layer clients with their current status.
// @Produce  json
// @Success 200 {object} Response{data=GetClientsResponse} "Success"
// @Failure 400 {object} Response "Failure"
// @Failure 500 {object} Response "Server Error"
// @Router /api/v1/clients [get]
func (ah *APIHandler) GetClients(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", contentTypeJSON)

	clientPool := ah.coordinator.ClientPool()
	clients := make([]*ClientResponse, 0, len(clientPool.GetAllClients()))

	for _, client := range clientPool.GetAllClients() {
		headSlot, headRoot := client.ConsensusClient.GetLastHead()
		blockNum, blockHash := client.ExecutionClient.GetLastHead()

		clientData := &ClientResponse{
			Index:         int(client.ConsensusClient.GetIndex()),
			Name:          client.ConsensusClient.GetName(),
			CLVersion:     client.ConsensusClient.GetVersion(),
			CLType:        int64(client.ConsensusClient.GetClientType()),
			CLHeadSlot:    uint64(headSlot),
			CLHeadRoot:    "0x" + hex.EncodeToString(headRoot[:]),
			CLLastRefresh: client.ConsensusClient.GetLastEventTime().Format(time.RFC3339),
			CLIsReady:     clientPool.GetConsensusPool().GetCanonicalFork(2).IsClientReady(client.ConsensusClient),
			ELVersion:     client.ExecutionClient.GetVersion(),
			ELType:        int64(client.ExecutionClient.GetClientType()),
			ELHeadNumber:  blockNum,
			ELHeadHash:    "0x" + hex.EncodeToString(blockHash[:]),
			ELLastRefresh: client.ExecutionClient.GetLastEventTime().Format(time.RFC3339),
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

		clients = append(clients, clientData)
	}

	response := &GetClientsResponse{
		Clients:     clients,
		ClientCount: len(clients),
	}

	ah.sendOKResponse(w, r.URL.String(), response)
}
