package agent

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// Test basic struct parsing
func TestStructParser_BasicTypes(t *testing.T) {
	type BasicStruct struct {
		Name  string  `json:"name" agent:"desc:Name of the person"`
		Age   int     `json:"age" agent:"desc:Age in years"`
		Score float64 `json:"score" agent:"desc:Test score"`
		Active bool   `json:"active" agent:"desc:Is active"`
	}

	parser := NewStructParser()
	schema, err := parser.ParseStruct(reflect.TypeOf(BasicStruct{}))
	if err != nil {
		t.Fatalf("ParseStruct failed: %v", err)
	}

	if schema.Type != "object" {
		t.Errorf("Expected type 'object', got '%s'", schema.Type)
	}

	// Check properties
	expectedProps := []string{"name", "age", "score", "active"}
	for _, prop := range expectedProps {
		if _, exists := schema.Properties[prop]; !exists {
			t.Errorf("Missing property: %s", prop)
		}
	}

	// Check property types
	if schema.Properties["name"].Type != "string" {
		t.Errorf("Expected name type 'string', got '%s'", schema.Properties["name"].Type)
	}
	if schema.Properties["age"].Type != "integer" {
		t.Errorf("Expected age type 'integer', got '%s'", schema.Properties["age"].Type)
	}
	if schema.Properties["score"].Type != "number" {
		t.Errorf("Expected score type 'number', got '%s'", schema.Properties["score"].Type)
	}
	if schema.Properties["active"].Type != "boolean" {
		t.Errorf("Expected active type 'boolean', got '%s'", schema.Properties["active"].Type)
	}
}

// Test agent tag parsing with various options
func TestStructParser_AgentTagParsing(t *testing.T) {
	type TagStruct struct {
		Name     string `json:"name" agent:"desc:Full name;required:true;default:Anonymous"`
		Age      int    `json:"age" agent:"desc:Age;required:false;range:0,150"`
		Status   string `json:"status" agent:"desc:User status;enum:active,inactive,pending"`
	}

	parser := NewStructParser()
	schema, err := parser.ParseStruct(reflect.TypeOf(TagStruct{}))
	if err != nil {
		t.Fatalf("ParseStruct failed: %v", err)
	}

	// Check required
	if len(schema.Required) != 1 || schema.Required[0] != "name" {
		t.Errorf("Expected required field 'name', got %v", schema.Required)
	}

	// Check default value
	if schema.Properties["name"].Default != "Anonymous" {
		t.Errorf("Expected default 'Anonymous', got '%v'", schema.Properties["name"].Default)
	}

	// Check description contains range hint
	if !strings.Contains(schema.Properties["age"].Description, "Range") {
		t.Errorf("Expected age description to contain range hint")
	}

	// Check enum
	if len(schema.Properties["status"].Enum) != 3 {
		t.Errorf("Expected 3 enum values, got %d", len(schema.Properties["status"].Enum))
	}
}

// Test nested struct parsing
func TestStructParser_NestedStruct(t *testing.T) {
	type Address struct {
		Street  string `json:"street" agent:"desc:Street address"`
		City    string `json:"city" agent:"desc:City name;required:true"`
		Country string `json:"country" agent:"desc:Country name"`
	}

	type Person struct {
		Name    string  `json:"name" agent:"desc:Full name"`
		Address Address `json:"address" agent:"desc:Postal address"`
	}

	parser := NewStructParser()
	schema, err := parser.ParseStruct(reflect.TypeOf(Person{}))
	if err != nil {
		t.Fatalf("ParseStruct failed: %v", err)
	}

	// Check nested property
	addressProp, exists := schema.Properties["address"]
	if !exists {
		t.Fatal("Missing property: address")
	}

	if addressProp.Type != "object" {
		t.Errorf("Expected address type 'object', got '%s'", addressProp.Type)
	}

	// Check nested properties
	if len(addressProp.Properties) != 3 {
		t.Errorf("Expected 3 nested properties, got %d", len(addressProp.Properties))
	}

	if _, exists := addressProp.Properties["street"]; !exists {
		t.Error("Missing nested property: street")
	}
}

// Test slice and array types
func TestStructParser_SliceTypes(t *testing.T) {
	type SliceStruct struct {
		Tags     []string `json:"tags" agent:"desc:List of tags"`
		Numbers  []int    `json:"numbers" agent:"desc:List of numbers"`
	}

	parser := NewStructParser()
	schema, err := parser.ParseStruct(reflect.TypeOf(SliceStruct{}))
	if err != nil {
		t.Fatalf("ParseStruct failed: %v", err)
	}

	// Check array types
	if schema.Properties["tags"].Type != "array" {
		t.Errorf("Expected tags type 'array', got '%s'", schema.Properties["tags"].Type)
	}

	if schema.Properties["tags"].Items == nil {
		t.Error("Expected items in tags array")
	}

	if schema.Properties["tags"].Items.Type != "string" {
		t.Errorf("Expected tags items type 'string', got '%s'", schema.Properties["tags"].Items.Type)
	}

	if schema.Properties["numbers"].Items.Type != "integer" {
		t.Errorf("Expected numbers items type 'integer', got '%s'", schema.Properties["numbers"].Items.Type)
	}
}

// Test map types
func TestStructParser_MapTypes(t *testing.T) {
	type MapStruct struct {
		Metadata map[string]interface{} `json:"metadata" agent:"desc:Additional metadata"`
	}

	parser := NewStructParser()
	schema, err := parser.ParseStruct(reflect.TypeOf(MapStruct{}))
	if err != nil {
		t.Fatalf("ParseStruct failed: %v", err)
	}

	if schema.Properties["metadata"].Type != "object" {
		t.Errorf("Expected metadata type 'object', got '%s'", schema.Properties["metadata"].Type)
	}
}

// Test JSON schema generation
func TestStructParser_ToJSONSchema(t *testing.T) {
	type SimpleStruct struct {
		Name string `json:"name" agent:"desc:Name;required:true"`
		Age  int    `json:"age" agent:"desc:Age"`
	}

	parser := NewStructParser()
	schema, err := parser.ParseStruct(reflect.TypeOf(SimpleStruct{}))
	if err != nil {
		t.Fatalf("ParseStruct failed: %v", err)
	}

	jsonSchema := parser.ToJSONSchema(schema)

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonSchema), &parsed); err != nil {
		t.Errorf("Generated schema is not valid JSON: %v", err)
	}

	// Check structure
	if parsed["type"] != "object" {
		t.Errorf("Expected type 'object' in JSON schema")
	}

	props, ok := parsed["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Missing properties in JSON schema")
	}

	if _, exists := props["name"]; !exists {
		t.Error("Missing 'name' property in JSON schema")
	}
}

// Test prompt instructions generation
func TestStructParser_GeneratePromptInstructions(t *testing.T) {
	type WeatherStruct struct {
		City        string  `json:"city" agent:"desc:City name;required:true"`
		Temperature float64 `json:"temperature" agent:"desc:Temperature in Celsius;required:true"`
		Condition   string  `json:"condition" agent:"desc:Weather condition;enum:sunny,cloudy,rainy"`
	}

	parser := NewStructParser()
	schema, err := parser.ParseStruct(reflect.TypeOf(WeatherStruct{}))
	if err != nil {
		t.Fatalf("ParseStruct failed: %v", err)
	}

	instructions := parser.GeneratePromptInstructions("Weather", schema)

	// Check key components
	if !strings.Contains(instructions, "## Structured Output Format") {
		t.Error("Missing output format section")
	}

	if !strings.Contains(instructions, "city") {
		t.Error("Missing 'city' field in instructions")
	}

	if !strings.Contains(instructions, "temperature") {
		t.Error("Missing 'temperature' field in instructions")
	}

	if !strings.Contains(instructions, "[Allowed:") {
		t.Error("Missing enum allowed values hint")
	}

	if !strings.Contains(instructions, "(required)") {
		t.Error("Missing required field hint")
	}
}

// Test max nesting depth
func TestStructParser_MaxNestingDepth(t *testing.T) {
	type Level3 struct {
		Value string `json:"value"`
	}
	type Level2 struct {
		L3 Level3 `json:"l3"`
	}
	type Level1 struct {
		L2 Level2 `json:"l2"`
	}

	parser := NewStructParser()
	parser.SetMaxNestingDepth(0) // Set to 0 so even Level2 (depth=1 when processing its fields) will fail

	_, err := parser.ParseStruct(reflect.TypeOf(Level1{}))
	if err == nil {
		t.Error("Expected error for exceeding max nesting depth")
	} else if !strings.Contains(err.Error(), "maximum nesting depth") {
		t.Errorf("Expected max depth error, got: %v", err)
	}
}

// Test unexported fields are skipped
func TestStructParser_UnexportedFieldsSkipped(t *testing.T) {
	type TestStruct struct {
		Name     string `json:"name"`
		internal string // unexported field
	}

	parser := NewStructParser()
	schema, err := parser.ParseStruct(reflect.TypeOf(TestStruct{}))
	if err != nil {
		t.Fatalf("ParseStruct failed: %v", err)
	}

	if len(schema.Properties) != 1 {
		t.Errorf("Expected 1 property (unexported should be skipped), got %d", len(schema.Properties))
	}

	if _, exists := schema.Properties["internal"]; exists {
		t.Error("Unexported field should be skipped")
	}
}

// Test fields with json tag "-" are skipped
func TestStructParser_DashTagSkipped(t *testing.T) {
	type TestStruct struct {
		Name   string `json:"name"`
		SkipMe string `json:"-"`
	}

	parser := NewStructParser()
	schema, err := parser.ParseStruct(reflect.TypeOf(TestStruct{}))
	if err != nil {
		t.Fatalf("ParseStruct failed: %v", err)
	}

	if len(schema.Properties) != 1 {
		t.Errorf("Expected 1 property (dash tag should be skipped), got %d", len(schema.Properties))
	}

	if _, exists := schema.Properties["skipMe"]; exists {
		t.Error("Field with json tag '-' should be skipped")
	}
}

// Test default description when no agent tag
func TestStructParser_DefaultDescription(t *testing.T) {
	type TestStruct struct {
		Name string `json:"name"`
	}

	parser := NewStructParser()
	schema, err := parser.ParseStruct(reflect.TypeOf(TestStruct{}))
	if err != nil {
		t.Fatalf("ParseStruct failed: %v", err)
	}

	if schema.Properties["name"].Description != "name" {
		t.Errorf("Expected default description 'name', got '%s'", schema.Properties["name"].Description)
	}
}
