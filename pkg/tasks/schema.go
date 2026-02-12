package tasks

import (
	"encoding/json"
	"fmt"
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

const schemaTypeString = "string"

// JSONSchema represents a JSON Schema object.
type JSONSchema struct {
	Type                 string                 `json:"type,omitempty"`
	Format               string                 `json:"format,omitempty"`
	Description          string                 `json:"description,omitempty"`
	Properties           map[string]*JSONSchema `json:"properties,omitempty"`
	PropertyOrder        []string               `json:"propertyOrder,omitempty"`
	Required             []string               `json:"required,omitempty"`
	Items                *JSONSchema            `json:"items,omitempty"`
	Default              any                    `json:"default,omitempty"`
	Enum                 []any                  `json:"enum,omitempty"`
	AdditionalProperties *JSONSchema            `json:"additionalProperties,omitempty"`
	// RequireGroup specifies requirement groups for this field.
	// Format: "A" or "A.1" where A is the group and .1 is the subgroup.
	// Fields in same subgroup must all be present together.
	// Multiple subgroups (A.1, A.2) are alternatives - one must be satisfied.
	RequireGroup string `json:"requireGroup,omitempty"`
}

// GenerateJSONSchema generates a JSON Schema from a Go struct via reflection.
// The value v is used both for type information and for extracting default values.
func GenerateJSONSchema(v any) (json.RawMessage, error) {
	schema := generateSchema(reflect.TypeOf(v), reflect.ValueOf(v))

	return json.Marshal(schema)
}

// invalidValue is a zero reflect.Value used when no default value is available.
var invalidValue reflect.Value

func generateSchema(t reflect.Type, v reflect.Value) *JSONSchema {
	// Handle pointer types
	if t.Kind() == reflect.Pointer {
		t = t.Elem()

		if v.IsValid() && !v.IsNil() {
			v = v.Elem()
		} else {
			v = invalidValue
		}
	}

	schema := &JSONSchema{}

	switch t.Kind() {
	case reflect.Struct:
		schema.Type = "object"
		schema.Properties = make(map[string]*JSONSchema, t.NumField())
		schema.PropertyOrder = make([]string, 0, t.NumField())

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

			// Skip deprecated fields
			if deprecatedTag := field.Tag.Get("deprecated"); deprecatedTag == "true" {
				continue
			}

			fieldName := getJSONFieldName(&field, jsonTag)

			// Get the field value for defaults (if available)
			var fieldValue reflect.Value
			if v.IsValid() {
				fieldValue = v.Field(i)
			}

			// Generate schema for field type
			fieldSchema := generateSchema(field.Type, fieldValue)

			// Add description from desc tag, falling back to yaml tag
			if descTag := field.Tag.Get("desc"); descTag != "" {
				fieldSchema.Description = descTag
			} else if yamlTag := field.Tag.Get("yaml"); yamlTag != "" {
				parts := strings.Split(yamlTag, ",")
				if len(parts) > 0 && parts[0] != "" {
					fieldSchema.Description = "Config field: " + parts[0]
				}
			}

			// Add requirement group from require tag
			if requireTag := field.Tag.Get("require"); requireTag != "" {
				fieldSchema.RequireGroup = requireTag
			}

			// Add format annotation from format tag
			if formatTag := field.Tag.Get("format"); formatTag != "" {
				fieldSchema.Format = formatTag
			}

			// Extract default value from the config instance
			if fieldValue.IsValid() {
				if def := extractDefault(fieldValue); def != nil {
					fieldSchema.Default = def
				}
			}

			schema.Properties[fieldName] = fieldSchema
			schema.PropertyOrder = append(schema.PropertyOrder, fieldName)
		}

		// Structs with no exported fields (e.g. big.Int) serialize as
		// simple values in JSON, so represent them as strings rather
		// than empty objects.
		if len(schema.Properties) == 0 {
			schema.Type = schemaTypeString
			schema.Properties = nil
			schema.PropertyOrder = nil
		}

	case reflect.Slice, reflect.Array:
		schema.Type = "array"
		schema.Items = generateSchema(t.Elem(), invalidValue)

	case reflect.Map:
		schema.Type = "object"
		schema.AdditionalProperties = generateSchema(t.Elem(), invalidValue)

	case reflect.String:
		schema.Type = schemaTypeString

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
		schema.Type = schemaTypeString

	case reflect.Chan, reflect.Func, reflect.UnsafePointer, reflect.Invalid:
		// These types are not representable in JSON Schema
		schema.Type = ""

	case reflect.Pointer:
		// Already handled above, but included for exhaustiveness
		return generateSchema(t.Elem(), invalidValue)
	}

	return schema
}

// extractDefault extracts a JSON-serializable default value from a reflect.Value.
// Returns nil for zero values or types that should not have defaults.
func extractDefault(v reflect.Value) any {
	if !v.IsValid() {
		return nil
	}

	// Dereference pointers
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}

		v = v.Elem()
	}

	// Skip zero values (they represent "no default")
	if v.IsZero() {
		return nil
	}

	switch v.Kind() {
	case reflect.String:
		return v.String()

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int()

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint()

	case reflect.Float32, reflect.Float64:
		return v.Float()

	case reflect.Bool:
		return v.Bool()

	case reflect.Struct:
		// For structs with no exported fields (like big.Int), use
		// fmt.Stringer if available to get a string representation.
		iface := v.Interface()
		if s, ok := iface.(fmt.Stringer); ok {
			return s.String()
		}

		return nil

	case reflect.Invalid, reflect.Complex64, reflect.Complex128, reflect.Array,
		reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Pointer, reflect.Slice, reflect.UnsafePointer:
		return nil
	}

	return nil
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
