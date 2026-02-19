package ai

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/ethpandaops/assertoor/pkg/tasks"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Static system prompt - this part can be cached by the LLM provider.
// Keep all static content here, dynamic content (like current YAML) should be added as separate messages.
const systemPromptTemplate = `You are an AI assistant for Assertoor, a tool for testing Ethereum consensus and execution clients. Your role is to help users build and modify test configurations through natural language interaction.

## CRITICAL: Only Use Listed Tasks

IMPORTANT: You must ONLY use the tasks listed in this prompt. Do NOT invent, guess, or use any task names that are not explicitly listed below. If a user requests functionality that doesn't match any available task, explain which available tasks might help or that the functionality is not available.

The complete and exhaustive list of all available tasks follows. There are no other tasks.

## Your Capabilities
- Create new test configurations from scratch based on user descriptions
- Modify existing test configurations
- Explain what specific tasks do
- Suggest appropriate tasks from the available list for testing scenarios
- Fix issues in test YAML configurations

## Test Configuration Format

Tests are defined in YAML format with the following structure:

%s

## Best Practices

### 1. Always Start with Health Check
Every test MUST begin with a check_clients_are_healthy task to ensure at least one client is ready:
` + "```yaml" + `
tasks:
  - name: check_clients_are_healthy
    title: "Check if at least one client is ready"
    timeout: 5m
    config:
      minClientCount: 1
` + "```" + `

### 2. Use Global Config Variables (Only If Needed)
Add settings to the test config section ONLY if they are actually used by tasks in the test. Do NOT include unused global variables.

Well-known global variables (provided by Assertoor runtime):
- ` + "`walletPrivkey`" + ` - Root wallet private key (NEVER use directly in tasks)
- ` + "`clientPairNames`" + ` - Array of all client names
- ` + "`validatorPairNames`" + ` - Array of client names with active validators
- ` + "`validatorMnemonic`" + ` - Mnemonic for the existing validator set (local devnets only)

IMPORTANT: Only declare a global variable in the test config if a task actually references it via configVars.
` + "```yaml" + `
# GOOD - only declares walletPrivkey because it's used by generate_child_wallet
config:
  walletPrivkey: ""
  depositCount: 10

# BAD - declares unused variables
config:
  walletPrivkey: ""
  clientPairNames: []         # Not used - don't include!
  validatorPairNames: []      # Not used - don't include!
` + "```" + `

### 3. Never Use Root Wallet Directly
Tests should NEVER use the root wallet (walletPrivkey) directly. Instead, create a child wallet with appropriate funding:
` + "```yaml" + `
- name: generate_child_wallet
  id: test_wallet
  title: "Generate wallet for test operations"
  config:
    walletSeed: "my-test-unique-seed"
    prefundMinBalance: 101000000000000000000  # 101 ETH - REQUIRED!
  configVars:
    privateKey: "walletPrivkey"  # Derives from root wallet
` + "```" + `
IMPORTANT: You MUST specify ` + "`prefundMinBalance`" + ` with enough ETH for the test operations:
- Each deposit requires ~32 ETH
- Each transaction needs gas (~0.01 ETH)
- Add buffer for multiple operations
- Calculate: (deposits * 32 ETH) + (transactions * 0.1 ETH) + buffer

Then use the child wallet in subsequent tasks via: tasks.test_wallet.outputs.childWallet

### 4. Validator Management

**Quick/Dirty Tests (Local Devnets Only):**
For simple tests that just need to interact with existing validators, use ` + "`validatorMnemonic`" + `:
` + "```yaml" + `
config:
  validatorMnemonic: ""  # Existing validator set mnemonic
` + "```" + `
This gives access to the pre-existing validator set on local devnets.

**Production/Testnet-Grade Tests:**
For proper tests, always deposit your own validators and exit them in cleanup tasks:
` + "```yaml" + `
# 1. Generate a random mnemonic (ALWAYS use random, never hardcoded!)
- name: get_random_mnemonic
  id: test_mnemonic
  title: "Generate random mnemonic for test validators"

# 2. Generate deposits with the random mnemonic
- name: generate_deposits
  title: "Create test validator deposits"
  config:
    depositCount: 2
    # ... other deposit config
  configVars:
    mnemonic: "tasks.test_mnemonic.outputs.mnemonic"
    walletPrivkey: "tasks.test_wallet.outputs.childWallet"

# In cleanupTasks: exit the validators
cleanupTasks:
  - name: generate_exits
    title: "Exit test validators"
    configVars:
      mnemonic: "tasks.test_mnemonic.outputs.mnemonic"
` + "```" + `

**IMPORTANT:** Depositing validators can take several hours to activate! Plan test timeouts accordingly.

## Complete List of Available Tasks

The following is the COMPLETE and EXHAUSTIVE list of all available tasks. You MUST NOT use any task name that is not in this list.

%s

## Variable System

### Test Variables (configVars)
Tasks can reference runtime variables using JQ expressions in the configVars field:

%s

### Task Outputs
Each task can produce outputs accessible by other tasks:
- Access pattern: tasks.<task-id>.outputs.<field>
- Example: tasks.get_specs.outputs.specs.ELECTRA_FORK_EPOCH

### Task Status Variables
Each task has status variables:
- tasks.<id>.result: 0=none, 1=success, 2=failure
- tasks.<id>.running: boolean
- tasks.<id>.started: boolean
- tasks.<id>.timeout: boolean

## Glue Tasks (Flow Control)

Glue tasks control execution flow. These ARE in the task list above, but here's how to use them:

%s

## YAML Type Rules

CRITICAL: Pay attention to field types in the task documentation!

### Array Fields (type: array)
Array fields MUST use YAML array syntax, NOT strings:
` + "```yaml" + `
# CORRECT - array syntax
config:
  targetClients: ["client-1", "client-2"]
  # OR multi-line:
  targetClients:
    - "client-1"
    - "client-2"

# WRONG - string instead of array (will cause errors!)
config:
  targetClients: "client-1"
` + "```" + `

### String Fields (type: string)
String fields use plain values:
` + "```yaml" + `
config:
  clientPattern: ".*"
  walletSeed: "my-seed"
` + "```" + `

### Boolean Fields (type: boolean)
Use true/false without quotes:
` + "```yaml" + `
config:
  enabled: true
  skipCheck: false
` + "```" + `

### Integer Fields (type: integer)
Use numbers without quotes:
` + "```yaml" + `
config:
  count: 10
  minClientCount: 1
` + "```" + `

## Response Format

When generating or modifying test configurations:
1. Always provide valid YAML in a code block
2. ONLY use task names from the list above - do not invent new task names
3. Pay attention to field types - arrays MUST be arrays, not strings!
4. Include explanations of what you changed or created
5. If the request is unclear, ask clarifying questions before generating YAML

When you generate YAML, wrap it in a yaml code block like this:
%stest configuration here%s

Always ensure the YAML is complete and valid. Do not use placeholders like "..." or "# add more tasks here".

REMINDER: Only use tasks from the Complete List of Available Tasks above. Any other task name will cause validation errors.
`

const testStructureTemplate = `name: "Test Name"
timeout: "30m"  # Optional timeout
config:
  # Only declare variables that are actually used by tasks!
  walletPrivkey: ""     # Needed because generate_child_wallet uses it
  depositCount: 10      # Custom test setting
tasks:
  # ALWAYS start with health check
  - name: check_clients_are_healthy
    title: "Check if at least one client is ready"
    timeout: 5m
    config:
      minClientCount: 1

  # Create child wallet for test operations (never use root wallet directly)
  - name: generate_child_wallet
    id: test_wallet
    title: "Generate test wallet"
    config:
      walletSeed: "my-test-seed"
      prefundMinBalance: 10000000000000000000  # 10 ETH
    configVars:
      privateKey: "walletPrivkey"  # This is why walletPrivkey is in config

  # Example task using outputs and config vars
  - name: task_type_name
    id: optional_task_id
    title: "Human readable title"
    timeout: "5m"
    config:
      # For array fields, always use YAML array syntax:
      clientPattern: ".*"           # string
      targetClients: ["client-1"]   # array of strings - use brackets!
    configVars:
      walletPrivkey: "tasks.test_wallet.outputs.childWallet"
      count: "depositCount"
cleanupTasks:
  # Tasks to run after main tasks complete (success or failure)
  - name: cleanup_task_name`

const configVarsTemplate = `Example:
  configVars:
    walletPrivkey: "walletPrivkey"  # Direct variable reference
    minEpoch: "tasks.get_specs.outputs.specs.ELECTRA_FORK_EPOCH"  # Task output
    computed: '| (.value1 | tonumber) + 100'  # JQ computation`

const typeAny = "any"

const glueTasksTemplate = `1. run_tasks: Sequential execution
   - Tasks run one after another
   - Fails on first child failure (unless continueOnFailure: true)

2. run_tasks_concurrent: Parallel execution
   - All tasks start simultaneously
   - Configure successThreshold/failureThreshold for completion criteria
   - Use stopOnThreshold: true for early termination

3. run_task_matrix: Parameterized execution
   - Runs same task multiple times with different values
   - Set matrixVar and matrixValues
   - runConcurrent: true for parallel execution

4. run_task_options: Single task wrapper
   - Retry support (retryOnFailure, maxRetryCount)
   - Result transformation (invertResult, ignoreResult)

5. run_task_background: Foreground + Background
   - Run a background task while executing foreground task
   - Configure exitOnForegroundSuccess/exitOnForegroundFailure`

// BuildSystemPrompt creates the static system prompt for the AI assistant.
// This prompt is designed to be cachable - all static content is included here.
// Dynamic content (current YAML context) should be added as separate messages.
func BuildSystemPrompt() string {
	taskDocs := buildTaskDocumentation()

	return fmt.Sprintf(
		systemPromptTemplate,
		testStructureTemplate,
		taskDocs,
		configVarsTemplate,
		glueTasksTemplate,
		"```yaml\n",
		"\n```",
	)
}

// buildTaskDocumentation creates complete documentation for all available tasks.
// This includes every task name, description, inputs (config fields), and outputs.
func buildTaskDocumentation() string {
	descriptors := tasks.GetAllTaskDescriptorsAPI()

	// Group tasks by category
	categories := make(map[string][]tasks.TaskDescriptorAPI)
	allTaskNames := make([]string, 0, len(descriptors))

	for _, desc := range descriptors {
		category := desc.Category
		if category == "" {
			category = "other"
		}

		categories[category] = append(categories[category], desc)
		allTaskNames = append(allTaskNames, desc.Name)
	}

	var builder strings.Builder

	// First, list all task names for quick reference
	builder.WriteString("### Quick Reference - All Valid Task Names\n\n")
	builder.WriteString("```\n")

	for i, name := range allTaskNames {
		builder.WriteString(name)

		if i < len(allTaskNames)-1 {
			if (i+1)%5 == 0 {
				builder.WriteString("\n")
			} else {
				builder.WriteString(", ")
			}
		}
	}

	builder.WriteString("\n```\n\n")
	builder.WriteString(fmt.Sprintf("Total: %d tasks available\n\n", len(allTaskNames)))

	// Define category order
	categoryOrder := []string{"check", "generate", "get", "run", "validator", "utility", "other"}

	titleCaser := cases.Title(language.English)

	for _, category := range categoryOrder {
		descs, ok := categories[category]
		if !ok || len(descs) == 0 {
			continue
		}

		builder.WriteString(fmt.Sprintf("### %s Tasks\n\n", titleCaser.String(category)))

		for _, desc := range descs {
			builder.WriteString(fmt.Sprintf("#### %s\n", desc.Name))
			builder.WriteString(fmt.Sprintf("**Description:** %s\n\n", desc.Description))

			// Add config inputs (from JSON schema)
			writeConfigInputs(&builder, desc.ConfigSchema)

			// Add outputs
			if len(desc.Outputs) > 0 {
				builder.WriteString("**Outputs:**\n")

				for _, output := range desc.Outputs {
					builder.WriteString(fmt.Sprintf("- `%s` (%s): %s\n", output.Name, output.Type, output.Description))
				}

				builder.WriteString("\n")
			}
		}
	}

	return builder.String()
}

// writeConfigInputs writes the config input fields from the JSON schema.
func writeConfigInputs(builder *strings.Builder, configSchema json.RawMessage) {
	if len(configSchema) == 0 {
		return
	}

	var schema map[string]any
	if err := json.Unmarshal(configSchema, &schema); err != nil {
		return
	}

	props, ok := schema["properties"].(map[string]any)
	if !ok || len(props) == 0 {
		return
	}

	// Collect requirement groups
	requireGroups := make(map[string][]string) // group -> list of fields

	builder.WriteString("**Config Inputs:**\n")

	for propName, propValue := range props {
		propMap, ok := propValue.(map[string]any)
		if !ok {
			continue
		}

		propType := getTypeString(propMap["type"])
		propDesc, _ := propMap["description"].(string)
		requireGroup, _ := propMap["requireGroup"].(string)

		// Track requirement groups
		if requireGroup != "" {
			requireGroups[requireGroup] = append(requireGroups[requireGroup], propName)
		}

		// Handle nested object types
		if propType == "object" {
			if nested, ok := propMap["properties"].(map[string]any); ok && len(nested) > 0 {
				propType = "object (nested)"
			} else if propMap["additionalProperties"] != nil {
				propType = "object (map)"
			}
		}

		// Handle array types with item type info - add clear array marker
		isArray := propType == "array"
		if isArray {
			itemType := "any"
			if items, ok := propMap["items"].(map[string]any); ok {
				itemType = getTypeString(items["type"])
			}

			propType = fmt.Sprintf("array[%s] - MUST use array syntax: [value1, value2]", itemType)
		}

		// Add requirement marker
		requireMarker := ""
		if requireGroup != "" {
			requireMarker = fmt.Sprintf(" [REQUIRED:%s]", requireGroup)
		}

		if propDesc != "" {
			fmt.Fprintf(builder, "- `%s` (%s)%s: %s\n", propName, propType, requireMarker, propDesc)
		} else {
			fmt.Fprintf(builder, "- `%s` (%s)%s\n", propName, propType, requireMarker)
		}
	}

	// Write requirement group explanations if any
	if len(requireGroups) > 0 {
		builder.WriteString("\n**Required Fields:**\n")
		writeRequirementExplanation(builder, requireGroups)
	}

	builder.WriteString("\n")
}

// writeRequirementExplanation writes a human-readable explanation of requirement groups.
func writeRequirementExplanation(builder *strings.Builder, groups map[string][]string) {
	// Group by base letter (A, B, C) to find alternatives
	baseGroups := make(map[string]map[string][]string) // base -> subgroup -> fields

	for group, fields := range groups {
		parts := strings.Split(group, ".")
		base := parts[0]
		subgroup := group

		if baseGroups[base] == nil {
			baseGroups[base] = make(map[string][]string)
		}

		baseGroups[base][subgroup] = fields
	}

	// Sort bases for consistent output
	bases := make([]string, 0, len(baseGroups))
	for base := range baseGroups {
		bases = append(bases, base)
	}

	sort.Strings(bases)

	for _, base := range bases {
		subgroups := baseGroups[base]
		if len(subgroups) == 1 {
			// Single option - all fields required
			for _, fields := range subgroups {
				if len(fields) == 1 {
					fmt.Fprintf(builder, "- `%s` is required\n", fields[0])
				} else {
					fmt.Fprintf(builder, "- All of: %s are required\n", formatFieldList(fields))
				}
			}
		} else {
			// Multiple options - one subgroup must be satisfied
			builder.WriteString("- One of the following is required:\n")

			for subgroup, fields := range subgroups {
				if len(fields) == 1 {
					fmt.Fprintf(builder, "  - Option %s: `%s`\n", subgroup, fields[0])
				} else {
					fmt.Fprintf(builder, "  - Option %s: %s (all together)\n", subgroup, formatFieldList(fields))
				}
			}
		}
	}
}

func formatFieldList(fields []string) string {
	quoted := make([]string, len(fields))
	for i, f := range fields {
		quoted[i] = "`" + f + "`"
	}

	return strings.Join(quoted, ", ")
}

func getTypeString(t any) string {
	if t == nil {
		return typeAny
	}

	switch v := t.(type) {
	case string:
		return v
	case []any:
		if len(v) > 0 {
			if s, ok := v[0].(string); ok {
				return s
			}
		}

		return typeAny
	default:
		return typeAny
	}
}
