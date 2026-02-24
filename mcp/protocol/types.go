package protocol

import (
	"encoding/json"
)

// InitializeRequest represents the initialization request payload.
type InitializeRequest struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ClientCapabilities `json:"capabilities"`
	ClientInfo      Implementation     `json:"clientInfo"`
}

// InitializeResponse represents the initialization response payload.
type InitializeResponse struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      Implementation     `json:"serverInfo"`
	Instructions    string             `json:"instructions,omitempty"`
}

// InitializedNotification represents the initialized notification payload.
type InitializedNotification struct{}

// ClientCapabilities describes the capabilities supported by the client.
type ClientCapabilities struct {
	Roots    *RootsCapability    `json:"roots,omitempty"`
	Sampling *SamplingCapability `json:"sampling,omitempty"`
}

// ServerCapabilities describes the capabilities supported by the server.
type ServerCapabilities struct {
	Tools     *ToolsCapability     `json:"tools,omitempty"`
	Resources *ResourcesCapability `json:"resources,omitempty"`
	Prompts   *PromptsCapability   `json:"prompts,omitempty"`
	Logging   *LoggingCapability   `json:"logging,omitempty"`
}

// Implementation describes the client or server implementation details.
type Implementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// RootsCapability indicates support for root listing.
type RootsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// SamplingCapability indicates support for sampling.
type SamplingCapability struct{}

// ToolsCapability indicates support for tools.
type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ResourcesCapability indicates support for resources.
type ResourcesCapability struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

// PromptsCapability indicates support for prompts.
type PromptsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// LoggingCapability indicates support for logging.
type LoggingCapability struct{}

// Tool represents an available tool.
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// ToolListRequest represents a request to list tools.
type ToolListRequest struct {
	Cursor string `json:"cursor,omitempty"`
}

// ToolListResponse represents the response to a tools/list request.
type ToolListResponse struct {
	Tools      []Tool  `json:"tools"`
	NextCursor *string `json:"nextCursor,omitempty"`
}

// ToolCallRequest represents a request to call a tool.
type ToolCallRequest struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// ToolCallResponse represents the response from a tool call.
type ToolCallResponse struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

// Content represents the content of a tool response or resource.
type Content struct {
	Type     string                 `json:"type"`
	Text     string                 `json:"text,omitempty"`
	Data     string                 `json:"data,omitempty"`
	MimeType string                 `json:"mimeType,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Resource represents an available resource.
type Resource struct {
	URI         string                 `json:"uri"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	MimeType    string                 `json:"mimeType,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ResourceListRequest represents a request to list resources.
type ResourceListRequest struct {
	Cursor string `json:"cursor,omitempty"`
}

// ResourceListResponse represents the response to a resources/list request.
type ResourceListResponse struct {
	Resources  []Resource `json:"resources"`
	NextCursor *string    `json:"nextCursor,omitempty"`
}

// ResourceTemplate represents a parameterized resource template.
type ResourceTemplate struct {
	URITemplate string `json:"uriTemplate"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// ResourceTemplateListRequest represents a request to list resource templates.
type ResourceTemplateListRequest struct {
	Cursor string `json:"cursor,omitempty"`
}

// ResourceTemplateListResponse represents the response to a resources/templates/list request.
type ResourceTemplateListResponse struct {
	ResourceTemplates []ResourceTemplate `json:"resourceTemplates"`
	NextCursor        *string            `json:"nextCursor,omitempty"`
}

// ResourceReadRequest represents a request to read a resource.
type ResourceReadRequest struct {
	URI string `json:"uri"`
}

// ResourceReadResponse represents the response to a resources/read request.
type ResourceReadResponse struct {
	Contents []ResourceContents `json:"contents"`
}

// ResourceContents represents the content of a read resource.
type ResourceContents struct {
	URI      string                 `json:"uri"`
	MimeType string                 `json:"mimeType,omitempty"`
	Text     string                 `json:"text,omitempty"`
	Blob     string                 `json:"blob,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ResourceSubscribeRequest represents a request to subscribe to a resource.
type ResourceSubscribeRequest struct {
	URI string `json:"uri"`
}

// ResourceUnsubscribeRequest represents a request to unsubscribe from a resource.
type ResourceUnsubscribeRequest struct {
	URI string `json:"uri"`
}

// Prompt represents an available prompt template.
type Prompt struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Arguments   []PromptArgument `json:"arguments,omitempty"`
}

// PromptArgument represents an argument for a prompt template.
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// PromptListRequest represents a request to list prompts.
type PromptListRequest struct {
	Cursor string `json:"cursor,omitempty"`
}

// PromptListResponse represents the response to a prompts/list request.
type PromptListResponse struct {
	Prompts    []Prompt `json:"prompts"`
	NextCursor *string  `json:"nextCursor,omitempty"`
}

// PromptGetRequest represents a request to get a prompt.
type PromptGetRequest struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// PromptGetResponse represents the response to a prompts/get request.
type PromptGetResponse struct {
	Description string          `json:"description,omitempty"`
	Messages    []PromptMessage `json:"messages"`
}

// PromptMessage represents a message in a prompt.
type PromptMessage struct {
	Role    string               `json:"role"`
	Content PromptMessageContent `json:"content"`
}

// PromptMessageContent represents the content of a prompt message.
type PromptMessageContent struct {
	Type     string          `json:"type"`
	Text     string          `json:"text,omitempty"`
	Data     string          `json:"data,omitempty"`
	MimeType string          `json:"mimeType,omitempty"`
	Resource *PromptResource `json:"resource,omitempty"`
}

// PromptResource represents an embedded resource in a prompt message.
type PromptResource struct {
	URI      string                 `json:"uri"`
	MimeType string                 `json:"mimeType"`
	Text     string                 `json:"text,omitempty"`
	Blob     string                 `json:"blob,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// SetLevelRequest represents a request to set the logging level.
type SetLevelRequest struct {
	Level string `json:"level"`
}

// LoggingMessageNotification represents a logging message notification.
type LoggingMessageNotification struct {
	Level    string                 `json:"level"`
	Data     string                 `json:"data"`
	Logger   string                 `json:"logger,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// CancelRequest represents a request to cancel a pending request.
type CancelRequest struct {
	RequestID     interface{}     `json:"requestId"`
	RequestIDJSON json.RawMessage `json:"requestId,omitempty"`
}

// CancelResponse represents the response to a cancel request.
type CancelResponse struct{}

// ProgressNotification represents a progress notification.
type ProgressNotification struct {
	ProgressToken interface{} `json:"progressToken"`
	Progress      float64     `json:"progress"`
	Delta         float64     `json:"delta,omitempty"`
}

// Progress represents the progress token.
type Progress struct {
	ProgressToken interface{} `json:"progressToken"`
}

// Log levels supported by the protocol.
const (
	LevelDebug     = "debug"
	LevelInfo      = "info"
	LevelNotice    = "notice"
	LevelWarning   = "warning"
	LevelError     = "error"
	LevelCritical  = "critical"
	LevelAlert     = "alert"
	LevelEmergency = "emergency"
)

// MethodHandler is a function that handles a specific JSON-RPC method.
type MethodHandler func(params json.RawMessage) (interface{}, error)

// MethodRegistry manages the registration of method handlers.
type MethodRegistry struct {
	handlers map[string]MethodHandler
}

// NewMethodRegistry creates a new MethodRegistry.
//
// Returns:
//   - *MethodRegistry: The created method registry.
func NewMethodRegistry() *MethodRegistry {
	return &MethodRegistry{
		handlers: make(map[string]MethodHandler),
	}
}

// RegisterMethod registers a handler for a specific method.
//
// Parameters:
//   - method: The method name.
//   - handler: The handler function.
func (r *MethodRegistry) RegisterMethod(method string, handler MethodHandler) {
	r.handlers[method] = handler
}

// GetHandler retrieves the handler for a specific method.
//
// Parameters:
//   - method: The method name.
//
// Returns:
//   - MethodHandler: The handler function.
//   - bool: True if the handler exists, false otherwise.
func (r *MethodRegistry) GetHandler(method string) (MethodHandler, bool) {
	handler, exists := r.handlers[method]
	return handler, exists
}

// ListMethods lists all registered methods.
//
// Returns:
//   - []string: A list of registered method names.
func (r *MethodRegistry) ListMethods() []string {
	methods := make([]string, 0, len(r.handlers))
	for method := range r.handlers {
		methods = append(methods, method)
	}
	return methods
}

// Protocol method names.
const (
	MethodInitialize             = "initialize"
	MethodInitialized            = "notifications/initialized"
	MethodShutdown               = "shutdown"
	MethodToolsList              = "tools/list"
	MethodToolsCall              = "tools/call"
	MethodToolsListChanged       = "notifications/tools/list_changed"
	MethodResourcesList          = "resources/list"
	MethodResourcesRead          = "resources/read"
	MethodResourcesSubscribe     = "resources/subscribe"
	MethodResourcesUnsubscribe   = "resources/unsubscribe"
	MethodResourcesListChanged   = "notifications/resources/list_changed"
	MethodResourcesUpdated       = "notifications/resources/updated"
	MethodResourcesTemplatesList = "resources/templates/list"
	MethodPromptsList            = "prompts/list"
	MethodPromptsGet             = "prompts/get"
	MethodPromptsListChanged     = "notifications/prompts/list_changed"
	MethodLoggingSetLevel        = "logging/set_level"
	MethodLoggingMessage         = "notifications/message"
	MethodCancel                 = "notifications/cancelled"
	MethodProgress               = "notifications/progress"
)
