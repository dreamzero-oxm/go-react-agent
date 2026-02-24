package protocol

import (
	"encoding/json"
	"fmt"
)

// Version is the JSON-RPC version supported by this package.
const (
	Version = "2.0"
)

// JSONRPCMessage represents a JSON-RPC 2.0 message.
// It can be a request, a response, or a notification.
type JSONRPCMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC 2.0 error object.
type JSONRPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// JSON-RPC 2.0 error codes.
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)

// NewError creates a new JSONRPCError.
//
// Parameters:
//   - code: The error code.
//   - message: The error message.
//   - data: Optional additional error data.
//
// Returns:
//   - *JSONRPCError: The created error object.
func NewError(code int, message string, data interface{}) *JSONRPCError {
	rpcErr := &JSONRPCError{
		Code:    code,
		Message: message,
	}
	if data != nil {
		if raw, err := json.Marshal(data); err == nil {
			rpcErr.Data = raw
		}
	}
	return rpcErr
}

// Error implements the error interface for JSONRPCError.
//
// Returns:
//   - string: The error message, including data if present.
func (e *JSONRPCError) Error() string {
	if e.Data != nil {
		return fmt.Sprintf("%s: %s", e.Message, string(e.Data))
	}
	return e.Message
}

// ParseRequest parses a byte slice into a JSONRPCMessage.
//
// Parameters:
//   - data: The raw JSON data.
//
// Returns:
//   - *JSONRPCMessage: The parsed message.
//   - error: An error if parsing fails or the version is invalid.
func ParseRequest(data []byte) (*JSONRPCMessage, error) {
	var msg JSONRPCMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, NewError(ParseError, "Parse error", err.Error())
	}

	if msg.JSONRPC != Version {
		return nil, NewError(InvalidRequest, "Invalid JSON-RPC version", nil)
	}

	return &msg, nil
}

// NewRequest creates a new JSON-RPC request message.
//
// Parameters:
//   - id: The request ID.
//   - method: The method name.
//   - params: The method parameters.
//
// Returns:
//   - *JSONRPCMessage: The created request message.
//   - error: An error if marshaling parameters fails.
func NewRequest(id interface{}, method string, params interface{}) (*JSONRPCMessage, error) {
	var paramsData json.RawMessage
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal params: %w", err)
		}
		paramsData = data
	}

	return &JSONRPCMessage{
		JSONRPC: Version,
		ID:      id,
		Method:  method,
		Params:  paramsData,
	}, nil
}

// NewResponse creates a new JSON-RPC response message.
//
// Parameters:
//   - id: The request ID this response corresponds to.
//   - result: The result of the successful execution (optional).
//   - err: The error object if execution failed (optional).
//
// Returns:
//   - *JSONRPCMessage: The created response message.
//   - error: An error if marshaling the result fails.
func NewResponse(id interface{}, result interface{}, err *JSONRPCError) (*JSONRPCMessage, error) {
	var resultData json.RawMessage
	if result != nil {
		data, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal result: %w", err)
		}
		resultData = data
	}

	return &JSONRPCMessage{
		JSONRPC: Version,
		ID:      id,
		Result:  resultData,
		Error:   err,
	}, nil
}

// NewNotification creates a new JSON-RPC notification message.
//
// Parameters:
//   - method: The method name.
//   - params: The notification parameters.
//
// Returns:
//   - *JSONRPCMessage: The created notification message.
//   - error: An error if marshaling parameters fails.
func NewNotification(method string, params interface{}) (*JSONRPCMessage, error) {
	var paramsData json.RawMessage
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal params: %w", err)
		}
		paramsData = data
	}

	return &JSONRPCMessage{
		JSONRPC: Version,
		Method:  method,
		Params:  paramsData,
	}, nil
}

// Marshal converts the JSONRPCMessage to its JSON representation.
//
// Returns:
//   - []byte: The JSON encoded message.
//   - error: An error if marshaling fails.
func (m *JSONRPCMessage) Marshal() ([]byte, error) {
	return json.Marshal(m)
}

// UnmarshalParams unmarshals the Params field into the provided value.
//
// Parameters:
//   - v: The pointer to the value where parameters will be unmarshaled.
//
// Returns:
//   - error: An error if unmarshaling fails.
func (m *JSONRPCMessage) UnmarshalParams(v interface{}) error {
	if m.Params == nil {
		return nil
	}
	return json.Unmarshal(m.Params, v)
}

// UnmarshalResult unmarshals the Result field into the provided value.
//
// Parameters:
//   - v: The pointer to the value where result will be unmarshaled.
//
// Returns:
//   - error: An error if unmarshaling fails.
func (m *JSONRPCMessage) UnmarshalResult(v interface{}) error {
	if m.Result == nil {
		return nil
	}
	return json.Unmarshal(m.Result, v)
}

// IsRequest checks if the message is a request.
//
// Returns:
//   - bool: True if the message is a request (has Method and ID).
func (m *JSONRPCMessage) IsRequest() bool {
	return m.Method != "" && m.ID != nil
}

// IsNotification checks if the message is a notification.
//
// Returns:
//   - bool: True if the message is a notification (has Method but no ID).
func (m *JSONRPCMessage) IsNotification() bool {
	return m.Method != "" && m.ID == nil
}

// IsResponse checks if the message is a response.
//
// Returns:
//   - bool: True if the message is a response (has ID and Result or Error).
func (m *JSONRPCMessage) IsResponse() bool {
	return m.ID != nil && (m.Result != nil || m.Error != nil)
}
