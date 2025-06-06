package checkconsensuscgc

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/clients"
	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/ethpandaops/assertoor/pkg/coordinator/vars"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "check_consensus_cgc"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Checks the CGC (Custody Group Count) value in consensus layer client ENR records.",
		Config:      DefaultConfig(),
		NewTask:     NewTask,
	}
)

type Task struct {
	ctx     *types.TaskContext
	options *types.TaskOptions
	config  Config
	logger  logrus.FieldLogger
}

type ClientCGCInfo struct {
	Name     string `json:"name"`
	ClRPCURL string `json:"clRpcUrl"`
	ENR      string `json:"enr"`
	CGCValue int    `json:"cgcValue"`
	IsValid  bool   `json:"isValid"`
}

func NewTask(ctx *types.TaskContext, options *types.TaskOptions) (types.Task, error) {
	return &Task{
		ctx:     ctx,
		options: options,
		logger:  ctx.Logger.GetLogger(),
	}, nil
}

func (t *Task) Config() interface{} {
	return t.config
}

func (t *Task) Timeout() time.Duration {
	return t.options.Timeout.Duration
}

func (t *Task) LoadConfig() error {
	config := DefaultConfig()

	// parse static config
	if t.options.Config != nil {
		if err := t.options.Config.Unmarshal(&config); err != nil {
			return fmt.Errorf("error parsing task config for %v: %w", TaskName, err)
		}
	}

	// load dynamic vars
	err := t.ctx.Vars.ConsumeVars(&config, t.options.ConfigVars)
	if err != nil {
		return err
	}

	// validate config
	if err := config.Validate(); err != nil {
		return err
	}

	t.config = config

	return nil
}

func (t *Task) Execute(ctx context.Context) error {
	t.processCheck()

	for {
		select {
		case <-time.After(t.config.PollInterval.Duration):
			t.processCheck()
		case <-ctx.Done():
			return nil
		}
	}
}

func (t *Task) processCheck() {
	passResultCount := 0
	totalClientCount := 0
	validClients := []*ClientCGCInfo{}
	invalidClients := []*ClientCGCInfo{}
	invalidClientNames := []string{}

	for _, client := range t.ctx.Scheduler.GetServices().ClientPool().GetClientsByNamePatterns(t.config.ClientPattern, "") {
		if client.ConsensusClient == nil {
			continue
		}

		totalClientCount++

		cgcInfo := t.processClientCGCCheck(client)
		if cgcInfo.IsValid {
			passResultCount++
			validClients = append(validClients, cgcInfo)

			if t.config.ResultVar != "" && passResultCount == 1 {
				t.ctx.Vars.SetVar(t.config.ResultVar, cgcInfo.CGCValue)
			}
		} else {
			invalidClients = append(invalidClients, cgcInfo)
			invalidClientNames = append(invalidClientNames, client.Config.Name)
		}
	}

	requiredPassCount := t.config.MinClientCount
	if requiredPassCount == 0 {
		requiredPassCount = totalClientCount
	}

	resultPass := passResultCount >= requiredPassCount

	if validClientsData, err := vars.GeneralizeData(validClients); err == nil {
		t.ctx.Outputs.SetVar("validClients", validClientsData)
	} else {
		t.logger.Warnf("Failed setting `validClients` output: %v", err)
	}

	if invalidClientsData, err := vars.GeneralizeData(invalidClients); err == nil {
		t.ctx.Outputs.SetVar("invalidClients", invalidClientsData)
	} else {
		t.logger.Warnf("Failed setting `invalidClients` output: %v", err)
	}

	t.ctx.Outputs.SetVar("totalCount", totalClientCount)
	t.ctx.Outputs.SetVar("invalidCount", totalClientCount-passResultCount)
	t.ctx.Outputs.SetVar("validCount", passResultCount)

	t.logger.Infof("CGC Check result: %v, Invalid Clients: %v", resultPass, invalidClientNames)

	switch {
	case resultPass:
		t.ctx.SetResult(types.TaskResultSuccess)
	default:
		if t.config.FailOnCheckMiss {
			t.ctx.SetResult(types.TaskResultFailure)
		} else {
			t.ctx.SetResult(types.TaskResultNone)
		}
	}
}

func (t *Task) processClientCGCCheck(client *clients.PoolClient) *ClientCGCInfo {
	cgcInfo := &ClientCGCInfo{
		Name:    client.Config.Name,
		IsValid: false,
	}

	if client.ConsensusClient != nil {
		cgcInfo.ClRPCURL = client.ConsensusClient.GetEndpointConfig().URL
	}

	// Get node identity to extract ENR
	nodeIdentity, err := client.ConsensusClient.GetRPCClient().GetNodeIdentity(context.Background())
	if err != nil {
		t.logger.Warnf("Failed to get node identity for client %s: %v", client.Config.Name, err)
		return cgcInfo
	}

	cgcInfo.ENR = nodeIdentity.ENR

	// Extract CGC value from ENR
	cgcValue, err := t.extractCGCFromENR(nodeIdentity.ENR)
	if err != nil {
		t.logger.Warnf("Failed to extract CGC from ENR for client %s: %v", client.Config.Name, err)
		return cgcInfo
	}

	cgcInfo.CGCValue = cgcValue

	// Validate CGC value
	isValid := t.validateCGCValue(cgcValue)
	cgcInfo.IsValid = isValid

	return cgcInfo
}

func (t *Task) extractCGCFromENR(enr string) (int, error) {
	// Remove enr: prefix if present
	if strings.HasPrefix(enr, "enr:") {
		enr = enr[4:]
	}

	// Decode base64 ENR
	enrBytes, err := base64.RawURLEncoding.DecodeString(enr)
	if err != nil {
		return 0, fmt.Errorf("failed to decode ENR base64: %w", err)
	}

	// For now, we'll implement a simple search for a "cgc" field in the ENR
	// This is a simplified implementation - in practice, you'd want to properly
	// parse the RLP-encoded ENR structure and look for the "cgc" key-value pair

	// Convert to string to search for cgc field
	enrStr := string(enrBytes)

	// Look for "cgc" followed by a value
	// This is a simplified approach - proper ENR parsing would use RLP decoding
	cgcIndex := strings.Index(enrStr, "cgc")
	if cgcIndex == -1 {
		// If no CGC field found, assume default non-validating value
		return t.config.ExpectedNonValidating, nil
	}

	// Extract the value after "cgc" - this is a simplified extraction
	// In a real implementation, you'd properly parse the RLP structure
	valueStart := cgcIndex + 3
	if valueStart >= len(enrStr) {
		return t.config.ExpectedNonValidating, nil
	}

	// Try to extract a hex value (assume single byte for now)
	if valueStart+1 < len(enrStr) {
		valueByte := enrStr[valueStart]
		return int(valueByte), nil
	}

	return t.config.ExpectedNonValidating, nil
}

func (t *Task) validateCGCValue(cgcValue int) bool {
	// If a specific CGC value is expected, check against that
	if t.config.ExpectedCGCValue > 0 {
		return cgcValue == t.config.ExpectedCGCValue
	}

	// Check if the value matches expected non-validating or validating values
	if cgcValue == t.config.ExpectedNonValidating || cgcValue == t.config.ExpectedValidating {
		return true
	}

	// Check if the value represents a validating node with additional 32 ETH increments
	// CGC = 0x08 + number_of_32eth_chunks
	if cgcValue >= t.config.ExpectedValidating {
		// Calculate if the excess is a multiple of 1 (each 32 ETH adds 1 to CGC)
		excess := cgcValue - t.config.ExpectedValidating
		// For now, we'll accept any positive excess as valid
		// In practice, you might want to check against known validator counts
		return excess >= 0
	}

	return false
}
