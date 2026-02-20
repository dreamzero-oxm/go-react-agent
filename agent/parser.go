package agent

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ResponseParser defines the interface for parsing LLM responses into
// structured ReActResponse objects.
type ResponseParser interface {
	// Parse parses the LLM response string into a ReActResponse.
	Parse(response string) (*ReActResponse, error)
}

// JSONParser implements ResponseParser for JSON-based responses.
type JSONParser struct{}

// NewJSONParser creates a new JSONParser instance.
func NewJSONParser() *JSONParser {
	return &JSONParser{}
}

// Parse parses the LLM response string into a ReActResponse
func (j *JSONParser) Parse(response string) (*ReActResponse, error) {
	// Clean response: remove markdown code blocks if present
	cleaned := j.cleanResponse(response)

	var result ReActResponse
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return nil, &ParseError{
			RawResponse: response,
			Err:         err,
			Context:     "JSON unmarshal",
		}
	}

	// Validate response
	if err := j.validate(&result); err != nil {
		return nil, &ParseError{
			RawResponse: response,
			Err:         err,
			Context:     "validation",
		}
	}

	return &result, nil
}

// cleanResponse removes markdown code blocks if present
func (j *JSONParser) cleanResponse(response string) string {
	trimmed := strings.TrimSpace(response)
	if strings.HasPrefix(trimmed, "```") {
		// Find the end of the first line
		firstNewline := strings.Index(trimmed, "\n")
		if firstNewline != -1 {
			trimmed = strings.TrimSpace(trimmed[firstNewline+1:])
		}
		// Remove closing ```
		if idx := strings.LastIndex(trimmed, "```"); idx != -1 {
			trimmed = strings.TrimSpace(trimmed[:idx])
		}
	}
	return trimmed
}

// validate validates the parsed ReActResponse
func (j *JSONParser) validate(result *ReActResponse) error {
	// Must have at least one thought
	if len(result.Thoughts) == 0 {
		return fmt.Errorf("response must contain at least one thought")
	}

	// Action and Answer are mutually exclusive
	if result.Action != nil && result.Answer != "" {
		return fmt.Errorf("action and answer are mutually exclusive")
	}

	// If done is true, answer must be provided
	if result.Done && result.Answer == "" {
		return fmt.Errorf("done=true requires an answer")
	}

	// If answer is provided, done should be true
	if result.Answer != "" && !result.Done {
		return fmt.Errorf("answer provided but done=false")
	}

	return nil
}

// ParseError provides detailed parsing error information.
type ParseError struct {
	RawResponse string // The original response that failed to parse
	Err         error  // The underlying error
	Context     string // The context where the error occurred
}

// Error returns the error message with context.
func (e *ParseError) Error() string {
	return fmt.Sprintf("parse error at %s: %v\nRaw response: %q", e.Context, e.Err, e.RawResponse)
}

// Unwrap returns the underlying error for error wrapping.
func (e *ParseError) Unwrap() error {
	return e.Err
}
