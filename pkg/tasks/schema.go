package tasks

import (
	"encoding/json"
	"reflect"
	"strings"

	"github.com/ethpandaops/assertoor/pkg/types"
)

// TaskDescriptorAPI represents a task descriptor for API responses.
type TaskDescriptorAPI struct {
	Name         string                       `json:"name"`
	Aliases      []string                     `json:"aliases,omitempty"`
	Description  string                       `json:"description"`
	Category     string                       `json:"category,omitempty"`
	ConfigSchema json.RawMessage              `json:"configSchema"`
	Outputs      []types.TaskOutputDefinition `json:"outputs,omitempty"`
}

// GetAllTaskDescriptorsAPI returns all task descriptors formatted for API responses.
func GetAllTaskDescriptorsAPI() []TaskDescriptorAPI {
	descriptors := make([]TaskDescriptorAPI, 0, len(AvailableTaskDescriptors))

	for _, desc := range AvailableTaskDescriptors {
		apiDesc := TaskDescriptorAPI{
			Name:        desc.Name,
			Aliases:     desc.Aliases,
			Description: desc.Description,
			Category:    desc.Category,
			Outputs:     desc.Outputs,
		}

		if desc.Config != nil {
			schema, err := GenerateJSONSchema(desc.Config)
			if err == nil {
				apiDesc.ConfigSchema = schema
			}
		}

		descriptors = append(descriptors, apiDesc)
	}

	return descriptors
}

// GetTaskDescriptorAPI returns a single task descriptor formatted for API response.
func GetTaskDescriptorAPI(name string) *TaskDescriptorAPI {
	desc := GetTaskDescriptor(name)
	if desc == nil {
		return nil
	}

	apiDesc := &TaskDescriptorAPI{
		Name:        desc.Name,
		Aliases:     desc.Aliases,
		Description: desc.Description,
		Category:    desc.Category,
		Outputs:     desc.Outputs,
	}

	if desc.Config != nil {
		schema, err := GenerateJSONSchema(desc.Config)
		if err == nil {
			apiDesc.ConfigSchema = schema
		}
	}

	return apiDesc
}

// JSONSchema represents a JSON Schema object.
type JSONSchema struct {
	Type                 string                 `json:"type,omitempty"`
	Description          string                 `json:"description,omitempty"`
	Properties           map[string]*JSONSchema `json:"properties,omitempty"`
	Required             []string               `json:"required,omitempty"`
	Items                *JSONSchema            `json:"items,omitempty"`
	Default              any                    `json:"default,omitempty"`
	Enum                 []any                  `json:"enum,omitempty"`
	AdditionalProperties *JSONSchema            `json:"additionalProperties,omitempty"`
}

// GenerateJSONSchema generates a JSON Schema from a Go struct via reflection.
func GenerateJSONSchema(v any) (json.RawMessage, error) {
	schema := generateSchemaFromType(reflect.TypeOf(v))

	return json.Marshal(schema)
}

func generateSchemaFromType(t reflect.Type) *JSONSchema {
	// Handle pointer types
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	schema := &JSONSchema{}

	switch t.Kind() {
	case reflect.Struct:
		schema.Type = "object"
		schema.Properties = make(map[string]*JSONSchema, t.NumField())

		for i := range t.NumField() {
			field := t.Field(i)

			// Skip unexported fields
			if !field.IsExported() {
				continue
			}

			// Get JSON tag for field name
			jsonTag := field.Tag.Get("json")
			if jsonTag == "-" {
				continue
			}

			fieldName := getJSONFieldName(&field, jsonTag)

			// Generate schema for field type
			fieldSchema := generateSchemaFromType(field.Type)

			// Add description from desc tag, falling back to yaml tag
			if descTag := field.Tag.Get("desc"); descTag != "" {
				fieldSchema.Description = descTag
			} else if yamlTag := field.Tag.Get("yaml"); yamlTag != "" {
				parts := strings.Split(yamlTag, ",")
				if len(parts) > 0 && parts[0] != "" {
					fieldSchema.Description = "Config field: " + parts[0]
				}
			}

			schema.Properties[fieldName] = fieldSchema
		}

	case reflect.Slice, reflect.Array:
		schema.Type = "array"
		schema.Items = generateSchemaFromType(t.Elem())

	case reflect.Map:
		schema.Type = "object"
		schema.AdditionalProperties = generateSchemaFromType(t.Elem())

	case reflect.String:
		schema.Type = "string"

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		schema.Type = "integer"

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		schema.Type = "integer"

	case reflect.Float32, reflect.Float64:
		schema.Type = "number"

	case reflect.Bool:
		schema.Type = "boolean"

	case reflect.Interface:
		// For interface{}/any, allow any type
		schema.Type = ""

	case reflect.Complex64, reflect.Complex128:
		schema.Type = "string"

	case reflect.Chan, reflect.Func, reflect.UnsafePointer, reflect.Invalid:
		// These types are not representable in JSON Schema
		schema.Type = ""

	case reflect.Pointer:
		// Already handled above, but included for exhaustiveness
		return generateSchemaFromType(t.Elem())
	}

	return schema
}

func getJSONFieldName(field *reflect.StructField, jsonTag string) string {
	if jsonTag != "" {
		parts := strings.Split(jsonTag, ",")
		if parts[0] != "" {
			return parts[0]
		}
	}

	return field.Name
}
