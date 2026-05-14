package checkconsensusapi

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// compileSchema turns an inline JSON Schema (as a map[string]interface{}) into
// a compiled validator. The map is re-marshaled to JSON because the schema
// library accepts JSON bytes, not Go structs.
func compileSchema(schemaMap map[string]interface{}) (*jsonschema.Schema, error) {
	if len(schemaMap) == 0 {
		return nil, nil
	}

	schemaBytes, err := json.Marshal(schemaMap)
	if err != nil {
		return nil, fmt.Errorf("failed to encode schema as JSON: %w", err)
	}

	c := jsonschema.NewCompiler()
	c.Draft = jsonschema.Draft2020

	if addErr := c.AddResource("inline.json", bytes.NewReader(schemaBytes)); addErr != nil {
		return nil, fmt.Errorf("failed to register schema: %w", addErr)
	}

	compiled, err := c.Compile("inline.json")
	if err != nil {
		return nil, fmt.Errorf("failed to compile schema: %w", err)
	}

	return compiled, nil
}

// validateBytes runs a compiled schema against raw JSON bytes. Returns a list
// of human-readable error messages (empty == valid). Non-JSON payloads return
// a single "invalid JSON" entry.
func validateBytes(schema *jsonschema.Schema, body []byte) []string {
	if schema == nil {
		return nil
	}

	var doc interface{}
	if err := json.Unmarshal(body, &doc); err != nil {
		return []string{fmt.Sprintf("invalid JSON: %v", err)}
	}

	if err := schema.Validate(doc); err != nil {
		return flattenValidationError(err)
	}

	return nil
}

func flattenValidationError(err error) []string {
	if err == nil {
		return nil
	}

	if ve, ok := err.(*jsonschema.ValidationError); ok {
		out := []string{}
		var walk func(e *jsonschema.ValidationError)
		walk = func(e *jsonschema.ValidationError) {
			if len(e.Causes) == 0 {
				loc := e.InstanceLocation
				if loc == "" {
					loc = "/"
				}
				out = append(out, fmt.Sprintf("%s: %s", loc, e.Message))
				return
			}
			for _, c := range e.Causes {
				walk(c)
			}
		}
		walk(ve)

		if len(out) == 0 {
			out = []string{err.Error()}
		}

		return out
	}

	return []string{err.Error()}
}
