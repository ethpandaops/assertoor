package checkconsensusidentity

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethpandaops/assertoor/pkg/clients"
	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/ethpandaops/assertoor/pkg/vars"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "check_consensus_identity"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Checks consensus client node identity information including CGC extraction from ENR.",
		Category:    "consensus",
		Config:      DefaultConfig(),
		Outputs: []types.TaskOutputDefinition{
			{
				Name:        "matchingClients",
				Type:        "array",
				Description: "List of clients that passed identity checks.",
			},
			{
				Name:        "failedClients",
				Type:        "array",
				Description: "List of clients that failed identity checks.",
			},
			{
				Name:        "totalCount",
				Type:        "int",
				Description: "Total number of clients checked.",
			},
			{
				Name:        "matchingCount",
				Type:        "int",
				Description: "Number of clients that passed checks.",
			},
			{
				Name:        "failedCount",
				Type:        "int",
				Description: "Number of clients that failed checks.",
			},
		},
		NewTask: NewTask,
	}
)

type Task struct {
	ctx     *types.TaskContext
	options *types.TaskOptions
	config  Config
	logger  logrus.FieldLogger
}

type IdentityCheckResult struct {
	ClientName         string                 `json:"clientName"`
	PeerID             string                 `json:"peerId"`
	ENR                string                 `json:"enr"`
	P2PAddresses       []string               `json:"p2pAddresses"`
	DiscoveryAddresses []string               `json:"discoveryAddresses"`
	SeqNumber          uint64                 `json:"seqNumber"`
	Attnets            string                 `json:"attnets"`
	Syncnets           string                 `json:"syncnets"`
	CGC                uint64                 `json:"cgc"`
	ENRFields          map[string]interface{} `json:"enrFields"`
	ChecksPassed       bool                   `json:"checksPassed"`
	FailureReasons     []string               `json:"failureReasons"`
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

	if t.options.Config != nil {
		if err := t.options.Config.Unmarshal(&config); err != nil {
			return fmt.Errorf("error parsing task config for %v: %w", TaskName, err)
		}
	}

	err := t.ctx.Vars.ConsumeVars(&config, t.options.ConfigVars)
	if err != nil {
		return err
	}

	if err := config.Validate(); err != nil {
		return err
	}

	t.config = config

	return nil
}

func (t *Task) Execute(ctx context.Context) error {
	checkCount := 0

	for {
		checkCount++

		if done, err := t.processCheck(checkCount); done {
			return err
		}

		select {
		case <-time.After(t.config.PollInterval.Duration):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (t *Task) processCheck(checkCount int) (bool, error) {
	passResultCount := 0
	totalClientCount := 0
	matchingClients := []*IdentityCheckResult{}
	failedClients := []*IdentityCheckResult{}
	failedClientNames := []string{}

	t.logger.Infof("Starting identity check for pattern: %s", t.config.ClientPattern)

	for _, client := range t.ctx.Scheduler.GetServices().ClientPool().GetClientsByNamePatterns(t.config.ClientPattern, "") {
		if client.ConsensusClient == nil {
			t.logger.Warnf("Client %s has no consensus client, skipping", client.Config.Name)
			continue
		}

		totalClientCount++

		t.logger.Infof("Checking identity for client: %s", client.Config.Name)

		result := t.checkClientIdentity(client)

		// Debug output for each client
		t.logger.Infof("Client %s identity check result:", result.ClientName)
		t.logger.Infof("  PeerID: %s", result.PeerID)
		t.logger.Infof("  ENR: %s", result.ENR)
		t.logger.Infof("  CGC: %d", result.CGC)
		t.logger.Infof("  P2P Addresses: %v", result.P2PAddresses)
		t.logger.Infof("  Discovery Addresses: %v", result.DiscoveryAddresses)
		t.logger.Infof("  Sequence Number: %d", result.SeqNumber)
		t.logger.Infof("  Checks Passed: %v", result.ChecksPassed)

		if len(result.FailureReasons) > 0 {
			t.logger.Infof("  Failure Reasons: %v", result.FailureReasons)
		}

		if result.ChecksPassed {
			passResultCount++

			matchingClients = append(matchingClients, result)
			t.logger.Infof("✅ Client %s passed all checks", result.ClientName)
		} else {
			failedClients = append(failedClients, result)
			failedClientNames = append(failedClientNames, result.ClientName)
			t.logger.Warnf("❌ Client %s failed checks: %v", result.ClientName, result.FailureReasons)
		}
	}

	requiredPassCount := t.config.MinClientCount
	if requiredPassCount == 0 {
		requiredPassCount = totalClientCount
	}

	resultPass := passResultCount >= requiredPassCount

	// Set output variables using context.Outputs
	if matchingClientsData, err := vars.GeneralizeData(matchingClients); err == nil {
		t.ctx.Outputs.SetVar("matchingClients", matchingClientsData)
	} else {
		t.logger.Warnf("Failed setting `matchingClients` output: %v", err)
	}

	if failedClientsData, err := vars.GeneralizeData(failedClients); err == nil {
		t.ctx.Outputs.SetVar("failedClients", failedClientsData)
	} else {
		t.logger.Warnf("Failed setting `failedClients` output: %v", err)
	}

	t.ctx.Outputs.SetVar("totalCount", totalClientCount)
	t.ctx.Outputs.SetVar("matchingCount", passResultCount)
	t.ctx.Outputs.SetVar("failedCount", totalClientCount-passResultCount)

	// Enhanced summary logging
	t.logger.Infof("📊 Identity check summary:")
	t.logger.Infof("  Total clients: %d", totalClientCount)
	t.logger.Infof("  Passed: %d", passResultCount)
	t.logger.Infof("  Failed: %d", totalClientCount-passResultCount)
	t.logger.Infof("  Required pass count: %d", requiredPassCount)
	t.logger.Infof("  Overall result: %v", resultPass)

	if len(failedClientNames) > 0 {
		t.logger.Infof("  Failed clients: %v", failedClientNames)
	}

	// Set task result - default to pending instead of failure unless explicitly configured
	switch {
	case t.config.MaxFailCount > -1 && len(failedClients) > t.config.MaxFailCount:
		if t.config.FailOnCheckMiss {
			t.logger.Infof("Setting result to FAILURE (too many failures: %d > %d)", len(failedClients), t.config.MaxFailCount)
			t.ctx.SetResult(types.TaskResultFailure)
			t.ctx.ReportProgress(0, fmt.Sprintf("Too many failures: %d (attempt %d)", len(failedClients), checkCount))

			return true, fmt.Errorf("too many identity check failures: %d", len(failedClients))
		}

		t.logger.Infof("Setting result to PENDING (too many failures but failOnCheckMiss=false)")
		t.ctx.SetResult(types.TaskResultNone)
		t.ctx.ReportProgress(0, fmt.Sprintf("Waiting for identity check... %d/%d (attempt %d)", passResultCount, requiredPassCount, checkCount))

		return false, nil
	case resultPass:
		t.logger.Infof("Setting result to SUCCESS (requirements met)")
		t.ctx.SetResult(types.TaskResultSuccess)
		t.ctx.ReportProgress(100, fmt.Sprintf("Identity check passed: %d/%d clients", passResultCount, totalClientCount))

		if !t.config.ContinueOnPass {
			return true, nil
		}

		return false, nil
	default:
		if t.config.FailOnCheckMiss {
			t.logger.Infof("Setting result to FAILURE (requirements not met and failOnCheckMiss=true)")
			t.ctx.SetResult(types.TaskResultFailure)
			t.ctx.ReportProgress(0, fmt.Sprintf("Identity check failed: %d/%d (attempt %d)", passResultCount, requiredPassCount, checkCount))

			return true, fmt.Errorf("identity check failed: %d/%d", passResultCount, requiredPassCount)
		}

		t.logger.Infof("Setting result to PENDING (requirements not met but failOnCheckMiss=false)")
		t.ctx.SetResult(types.TaskResultNone)
		t.ctx.ReportProgress(0, fmt.Sprintf("Waiting for identity check... %d/%d (attempt %d)", passResultCount, requiredPassCount, checkCount))

		return false, nil
	}
}

func (t *Task) checkClientIdentity(client *clients.PoolClient) *IdentityCheckResult {
	result := &IdentityCheckResult{
		ClientName:     client.Config.Name,
		ChecksPassed:   true,
		FailureReasons: []string{},
	}

	t.logger.Debugf("Getting node identity for client %s", client.Config.Name)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	identity, err := client.ConsensusClient.GetRPCClient().GetNodeIdentity(ctx)
	if err != nil {
		t.logger.Errorf("Failed to get node identity for client %s: %v", client.Config.Name, err)

		result.ChecksPassed = false
		result.FailureReasons = append(result.FailureReasons, fmt.Sprintf("Failed to get node identity: %v", err))

		return result
	}

	t.logger.Debugf("Retrieved node identity for client %s: PeerID=%s, ENR=%s",
		client.Config.Name, identity.PeerID, identity.ENR)

	result.PeerID = identity.PeerID
	result.ENR = identity.ENR
	result.P2PAddresses = identity.P2PAddresses
	result.DiscoveryAddresses = identity.DiscoveryAddresses
	result.SeqNumber = identity.Metadata.SeqNumber
	result.Attnets = identity.Metadata.Attnets
	result.Syncnets = identity.Metadata.Syncnets

	// Extract CGC from ENR
	t.logger.Debugf("Extracting CGC from ENR for client %s", client.Config.Name)

	cgc, enrFields, err := t.extractCGCFromENR(identity.ENR)
	if err != nil {
		t.logger.Errorf("Failed to parse ENR for client %s: %v", client.Config.Name, err)

		result.ChecksPassed = false
		result.FailureReasons = append(result.FailureReasons, fmt.Sprintf("Failed to parse ENR: %v", err))

		return result
	}

	t.logger.Debugf("Extracted CGC=%d for client %s", cgc, client.Config.Name)

	result.CGC = cgc
	result.ENRFields = enrFields

	// Perform configured checks
	t.logger.Debugf("Performing checks for client %s", client.Config.Name)
	t.performChecks(result)

	return result
}

func (t *Task) performChecks(result *IdentityCheckResult) {
	// Check CGC expectations
	if t.config.ExpectCGC != nil && result.CGC != *t.config.ExpectCGC {
		result.ChecksPassed = false
		result.FailureReasons = append(result.FailureReasons,
			fmt.Sprintf("Expected CGC %d, got %d", *t.config.ExpectCGC, result.CGC))
	}

	if t.config.MinCGC != nil && result.CGC < *t.config.MinCGC {
		result.ChecksPassed = false
		result.FailureReasons = append(result.FailureReasons,
			fmt.Sprintf("CGC %d is below minimum %d", result.CGC, *t.config.MinCGC))
	}

	if t.config.MaxCGC != nil && result.CGC > *t.config.MaxCGC {
		result.ChecksPassed = false
		result.FailureReasons = append(result.FailureReasons,
			fmt.Sprintf("CGC %d is above maximum %d", result.CGC, *t.config.MaxCGC))
	}

	// Check PeerID pattern
	if t.config.ExpectPeerIDPattern != "" {
		matched, err := regexp.MatchString(t.config.ExpectPeerIDPattern, result.PeerID)
		if err != nil {
			result.ChecksPassed = false
			result.FailureReasons = append(result.FailureReasons,
				fmt.Sprintf("Invalid PeerID pattern: %v", err))
		} else if !matched {
			result.ChecksPassed = false
			result.FailureReasons = append(result.FailureReasons,
				fmt.Sprintf("PeerID %s does not match pattern %s", result.PeerID, t.config.ExpectPeerIDPattern))
		}
	}

	// Check P2P address count
	if t.config.ExpectP2PAddressCount != nil && len(result.P2PAddresses) != *t.config.ExpectP2PAddressCount {
		result.ChecksPassed = false
		result.FailureReasons = append(result.FailureReasons,
			fmt.Sprintf("Expected %d P2P addresses, got %d", *t.config.ExpectP2PAddressCount, len(result.P2PAddresses)))
	}

	// Check P2P address match
	if t.config.ExpectP2PAddressMatch != "" {
		found := false

		for _, addr := range result.P2PAddresses {
			if matched, _ := regexp.MatchString(t.config.ExpectP2PAddressMatch, addr); matched {
				found = true
				break
			}
		}

		if !found {
			result.ChecksPassed = false
			result.FailureReasons = append(result.FailureReasons,
				fmt.Sprintf("No P2P address matches pattern %s", t.config.ExpectP2PAddressMatch))
		}
	}

	// Check sequence number
	if t.config.ExpectSeqNumber != nil && result.SeqNumber != *t.config.ExpectSeqNumber {
		result.ChecksPassed = false
		result.FailureReasons = append(result.FailureReasons,
			fmt.Sprintf("Expected sequence number %d, got %d", *t.config.ExpectSeqNumber, result.SeqNumber))
	}

	if t.config.MinSeqNumber != nil && result.SeqNumber < *t.config.MinSeqNumber {
		result.ChecksPassed = false
		result.FailureReasons = append(result.FailureReasons,
			fmt.Sprintf("Sequence number %d is below minimum %d", result.SeqNumber, *t.config.MinSeqNumber))
	}

	// Check ENR fields
	for expectedField, expectedValue := range t.config.ExpectENRField {
		if actualValue, exists := result.ENRFields[expectedField]; !exists {
			result.ChecksPassed = false
			result.FailureReasons = append(result.FailureReasons,
				fmt.Sprintf("Expected ENR field %s not found", expectedField))
		} else if actualValue != expectedValue {
			result.ChecksPassed = false
			result.FailureReasons = append(result.FailureReasons,
				fmt.Sprintf("ENR field %s expected %v, got %v", expectedField, expectedValue, actualValue))
		}
	}
}

// extractCGCFromENR extracts the Custody Group Count from ENR using proper ENR parsing
func (t *Task) extractCGCFromENR(enrStr string) (cgc uint64, enrFields map[string]interface{}, err error) {
	if enrStr == "" {
		t.logger.Debugf("Empty ENR provided")
		return 0, nil, fmt.Errorf("empty ENR")
	}

	t.logger.Debugf("Parsing ENR: %s", enrStr)

	// Decode ENR using go-ethereum's ENR package
	record, err := t.decodeENR(enrStr)
	if err != nil {
		t.logger.Errorf("Failed to decode ENR: %v", err)
		return 0, nil, err
	}

	// Get all key-value pairs from ENR
	enrFields = t.getKeyValuesFromENR(record)

	if cgcHex, ok := enrFields["cgc"]; ok {
		// CGC is stored as hex string, parse it
		cgcStr, ok := cgcHex.(string)
		if !ok {
			t.logger.Warnf("CGC field is not a string: %v", cgcHex)
		} else {
			// Remove "0x" prefix if present
			cgcStr = strings.TrimPrefix(cgcStr, "0x")

			val, err := strconv.ParseUint(cgcStr, 16, 64)
			if err != nil {
				t.logger.Errorf("Failed to parse CGC value %s: %v", cgcStr, err)
			} else {
				cgc = val
				t.logger.Debugf("Found CGC in ENR: %d", cgc)
			}
		}
	} else {
		t.logger.Debugf("No CGC field found in ENR")
	}

	enrFields["enr_original"] = enrStr

	return cgc, enrFields, nil
}

// decodeENR decodes an ENR string into a Record (from Dora's implementation)
func (t *Task) decodeENR(raw string) (*enr.Record, error) {
	b := []byte(raw)
	if strings.HasPrefix(raw, "enr:") {
		b = b[4:]
	}

	dec := make([]byte, base64.RawURLEncoding.DecodedLen(len(b)))

	n, err := base64.RawURLEncoding.Decode(dec, b)
	if err != nil {
		return nil, err
	}

	var r enr.Record

	err = rlp.DecodeBytes(dec[:n], &r)

	return &r, err
}

// getKeyValuesFromENR extracts all key-value pairs from an ENR record (from Dora's implementation)
func (t *Task) getKeyValuesFromENR(r *enr.Record) map[string]interface{} {
	fields := make(map[string]interface{})

	fields["seq"] = r.Seq()
	fields["signature"] = "0x" + hex.EncodeToString(r.Signature())

	// Get all key-value pairs from the record
	kv := r.AppendElements(nil)[1:] // Skip the sequence number
	for i := 0; i < len(kv); i += 2 {
		key, ok := kv[i].(string)
		if !ok {
			t.logger.Warnf("Invalid ENR key type: %T", kv[i])
			continue
		}

		val, ok := kv[i+1].(rlp.RawValue)
		if !ok {
			t.logger.Warnf("Invalid ENR value type for key %s: %T", key, kv[i+1])
			continue
		}

		// Format the value based on the key
		fmtval := t.formatENRValue(key, val)
		fields[key] = fmtval
	}

	return fields
}

// formatENRValue formats an ENR value based on its key type
func (t *Task) formatENRValue(key string, val rlp.RawValue) string {
	switch key {
	case "id":
		content, _, err := rlp.SplitString(val)
		if err == nil {
			return string(content)
		}
	case "ip", "ip6":
		content, _, err := rlp.SplitString(val)
		if err == nil && (len(content) == 4 || len(content) == 16) {
			return fmt.Sprintf("%v", content) // Return IP as string
		}
	case "tcp", "tcp6", "udp", "udp6":
		var x uint64
		if err := rlp.DecodeBytes(val, &x); err == nil {
			return strconv.FormatUint(x, 10)
		}
	case "cgc":
		// CGC is stored as a single byte
		content, _, err := rlp.SplitString(val)
		if err == nil && len(content) > 0 {
			return "0x" + hex.EncodeToString(content)
		}
	}

	// Default: return as hex
	content, _, err := rlp.SplitString(val)
	if err == nil {
		return "0x" + hex.EncodeToString(content)
	}

	return "0x" + hex.EncodeToString(val)
}
