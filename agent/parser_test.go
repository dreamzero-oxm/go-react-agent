package agent

import (
	"strings"
	"testing"
)

func TestJSONParserValidResponse(t *testing.T) {
	parser := NewJSONParser()

	tests := []struct {
		name     string
		response string
		wantDone bool
		wantAns  string
	}{
		{
			name:     "Action response",
			response: `{"thoughts":[{"content":"thinking"}],"action":{"name":"tool","input":{"key":"value"}},"answer":"","done":false}`,
			wantDone: false,
		},
		{
			name:     "Answer response",
			response: `{"thoughts":[{"content":"done"}],"action":null,"answer":"final answer","done":true}`,
			wantDone: true,
			wantAns:  "final answer",
		},
		{
			name:     "Multiple thoughts",
			response: `{"thoughts":[{"content":"first"},{"content":"second"}],"action":null,"answer":"answer","done":true}`,
			wantDone: true,
			wantAns:  "answer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse(tt.response)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if result.Done != tt.wantDone {
				t.Errorf("Done = %v, want %v", result.Done, tt.wantDone)
			}
			if tt.wantAns != "" && result.Answer != tt.wantAns {
				t.Errorf("Answer = %v, want %v", result.Answer, tt.wantAns)
			}
		})
	}
}

func TestJSONParserInvalidJSON(t *testing.T) {
	parser := NewJSONParser()

	_, err := parser.Parse("not json")
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestJSONParserNoThoughts(t *testing.T) {
	parser := NewJSONParser()

	_, err := parser.Parse(`{"thoughts":[],"action":null,"answer":"answer","done":true}`)
	if err == nil {
		t.Error("Expected error for empty thoughts")
	}
}

func TestJSONParserBothActionAndAnswer(t *testing.T) {
	parser := NewJSONParser()

	_, err := parser.Parse(`{"thoughts":[{"content":"test"}],"action":{"name":"tool","input":{}},"answer":"answer","done":false}`)
	if err == nil {
		t.Error("Expected error for both action and answer")
	}
}

func TestJSONParserDoneWithoutAnswer(t *testing.T) {
	parser := NewJSONParser()

	_, err := parser.Parse(`{"thoughts":[{"content":"test"}],"action":null,"answer":"","done":true}`)
	if err == nil {
		t.Error("Expected error for done=true without answer")
	}
}

func TestJSONParserAnswerWithoutDone(t *testing.T) {
	parser := NewJSONParser()

	_, err := parser.Parse(`{"thoughts":[{"content":"test"}],"action":null,"answer":"answer","done":false}`)
	if err == nil {
		t.Error("Expected error for answer provided but done=false")
	}
}

func TestJSONParserMarkdownCodeBlock(t *testing.T) {
	parser := NewJSONParser()

	response := "```json\n" + `{"thoughts":[{"content":"test"}],"action":null,"answer":"answer","done":true}` + "\n```"

	result, err := parser.Parse(response)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if result.Answer != "answer" {
		t.Errorf("Answer = %v, want 'answer'", result.Answer)
	}
}

func TestJSONParserMarkdownCodeBlockWithoutLanguage(t *testing.T) {
	parser := NewJSONParser()

	response := "```\n" + `{"thoughts":[{"content":"test"}],"action":null,"answer":"answer","done":true}` + "\n```"

	result, err := parser.Parse(response)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if result.Answer != "answer" {
		t.Errorf("Answer = %v, want 'answer'", result.Answer)
	}
}

func TestJSONParserParseError(t *testing.T) {
	parser := NewJSONParser()

	_, err := parser.Parse("invalid")
	if err == nil {
		t.Fatal("Expected error")
	}

	parseErr, ok := err.(*ParseError)
	if !ok {
		t.Fatalf("Expected *ParseError, got %T", err)
	}

	if parseErr.Context != "JSON unmarshal" {
		t.Errorf("Context = %v, want 'JSON unmarshal'", parseErr.Context)
	}

	if parseErr.RawResponse != "invalid" {
		t.Errorf("RawResponse = %v, want 'invalid'", parseErr.RawResponse)
	}
}

func TestJSONParserValidationError(t *testing.T) {
	parser := NewJSONParser()

	_, err := parser.Parse(`{"thoughts":[],"action":null,"answer":"answer","done":true}`)
	if err == nil {
		t.Fatal("Expected error")
	}

	parseErr, ok := err.(*ParseError)
	if !ok {
		t.Fatalf("Expected *ParseError, got %T", err)
	}

	if parseErr.Context != "validation" {
		t.Errorf("Context = %v, want 'validation'", parseErr.Context)
	}
}

func TestJSONParserActionWithMultipleInputParams(t *testing.T) {
	parser := NewJSONParser()

	response := `{"thoughts":[{"content":"test"}],"action":{"name":"tool","input":{"param1":"value1","param2":123,"param3":true}},"answer":"","done":false}`

	result, err := parser.Parse(response)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if result.Action.Name != "tool" {
		t.Errorf("Action name = %v, want 'tool'", result.Action.Name)
	}

	if result.Action.Input["param1"] != "value1" {
		t.Errorf("param1 = %v, want 'value1'", result.Action.Input["param1"])
	}

	if result.Action.Input["param2"] != float64(123) {
		t.Errorf("param2 = %v, want 123", result.Action.Input["param2"])
	}

	if result.Action.Input["param3"] != true {
		t.Errorf("param3 = %v, want true", result.Action.Input["param3"])
	}
}

func TestJSONParserWithWhitespace(t *testing.T) {
	parser := NewJSONParser()

	response := `
	{
		"thoughts": [
			{
				"content": "test"
			}
		],
		"action": null,
		"answer": "answer",
		"done": true
	}
	`

	result, err := parser.Parse(response)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if result.Answer != "answer" {
		t.Errorf("Answer = %v, want 'answer'", result.Answer)
	}
}

func TestJSONParserEmptyInput(t *testing.T) {
	parser := NewJSONParser()

	response := `{"thoughts":[{"content":"test"}],"action":{"name":"tool","input":{}},"answer":"","done":false}`

	result, err := parser.Parse(response)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if result.Action.Name != "tool" {
		t.Errorf("Action name = %v, want 'tool'", result.Action.Name)
	}

	if len(result.Action.Input) != 0 {
		t.Errorf("Input = %v, want empty map", result.Action.Input)
	}
}

func TestJSONParserMultipleThoughts(t *testing.T) {
	parser := NewJSONParser()

	response := `{"thoughts":[{"content":"first"},{"content":"second"},{"content":"third"}],"action":null,"answer":"final","done":true}`

	result, err := parser.Parse(response)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(result.Thoughts) != 3 {
		t.Errorf("Thoughts count = %v, want 3", len(result.Thoughts))
	}

	if result.Thoughts[0].Content != "first" {
		t.Errorf("First thought = %v, want 'first'", result.Thoughts[0].Content)
	}

	if result.Thoughts[1].Content != "second" {
		t.Errorf("Second thought = %v, want 'second'", result.Thoughts[1].Content)
	}

	if result.Thoughts[2].Content != "third" {
		t.Errorf("Third thought = %v, want 'third'", result.Thoughts[2].Content)
	}
}

func TestParseErrorUnwrap(t *testing.T) {
	parser := NewJSONParser()

	_, err := parser.Parse("invalid json")
	if err == nil {
		t.Fatal("Expected error")
	}

	// Test Unwrap() works correctly
	unwrapped := err.(interface{ Unwrap() error }).Unwrap()
	if unwrapped == nil {
		t.Error("Unwrap() returned nil, expected original error")
	}

	// Verify it's a JSON syntax error
	if !strings.Contains(unwrapped.Error(), "invalid") && !strings.Contains(unwrapped.Error(), "JSON") {
		t.Errorf("Unwrapped error = %v, expected JSON syntax error", unwrapped)
	}
}
