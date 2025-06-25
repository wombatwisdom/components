package spec

import "encoding/json"

// ComponentSpec defines the schema and metadata for a component type.
// This enables benthos-compatible component registration and validation.
type ComponentSpec interface {
	// Name returns the component name used for registration
	Name() string

	// Summary returns a brief description of the component
	Summary() string

	// Description returns detailed documentation for the component
	Description() string

	// InputConfigSchema returns the JSON schema for input component configuration
	InputConfigSchema() string

	// OutputConfigSchema returns the JSON schema for output component configuration
	OutputConfigSchema() string

	// SystemConfigSchema returns the JSON schema for system configuration
	SystemConfigSchema() string
}

// SchemaField represents a configuration field with validation rules.
type SchemaField struct {
	Name        string
	Type        string
	Description string
	Required    bool
	Default     any
	Examples    []any
}

// ConfigSchema provides a programmatic way to build component schemas.
// This is useful for components that need dynamic schema generation.
type ConfigSchema interface {
	// AddField adds a configuration field to the schema
	AddField(field SchemaField) ConfigSchema

	// ToJSON returns the JSON schema representation
	ToJSON() (string, error)
}

// NewConfigSchema creates a new configuration schema builder.
func NewConfigSchema() ConfigSchema {
	return &configSchema{
		fields: make(map[string]SchemaField),
	}
}

type configSchema struct {
	fields map[string]SchemaField
}

func (c *configSchema) AddField(field SchemaField) ConfigSchema {
	c.fields[field.Name] = field
	return c
}

func (c *configSchema) ToJSON() (string, error) {
	schema := map[string]any{
		"type":       "object",
		"properties": make(map[string]any),
		"required":   []string{},
	}

	properties := schema["properties"].(map[string]any)
	required := []string{}

	for name, field := range c.fields {
		prop := map[string]any{
			"type":        field.Type,
			"description": field.Description,
		}

		if field.Default != nil {
			prop["default"] = field.Default
		}

		if len(field.Examples) > 0 {
			prop["examples"] = field.Examples
		}

		properties[name] = prop

		if field.Required {
			required = append(required, name)
		}
	}

	schema["required"] = required

	// Convert to JSON
	jsonBytes, err := json.Marshal(schema)
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}
