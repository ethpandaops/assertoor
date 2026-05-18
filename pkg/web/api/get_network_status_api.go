package api

import (
	"encoding/hex"
	"net/http"

	"github.com/ethpandaops/assertoor/pkg/clients/consensus"
	"github.com/ethpandaops/assertoor/pkg/clients/execution"
	"github.com/ethpandaops/assertoor/pkg/types"
)

// NetworkStatusResponse describes the network-wide snapshot rendered
// by the dashboard's `network_status` tile. Reflects the aggregated
// state across all configured clients (canonical head, finalized
// checkpoint, queue size, etc.). Cheap to compute — no extra RPC
// calls; everything is served from the in-process pool/cache.
type NetworkStatusResponse struct {
	ChainID         uint64 `json:"chain_id"`
	NetworkName     string `json:"network_name"`
	GenesisTime     int64  `json:"genesis_time"`
	SlotDurationMs  uint64 `json:"slot_duration_ms"`
	SlotsPerEpoch   uint64 `json:"slots_per_epoch"`
	CurrentSlot     uint64 `json:"current_slot"`
	CurrentEpoch    uint64 `json:"current_epoch"`
	HeadSlot        uint64 `json:"head_slot"`
	HeadRoot        string `json:"head_root"`
	FinalizedEpoch  uint64 `json:"finalized_epoch"`
	FinalizedRoot   string `json:"finalized_root"`
	JustifiedEpoch  uint64 `json:"justified_epoch"`
	JustifiedRoot   string `json:"justified_root"`
	ClientCount     int    `json:"client_count"`
	CLReadyCount    int    `json:"cl_ready_count"`
	ELReadyCount    int    `json:"el_ready_count"`
	ELHeadNumber    uint64 `json:"el_head_number"`
	ELHeadHash      string `json:"el_head_hash"`
	TestsRunning    int    `json:"tests_running"`
	TestsQueued     int    `json:"tests_queued"`
}

// GetNetworkStatus godoc
// @Id getNetworkStatus
// @Summary Get a network-wide status snapshot
// @Tags Network
// @Description Aggregated snapshot of the chain (head, finalized,
// @Description justified, current slot/epoch) and the orchestration
// @Description state (client readiness counts, test queue depth).
// @Description Powers the dashboard `network_status` tile.
// @Produce json
// @Success 200 {object} Response{data=NetworkStatusResponse} "Success"
// @Failure 500 {object} Response "Server Error"
// @Router /api/v1/network_status [get]
func (ah *APIHandler) GetNetworkStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", contentTypeJSON)

	resp := &NetworkStatusResponse{}

	clientPool := ah.coordinator.ClientPool()
	consensusPool := clientPool.GetConsensusPool()
	cache := consensusPool.GetBlockCache()

	if specs := cache.GetSpecs(); specs != nil {
		resp.NetworkName = specs.ConfigName
		resp.SlotDurationMs = specs.SlotDurationMs
		resp.SlotsPerEpoch = specs.SlotsPerEpoch
	}

	if genesis := cache.GetGenesis(); genesis != nil {
		resp.GenesisTime = genesis.GenesisTime.Unix()
	}

	if wallclock := cache.GetWallclock(); wallclock != nil {
		s, e, _ := wallclock.Now()
		resp.CurrentSlot = s.Number()
		resp.CurrentEpoch = e.Number()
	}

	finEpoch, finRoot := cache.GetFinalizedCheckpoint()
	resp.FinalizedEpoch = uint64(finEpoch)
	resp.FinalizedRoot = "0x" + hex.EncodeToString(finRoot[:])

	// The current canonical fork carries the agreed head slot/root
	// across all clients that have caught up.
	if fork := consensusPool.GetCanonicalFork(-1); fork != nil {
		resp.HeadSlot = uint64(fork.Slot)
		resp.HeadRoot = "0x" + hex.EncodeToString(fork.Root[:])
	}

	allClients := clientPool.GetAllClients()
	resp.ClientCount = len(allClients)

	var elHead uint64

	var elHash []byte

	for _, c := range allClients {
		if c.ConsensusClient.GetStatus() == consensus.ClientStatusOnline {
			resp.CLReadyCount++
		}

		if c.ExecutionClient.GetStatus() == execution.ClientStatusOnline {
			resp.ELReadyCount++

			if num, hash := c.ExecutionClient.GetLastHead(); num > elHead {
				elHead = num
				elHash = hash[:]
			}
		}
	}

	resp.ELHeadNumber = elHead

	if len(elHash) > 0 {
		resp.ELHeadHash = "0x" + hex.EncodeToString(elHash)
	}

	for _, t := range ah.coordinator.GetTestQueue() {
		switch t.Status() {
		case types.TestStatusRunning:
			resp.TestsRunning++
		case types.TestStatusPending:
			resp.TestsQueued++
		}
	}

	// Chain ID is sourced from the EL pool's shared block cache.
	if elPool := clientPool.GetExecutionPool(); elPool != nil {
		if elCache := elPool.GetBlockCache(); elCache != nil {
			if chainID := elCache.GetChainID(); chainID != nil {
				resp.ChainID = chainID.Uint64()
			}
		}
	}

	ah.sendOKResponse(w, r.URL.String(), resp)
}
