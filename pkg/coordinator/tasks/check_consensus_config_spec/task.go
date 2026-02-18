package checkconsensusconfigspec

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/clients"
	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/ethpandaops/assertoor/pkg/coordinator/vars"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var (
	TaskName       = "check_consensus_config_spec"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Checks consensus clients for compliance with the /eth/v1/config/spec endpoint specification.",
		Config:      DefaultConfig(),
		NewTask:     NewTask,
	}
)

type Task struct {
	ctx              *types.TaskContext
	options          *types.TaskOptions
	config           Config
	logger           logrus.FieldLogger
	expectedSpec     map[string]interface{}
	expectedSpecKeys []string
}

type ClientValidationResult struct {
	Name            string            `json:"name"`
	IsValid         bool              `json:"isValid"`
	MissingFields   []string          `json:"missingFields"`
	ExtraFields     []string          `json:"extraFields"`
	ErrorMessage    string            `json:"errorMessage"`
	ReceivedSpec    map[string]interface{} `json:"receivedSpec"`
	ComparisonIssues []string         `json:"comparisonIssues"`
}

type ValidationSummary struct {
	TotalClients   int                      `json:"totalClients"`
	ValidClients   int                      `json:"validClients"`
	InvalidClients int                      `json:"invalidClients"`
	Results        []*ClientValidationResult `json:"results"`
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
	// Fetch expected spec from the source
	if err := t.fetchExpectedSpec(ctx); err != nil {
		return fmt.Errorf("failed to fetch expected spec: %w", err)
	}

	t.logger.Infof("Loaded combined spec with %d fields from main config and %d preset files", 
		len(t.expectedSpecKeys), len(t.config.PresetFiles))

	// Initial check
	t.processCheck(ctx)

	// Poll for updates
	for {
		select {
		case <-time.After(t.config.PollInterval.Duration):
			t.processCheck(ctx)
		case <-ctx.Done():
			return nil
		}
	}
}

func (t *Task) fetchExpectedSpec(ctx context.Context) error {
	// Initialize the combined spec
	t.expectedSpec = make(map[string]interface{})

	// First, fetch the main config spec
	if err := t.fetchConfigFile(ctx, t.config.specSource); err != nil {
		return fmt.Errorf("failed to fetch main config spec: %w", err)
	}

	// Then fetch and combine all preset files
	for _, presetFile := range t.config.PresetFiles {
		presetURL := fmt.Sprintf("%s/%s", t.config.presetBaseURL, presetFile)
		if err := t.fetchConfigFile(ctx, presetURL); err != nil {
			// Log warning but continue if preset file is not found (some might be optional)
			t.logger.Warnf("Failed to fetch preset file %s: %v", presetFile, err)
			continue
		}
	}

	// Build the keys list from the combined spec
	t.expectedSpecKeys = make([]string, 0, len(t.expectedSpec))
	for key := range t.expectedSpec {
		t.expectedSpecKeys = append(t.expectedSpecKeys, key)
	}

	// If no required fields specified in config, use all fields from combined spec
	if len(t.config.RequiredFields) == 0 {
		t.config.RequiredFields = t.expectedSpecKeys
	}

	t.logger.Infof("Combined spec contains %d total fields from main config and %d preset files", 
		len(t.expectedSpecKeys), len(t.config.PresetFiles))

	return nil
}

func (t *Task) fetchConfigFile(ctx context.Context, url string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request for %s: %w", url, err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch %s, status code: %d", url, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body from %s: %w", url, err)
	}

	var config map[string]interface{}
	if err := yaml.Unmarshal(body, &config); err != nil {
		return fmt.Errorf("failed to parse YAML from %s: %w", url, err)
	}

	// Merge the config into the expected spec
	for key, value := range config {
		if existingValue, exists := t.expectedSpec[key]; exists {
			t.logger.Debugf("Overriding spec field %s: %v -> %v", key, existingValue, value)
		}
		t.expectedSpec[key] = value
	}

	return nil
}

func (t *Task) processCheck(ctx context.Context) {
	allResultsPass := true
	validationSummary := &ValidationSummary{
		Results: []*ClientValidationResult{},
	}

	clients := t.ctx.Scheduler.GetServices().ClientPool().GetClientsByNamePatterns(t.config.ClientPattern, "")
	validationSummary.TotalClients = len(clients)

	for _, client := range clients {
		result := t.validateClient(ctx, client)
		validationSummary.Results = append(validationSummary.Results, result)

		if result.IsValid {
			validationSummary.ValidClients++
		} else {
			validationSummary.InvalidClients++
			allResultsPass = false
		}
	}

	t.logValidationResults(validationSummary)

	// Set output variables
	if validationData, err := vars.GeneralizeData(validationSummary); err == nil {
		t.ctx.Outputs.SetVar("validationSummary", validationData)
	} else {
		t.logger.Warnf("Failed setting `validationSummary` output: %v", err)
	}

	if allResultsPass {
		t.ctx.SetResult(types.TaskResultSuccess)
	} else {
		t.ctx.SetResult(types.TaskResultNone)
	}
}

func (t *Task) validateClient(ctx context.Context, client *clients.PoolClient) *ClientValidationResult {
	result := &ClientValidationResult{
		Name:            client.Config.Name,
		MissingFields:   []string{},
		ExtraFields:     []string{},
		ComparisonIssues: []string{},
	}

	checkLogger := t.logger.WithField("client", client.Config.Name)

	// Fetch spec from client
	receivedSpec, err := client.ConsensusClient.GetRPCClient().GetConfigSpecs(ctx)
	if err != nil {
		checkLogger.Errorf("Failed to fetch config specs: %v", err)
		result.ErrorMessage = fmt.Sprintf("Failed to fetch config specs: %v", err)
		return result
	}

	if receivedSpec == nil {
		checkLogger.Error("Received nil config specs response")
		result.ErrorMessage = "Received nil config specs response"
		return result
	}

	result.ReceivedSpec = receivedSpec

	// Check for missing required fields
	for _, requiredField := range t.config.RequiredFields {
		if _, exists := receivedSpec[requiredField]; !exists {
			result.MissingFields = append(result.MissingFields, requiredField)
		}
	}

	// Check for extra fields if not allowed
	if !t.config.AllowExtraFields {
		expectedFieldsMap := make(map[string]bool)
		for _, field := range t.expectedSpecKeys {
			expectedFieldsMap[field] = true
		}

		for receivedField := range receivedSpec {
			if !expectedFieldsMap[receivedField] {
				result.ExtraFields = append(result.ExtraFields, receivedField)
			}
		}
	}

	// Compare field values with expected spec
	for field, expectedValue := range t.expectedSpec {
		if receivedValue, exists := receivedSpec[field]; exists {
			if !t.compareValues(expectedValue, receivedValue) {
				result.ComparisonIssues = append(result.ComparisonIssues, 
					fmt.Sprintf("Field '%s': expected %v (type: %T), got %v (type: %T)", 
						field, expectedValue, expectedValue, receivedValue, receivedValue))
			}
		}
	}

	result.IsValid = len(result.MissingFields) == 0 && 
		(t.config.AllowExtraFields || len(result.ExtraFields) == 0) &&
		len(result.ComparisonIssues) == 0

	return result
}

func (t *Task) compareValues(expected, received interface{}) bool {
	// Convert both values to strings for comparison to handle type differences
	expectedStr := fmt.Sprintf("%v", expected)
	receivedStr := fmt.Sprintf("%v", received)
	
	// Try to compare as strings first
	if expectedStr == receivedStr {
		return true
	}
	
	// For numeric values, try to compare with type conversion
	if reflect.TypeOf(expected).Kind() == reflect.TypeOf(received).Kind() {
		return reflect.DeepEqual(expected, received)
	}
	
	return false
}

func (t *Task) logValidationResults(summary *ValidationSummary) {
	if summary.InvalidClients == 0 {
		t.logger.Infof("✅ All %d clients passed spec validation", summary.TotalClients)
		return
	}

	t.logger.Errorf("❌ %d/%d clients failed spec validation", summary.InvalidClients, summary.TotalClients)

	for _, result := range summary.Results {
		if !result.IsValid {
			clientLogger := t.logger.WithField("client", result.Name)
			
			if result.ErrorMessage != "" {
				clientLogger.Errorf("Error: %s", result.ErrorMessage)
				continue
			}

			if len(result.MissingFields) > 0 {
				clientLogger.Errorf("Missing required fields: %s", strings.Join(result.MissingFields, ", "))
			}

			if len(result.ExtraFields) > 0 {
				clientLogger.Warnf("Extra fields not in spec: %s", strings.Join(result.ExtraFields, ", "))
			}

			if len(result.ComparisonIssues) > 0 {
				clientLogger.Errorf("Value mismatches: %s", strings.Join(result.ComparisonIssues, "; "))
			}
		}
	}
}