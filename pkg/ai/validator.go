package ai

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/ethpandaops/assertoor/pkg/tasks"
	"gopkg.in/yaml.v3"
)

// ValidationIssue represents a problem found in the generated YAML.
type ValidationIssue struct {
	Type    string `json:"type"`    // "error" or "warning"
	Path    string `json:"path"`    // Path to the issue (e.g., "tasks[0].name")
	Message string `json:"message"` // Human-readable message
}

// ValidationResult contains the results of validating generated YAML.
type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Issues []ValidationIssue `json:"issues,omitempty"`
}

// taskConfig represents the minimal task structure we need to validate.
type taskConfig struct {
	Name       string         `yaml:"name"`
	Title      string         `yaml:"title"`
	Config     map[string]any `yaml:"config"`
	ConfigVars map[string]any `yaml:"configVars"`

	// Child task fields for various glue tasks
	Tasks          []taskConfig `yaml:"tasks"`          // For run_tasks, run_tasks_concurrent
	Task           *taskConfig  `yaml:"task"`           // For run_task_options, run_task_matrix
	ForegroundTask *taskConfig  `yaml:"foregroundTask"` // For run_task_background
	BackgroundTask *taskConfig  `yaml:"backgroundTask"` // For run_task_background
}

// testConfig represents the minimal test structure we need to validate.
type testConfig struct {
	ID           string       `yaml:"id"`
	Name         string       `yaml:"name"`
	Tasks        []taskConfig `yaml:"tasks"`
	CleanupTasks []taskConfig `yaml:"cleanupTasks"`
}

// ValidateGeneratedYaml validates the AI-generated YAML for correct task names and configs.
func ValidateGeneratedYaml(yamlContent string) *ValidationResult {
	result := &ValidationResult{
		Valid:  true,
		Issues: []ValidationIssue{},
	}

	if yamlContent == "" {
		return result
	}

	// Parse the YAML
	var config testConfig

	err := yaml.Unmarshal([]byte(yamlContent), &config)
	if err != nil {
		result.Valid = false
		result.Issues = append(result.Issues, ValidationIssue{
			Type:    "error",
			Path:    "",
			Message: fmt.Sprintf("Invalid YAML syntax: %v", err),
		})

		return result
	}

	// Validate main tasks
	validateTasks(config.Tasks, "tasks", result)

	// Validate cleanup tasks
	validateTasks(config.CleanupTasks, "cleanupTasks", result)

	return result
}

func validateTasks(taskList []taskConfig, basePath string, result *ValidationResult) {
	for i := range taskList {
		taskPath := fmt.Sprintf("%s[%d]", basePath, i)
		validateTask(&taskList[i], taskPath, result)
	}
}

func validateTask(task *taskConfig, path string, result *ValidationResult) {
	// Check if task name is provided
	if task.Name == "" {
		result.Valid = false
		result.Issues = append(result.Issues, ValidationIssue{
			Type:    "error",
			Path:    path + ".name",
			Message: "Task name is required",
		})

		return
	}

	// Check if task exists
	descriptor := tasks.GetTaskDescriptor(task.Name)
	if descriptor == nil {
		result.Valid = false

		// Try to find similar task names
		suggestions := findSimilarTaskNames(task.Name)
		suggestionMsg := ""

		if len(suggestions) > 0 {
			suggestionMsg = fmt.Sprintf(". Did you mean: %s?", strings.Join(suggestions, ", "))
		}

		result.Issues = append(result.Issues, ValidationIssue{
			Type:    "error",
			Path:    path + ".name",
			Message: fmt.Sprintf("Unknown task '%s'%s", task.Name, suggestionMsg),
		})

		return
	}

	// Validate config fields if config exists
	if task.Config != nil && descriptor.Config != nil {
		validateTaskConfig(task.Config, descriptor.Config, path+".config", result)
	}

	// Validate configVars - all values must be string expressions and field names must be valid
	if task.ConfigVars != nil {
		validateConfigVars(task.ConfigVars, descriptor.Config, path+".configVars", result)
	}

	// Recursively validate child tasks (for glue tasks)
	if len(task.Tasks) > 0 {
		validateTasks(task.Tasks, path+".tasks", result)
	}

	// Validate single child task fields (run_task_options, run_task_matrix)
	if task.Task != nil {
		validateTask(task.Task, path+".task", result)
	}

	// Validate background task fields (run_task_background)
	if task.ForegroundTask != nil {
		validateTask(task.ForegroundTask, path+".foregroundTask", result)
	}

	if task.BackgroundTask != nil {
		validateTask(task.BackgroundTask, path+".backgroundTask", result)
	}
}

func validateTaskConfig(provided map[string]any, expected any, path string, result *ValidationResult) {
	expectedType := reflect.TypeOf(expected)
	if expectedType.Kind() == reflect.Pointer {
		expectedType = expectedType.Elem()
	}

	if expectedType.Kind() != reflect.Struct {
		return
	}

	// Build a map of valid field names (from json tags)
	validFields := make(map[string]reflect.StructField)

	for i := range expectedType.NumField() {
		field := expectedType.Field(i)
		if !field.IsExported() {
			continue
		}

		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}

		fieldName := getFieldName(jsonTag, field.Name)
		validFields[fieldName] = field

		// Also check yaml tag as alternative
		yamlTag := field.Tag.Get("yaml")
		if yamlTag != "" && yamlTag != "-" {
			yamlFieldName := strings.Split(yamlTag, ",")[0]
			if yamlFieldName != "" && yamlFieldName != fieldName {
				validFields[yamlFieldName] = field
			}
		}
	}

	// Check each provided config field
	for fieldName, providedValue := range provided {
		field, ok := validFields[fieldName]
		if !ok {
			// Field doesn't exist - find similar fields
			suggestions := findSimilarFields(fieldName, validFields)
			suggestionMsg := ""

			if len(suggestions) > 0 {
				suggestionMsg = fmt.Sprintf(". Did you mean: %s?", strings.Join(suggestions, ", "))
			}

			// Unknown fields are errors, not warnings
			result.Valid = false
			result.Issues = append(result.Issues, ValidationIssue{
				Type:    "error",
				Path:    path + "." + fieldName,
				Message: fmt.Sprintf("Unknown config field '%s'%s", fieldName, suggestionMsg),
			})

			continue
		}

		// Check for type mismatches (especially string vs array)
		validateFieldType(providedValue, field.Type, path+"."+fieldName, result)
	}
}

// validateConfigVars validates that all configVars are string expressions and reference valid config fields.
func validateConfigVars(configVars map[string]any, expectedConfig any, path string, result *ValidationResult) {
	// Build valid field names from the expected config
	validFields := make(map[string]bool)

	if expectedConfig != nil {
		expectedType := reflect.TypeOf(expectedConfig)
		if expectedType.Kind() == reflect.Pointer {
			expectedType = expectedType.Elem()
		}

		if expectedType.Kind() == reflect.Struct {
			for i := range expectedType.NumField() {
				field := expectedType.Field(i)
				if !field.IsExported() {
					continue
				}

				jsonTag := field.Tag.Get("json")
				if jsonTag == "-" {
					continue
				}

				fieldName := getFieldName(jsonTag, field.Name)
				validFields[fieldName] = true

				// Also check yaml tag as alternative
				yamlTag := field.Tag.Get("yaml")
				if yamlTag != "" && yamlTag != "-" {
					yamlFieldName := strings.Split(yamlTag, ",")[0]
					if yamlFieldName != "" {
						validFields[yamlFieldName] = true
					}
				}
			}
		}
	}

	for fieldName, value := range configVars {
		fieldPath := path + "." + fieldName

		// Check that the value is a string (JQ expression)
		if _, ok := value.(string); !ok {
			result.Valid = false

			valueType := "nil"
			if value != nil {
				valueType = reflect.TypeOf(value).Kind().String()
			}

			result.Issues = append(result.Issues, ValidationIssue{
				Type:    "error",
				Path:    fieldPath,
				Message: fmt.Sprintf("configVars values must be string expressions, but '%s' has type %s", fieldName, valueType),
			})

			continue
		}

		// Check that the field name is valid (if we have config schema)
		if len(validFields) > 0 && !validFields[fieldName] {
			suggestions := findSimilarConfigVarFields(fieldName, validFields)
			suggestionMsg := ""

			if len(suggestions) > 0 {
				suggestionMsg = fmt.Sprintf(". Did you mean: %s?", strings.Join(suggestions, ", "))
			}

			result.Valid = false
			result.Issues = append(result.Issues, ValidationIssue{
				Type:    "error",
				Path:    fieldPath,
				Message: fmt.Sprintf("Unknown config field '%s' in configVars%s", fieldName, suggestionMsg),
			})
		}
	}
}

// findSimilarConfigVarFields finds field names similar to the given name.
func findSimilarConfigVarFields(name string, validFields map[string]bool) []string {
	suggestions := []string{}
	nameLower := strings.ToLower(name)

	for fieldName := range validFields {
		fieldNameLower := strings.ToLower(fieldName)

		// Check for substring match
		if strings.Contains(fieldNameLower, nameLower) || strings.Contains(nameLower, fieldNameLower) {
			suggestions = append(suggestions, fieldName)

			if len(suggestions) >= 3 {
				break
			}
		}
	}

	return suggestions
}

// validateFieldType checks if the provided value matches the expected type.
func validateFieldType(provided any, expectedType reflect.Type, path string, result *ValidationResult) {
	if provided == nil {
		return
	}

	// Handle pointer types
	if expectedType.Kind() == reflect.Pointer {
		expectedType = expectedType.Elem()
	}

	providedType := reflect.TypeOf(provided)

	// Check for array/slice type mismatch
	if expectedType.Kind() == reflect.Slice || expectedType.Kind() == reflect.Array {
		// Expected is an array, check if provided is NOT an array
		if providedType.Kind() != reflect.Slice && providedType.Kind() != reflect.Array {
			result.Valid = false
			result.Issues = append(result.Issues, ValidationIssue{
				Type:    "error",
				Path:    path,
				Message: fmt.Sprintf("Expected array but got %s. Use array syntax: [value1, value2]", providedType.Kind()),
			})
		}

		return
	}

	// Check for string expected but array provided
	if expectedType.Kind() == reflect.String {
		if providedType.Kind() == reflect.Slice || providedType.Kind() == reflect.Array {
			result.Valid = false
			result.Issues = append(result.Issues, ValidationIssue{
				Type:    "error",
				Path:    path,
				Message: "Expected string but got array",
			})
		}
	}
}

func getFieldName(jsonTag, defaultName string) string {
	if jsonTag != "" {
		parts := strings.Split(jsonTag, ",")
		if parts[0] != "" {
			return parts[0]
		}
	}

	return defaultName
}

// findSimilarTaskNames finds task names similar to the given name.
func findSimilarTaskNames(name string) []string {
	allDescriptors := tasks.GetAllTaskDescriptorsAPI()
	suggestions := []string{}
	nameLower := strings.ToLower(name)

	for _, desc := range allDescriptors {
		descNameLower := strings.ToLower(desc.Name)

		// Check for substring match
		if strings.Contains(descNameLower, nameLower) || strings.Contains(nameLower, descNameLower) {
			suggestions = append(suggestions, desc.Name)

			if len(suggestions) >= 3 {
				break
			}

			continue
		}

		// Check for common prefix
		if len(nameLower) > 3 && len(descNameLower) > 3 {
			if strings.HasPrefix(descNameLower, nameLower[:3]) {
				suggestions = append(suggestions, desc.Name)

				if len(suggestions) >= 3 {
					break
				}
			}
		}
	}

	return suggestions
}

// findSimilarFields finds field names similar to the given name.
func findSimilarFields(name string, validFields map[string]reflect.StructField) []string {
	suggestions := []string{}
	nameLower := strings.ToLower(name)

	for fieldName := range validFields {
		fieldNameLower := strings.ToLower(fieldName)

		// Check for substring match
		if strings.Contains(fieldNameLower, nameLower) || strings.Contains(nameLower, fieldNameLower) {
			suggestions = append(suggestions, fieldName)

			if len(suggestions) >= 3 {
				break
			}
		}
	}

	return suggestions
}

// GetAvailableTaskNames returns all available task names for reference.
func GetAvailableTaskNames() []string {
	allDescriptors := tasks.GetAllTaskDescriptorsAPI()
	names := make([]string, len(allDescriptors))

	for i, desc := range allDescriptors {
		names[i] = desc.Name
	}

	return names
}

// FormatValidationIssues formats validation issues as a human-readable string.
func FormatValidationIssues(issues []ValidationIssue) string {
	if len(issues) == 0 {
		return ""
	}

	var sb strings.Builder

	sb.WriteString("Validation issues found:\n")

	for _, issue := range issues {
		prefix := "⚠️"
		if issue.Type == "error" {
			prefix = "❌"
		}

		if issue.Path != "" {
			sb.WriteString(fmt.Sprintf("%s [%s] %s\n", prefix, issue.Path, issue.Message))
		} else {
			sb.WriteString(fmt.Sprintf("%s %s\n", prefix, issue.Message))
		}
	}

	return sb.String()
}

// SerializeValidationResult converts a validation result to JSON.
func SerializeValidationResult(result *ValidationResult) (json.RawMessage, error) {
	return json.Marshal(result)
}
