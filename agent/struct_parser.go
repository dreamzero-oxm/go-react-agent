package agent

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// StructParser handles struct analysis and schema generation
type StructParser struct {
	maxNestingDepth int
}

// NewStructParser creates a new struct parser
func NewStructParser() *StructParser {
	return &StructParser{
		maxNestingDepth: 5, // Default max nesting depth
	}
}

// SetMaxNestingDepth sets the maximum nesting depth for struct parsing
func (s *StructParser) SetMaxNestingDepth(depth int) {
	s.maxNestingDepth = depth
}

// FieldConfig holds parsed field configuration from agent tags
type FieldConfig struct {
	Description string
	Required    bool
	Default     interface{}
	Range       []interface{} // For numeric ranges
	Enum        []interface{} // For enum values
}

// StructSchema represents the generated schema for a struct
type StructSchema struct {
	Type        string                     `json:"type"`
	Description string                     `json:"description,omitempty"`
	Properties  map[string]*PropertySchema `json:"properties"`
	Required    []string                   `json:"required,omitempty"`
}

// PropertySchema represents a single field's schema
type PropertySchema struct {
	Type        string                     `json:"type"`
	Description string                     `json:"description"`
	Required    bool                       `json:"required,omitempty"`
	Default     interface{}                `json:"default,omitempty"`
	Properties  map[string]*PropertySchema `json:"properties,omitempty"` // For nested structs
	Items       *PropertySchema            `json:"items,omitempty"`      // For slices/arrays
	Enum        []interface{}              `json:"enum,omitempty"`       // For enum values
}

// ParseStruct analyzes a struct type and generates its schema
func (s *StructParser) ParseStruct(typ reflect.Type) (*StructSchema, error) {
	if typ.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct type, got %s", typ.Kind())
	}

	schema := &StructSchema{
		Type:       "object",
		Properties: make(map[string]*PropertySchema),
		Required:   []string{},
	}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Get JSON tag name
		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}

		fieldName := strings.Split(jsonTag, ",")[0]
		if fieldName == "" {
			fieldName = field.Name
		}

		// Parse agent tag
		config := s.parseAgentTag(field.Tag.Get("agent"))

		// Use field name as default description if not provided
		if config.Description == "" {
			config.Description = fieldName
		}

		// Generate property schema
		propSchema, err := s.generatePropertySchema(field.Type, config, 0)
		if err != nil {
			return nil, fmt.Errorf("error processing field %s: %w", field.Name, err)
		}

		schema.Properties[fieldName] = propSchema

		if config.Required {
			schema.Required = append(schema.Required, fieldName)
		}
	}

	return schema, nil
}

// parseAgentTag parses the agent tag string into a FieldConfig
func (s *StructParser) parseAgentTag(tag string) FieldConfig {
	config := FieldConfig{
		Required: false,
	}

	if tag == "" {
		return config
	}

	// Parse format: desc:xxx;required:true;default:yyy;range:0,150;enum:a,b,c
	parts := strings.Split(tag, ";")
	for _, part := range parts {
		kv := strings.SplitN(strings.TrimSpace(part), ":", 2)
		if len(kv) != 2 {
			continue
		}

		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])

		switch key {
		case "desc":
			config.Description = value
		case "required":
			config.Required = strings.ToLower(value) == "true"
		case "default":
			config.Default = value
		case "range":
			// Parse range like "0,150"
			rangeParts := strings.Split(value, ",")
			if len(rangeParts) == 2 {
				config.Range = []interface{}{strings.TrimSpace(rangeParts[0]), strings.TrimSpace(rangeParts[1])}
			}
		case "enum":
			enumParts := strings.Split(value, ",")
			config.Enum = make([]interface{}, len(enumParts))
			for i, part := range enumParts {
				config.Enum[i] = strings.TrimSpace(part)
			}
		}
	}

	return config
}

// generatePropertySchema creates a property schema for a given type
func (s *StructParser) generatePropertySchema(typ reflect.Type, config FieldConfig, depth int) (*PropertySchema, error) {
	if depth > s.maxNestingDepth {
		return nil, fmt.Errorf("maximum nesting depth %d exceeded", s.maxNestingDepth)
	}

	schema := &PropertySchema{
		Description: config.Description,
		Required:    config.Required,
		Default:     config.Default,
	}

	if len(config.Enum) > 0 {
		schema.Enum = config.Enum
	}

	if len(config.Range) > 0 {
		if schema.Description == "" {
			schema.Description = fmt.Sprintf("Range: [%v, %v]", config.Range[0], config.Range[1])
		} else {
			schema.Description = fmt.Sprintf("%s (Range: [%v, %v])", schema.Description, config.Range[0], config.Range[1])
		}
	}

	switch typ.Kind() {
	case reflect.String:
		schema.Type = "string"

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		schema.Type = "integer"

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		schema.Type = "integer"

	case reflect.Float32, reflect.Float64:
		schema.Type = "number"

	case reflect.Bool:
		schema.Type = "boolean"

	case reflect.Struct:
		// Handle nested struct or special types
		if typ.String() == "time.Time" {
			schema.Type = "string"
			if schema.Description != "" {
				schema.Description += " (ISO 8601 datetime)"
			} else {
				schema.Description = "ISO 8601 datetime"
			}
		} else {
			schema.Type = "object"
			schema.Properties = make(map[string]*PropertySchema)

			for i := 0; i < typ.NumField(); i++ {
				field := typ.Field(i)
				if !field.IsExported() {
					continue
				}

				jsonTag := field.Tag.Get("json")
				if jsonTag == "" || jsonTag == "-" {
					continue
				}

				fieldName := strings.Split(jsonTag, ",")[0]
				if fieldName == "" {
					fieldName = field.Name
				}

				fieldConfig := s.parseAgentTag(field.Tag.Get("agent"))
				if fieldConfig.Description == "" {
					fieldConfig.Description = fieldName
				}

				prop, err := s.generatePropertySchema(field.Type, fieldConfig, depth+1)
				if err != nil {
					return nil, err
				}

				schema.Properties[fieldName] = prop
			}
		}

	case reflect.Slice, reflect.Array:
		schema.Type = "array"
		elemType := typ.Elem()

		// Recursively generate element schema
		elemConfig := FieldConfig{Description: "array element"}
		items, err := s.generatePropertySchema(elemType, elemConfig, depth+1)
		if err != nil {
			return nil, err
		}
		schema.Items = items

	case reflect.Map:
		schema.Type = "object"
		// For maps, describe the value type in the description
		if typ.Elem().Kind() != reflect.Interface {
			if schema.Description != "" {
				schema.Description += fmt.Sprintf(" (Map with values of type %s)", typ.Elem().Name())
			} else {
				schema.Description = fmt.Sprintf("Map with values of type %s", typ.Elem().Name())
			}
		} else {
			if schema.Description == "" {
				schema.Description = "Map with string keys and any values"
			}
		}

	case reflect.Interface:
		schema.Type = "object"
		if schema.Description == "" {
			schema.Description = "any JSON value"
		}

	case reflect.Pointer:
		// Dereference and process
		return s.generatePropertySchema(typ.Elem(), config, depth)

	default:
		return nil, fmt.Errorf("unsupported type: %s", typ.Kind())
	}

	return schema, nil
}

// ToJSONSchema converts the struct schema to a JSON Schema string
func (s *StructParser) ToJSONSchema(schema *StructSchema) string {
	data, _ := json.MarshalIndent(schema, "", "  ")
	return string(data)
}

// GeneratePromptInstructions generates natural language instructions for the struct
func (s *StructParser) GeneratePromptInstructions(structName string, schema *StructSchema) string {
	var builder strings.Builder

	builder.WriteString("## Structured Output Format\n\n")
	builder.WriteString("Your final response must be a valid JSON object matching this structure:\n\n")
	builder.WriteString(fmt.Sprintf("```json\n%s\n```\n\n", s.ToJSONSchema(schema)))

	builder.WriteString("### Field Descriptions:\n\n")
	for name, prop := range schema.Properties {
		required := ""
		if prop.Required {
			required = " (required)"
		}
		enumHint := ""
		if len(prop.Enum) > 0 {
			enumHint = fmt.Sprintf(" [Allowed: %v]", prop.Enum)
		}
		builder.WriteString(fmt.Sprintf("- **%s** (%s)%s%s: %s\n", name, prop.Type, required, enumHint, prop.Description))
	}

	firstKey := getFirstKey(schema.Properties)
	builder.WriteString(fmt.Sprintf(`
### Output Instructions

When providing your final answer, set the "answer" field to a JSON string containing your structured response.

For example:
%sjson
{
  "thoughts": [{"content": "I have gathered all the information"}],
  "action": null,
  "answer": "{\\"%s\\": \\"value\\"}",
  "done": true
}
%s

The "answer" field must contain a valid JSON string that can be parsed into the target structure.
`, "```", firstKey, "```"))

	return builder.String()
}

// getFirstKey returns the first key from the properties map
func getFirstKey(props map[string]*PropertySchema) string {
	for k := range props {
		return k
	}
	return "field"
}
