package mcp

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dreamzero-oxm/go-react-agent/mcp/protocol"
	"github.com/dreamzero-oxm/go-react-agent/mcp/transport"
)

// Client represents an MCP client that connects to an MCP server.
type Client struct {
	name         string
	transport    transport.Transport
	lifecycle    *protocol.LifecycleManager
	methodReg    *protocol.MethodRegistry
	requestID    int64
	requestMutex sync.Mutex
	responseChan chan *protocol.JSONRPCMessage
	ctx          context.Context
	cancel       context.CancelFunc
	mu           sync.RWMutex
	capabilities *protocol.ServerCapabilities
	serverInfo   *protocol.Implementation
	debug        bool
}

// ClientConfig holds the configuration for creating a new Client.
type ClientConfig struct {
	Name      string
	Transport transport.Transport
	Debug     bool
}

// NewClient creates a new MCP client.
//
// Parameters:
//   - config: The client configuration.
//
// Returns:
//   - *Client: The created client.
func NewClient(config *ClientConfig) *Client {
	ctx, cancel := context.WithCancel(context.Background())

	return &Client{
		name:         config.Name,
		transport:    config.Transport,
		lifecycle:    protocol.NewLifecycleManager(),
		methodReg:    protocol.NewMethodRegistry(),
		requestID:    0,
		responseChan: make(chan *protocol.JSONRPCMessage, 100),
		ctx:          ctx,
		cancel:       cancel,
		debug:        config.Debug,
	}
}

func (c *Client) debugLog(format string, args ...interface{}) {
	if c.debug {
		fmt.Printf("[DEBUG Client %s] "+format+"\n", append([]interface{}{c.name}, args...)...)
	}
}

// Start starts the client transport and response loop.
//
// Returns:
//   - error: An error if starting the transport fails.
func (c *Client) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.debugLog("Starting client transport")

	c.transport.SetMessageHandler(c.handleMessage)
	c.transport.SetErrorHandler(c.handleError)

	if err := c.transport.Start(c.ctx); err != nil {
		c.debugLog("Failed to start transport: %v", err)
		return fmt.Errorf("failed to start transport: %w", err)
	}

	c.debugLog("Transport started successfully, starting response loop")

	go c.responseLoop()

	return nil
}

// Initialize performs the MCP initialization handshake.
//
// Parameters:
//   - initReq: The initialization request parameters.
//
// Returns:
//   - *protocol.InitializeResponse: The server's initialization response.
//   - error: An error if initialization fails.
func (c *Client) Initialize(initReq *protocol.InitializeRequest) (*protocol.InitializeResponse, error) {
	c.debugLog("Starting initialization handshake")

	if err := c.lifecycle.Initialize(); err != nil {
		c.debugLog("Failed to initialize lifecycle: %v", err)
		return nil, fmt.Errorf("failed to initialize lifecycle: %w", err)
	}

	c.debugLog("Sending initialize request (protocol version: %s)", initReq.ProtocolVersion)

	var result protocol.InitializeResponse
	if err := c.sendRequest(protocol.MethodInitialize, initReq, &result); err != nil {
		c.debugLog("Initialize request failed: %v", err)
		return nil, fmt.Errorf("initialize request failed: %w", err)
	}

	c.debugLog("Initialize response received: server=%s version=%s", result.ServerInfo.Name, result.ServerInfo.Version)

	c.mu.Lock()
	c.capabilities = &result.Capabilities
	c.serverInfo = &result.ServerInfo
	c.mu.Unlock()

	c.debugLog("Sending initialized notification")
	if err := c.sendNotification(protocol.MethodInitialized, &protocol.InitializedNotification{}); err != nil {
		c.debugLog("Failed to send initialized notification: %v", err)
		return nil, fmt.Errorf("failed to send initialized notification: %w", err)
	}

	c.debugLog("Initialization completed successfully")

	return &result, nil
}

// Shutdown initiates the graceful shutdown of the client.
//
// Returns:
//   - error: An error if shutdown fails.
func (c *Client) Shutdown() error {
	if err := c.lifecycle.Shutdown(); err != nil {
		return err
	}

	if err := c.sendRequest(protocol.MethodShutdown, nil, nil); err != nil {
		return err
	}

	return c.Close()
}

// Close closes the client and its transport immediately.
//
// Returns:
//   - error: An error if closing fails.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.lifecycle.Close(); err != nil {
		return err
	}

	c.cancel()
	c.transport.Close()
	close(c.responseChan)

	return nil
}

// nextRequestID generates the next unique request ID.
//
// Returns:
//   - int64: The new request ID.
func (c *Client) nextRequestID() int64 {
	c.requestMutex.Lock()
	defer c.requestMutex.Unlock()
	c.requestID++
	return c.requestID
}

// sendRequest sends a JSON-RPC request and waits for the response.
//
// Parameters:
//   - method: The method name.
//   - params: The request parameters.
//   - result: Pointer to store the result (optional).
//
// Returns:
//   - error: An error if the request fails or returns an error.
func (c *Client) sendRequest(method string, params interface{}, result interface{}) error {
	id := c.nextRequestID()

	c.debugLog("Sending request: method=%s id=%d", method, id)

	msg, err := protocol.NewRequest(id, method, params)
	if err != nil {
		c.debugLog("Failed to create request: %v", err)
		return fmt.Errorf("failed to create request: %w", err)
	}

	data, err := msg.Marshal()
	if err != nil {
		c.debugLog("Failed to marshal request: %v", err)
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	if c.debug {
		c.debugLog("Request payload: %s", string(data))
	}

	if err := c.transport.Send(data); err != nil {
		c.debugLog("Failed to send request: %v", err)
		return fmt.Errorf("failed to send request: %w", err)
	}

	c.debugLog("Request sent, waiting for response (timeout: 30s)")

	timeout := time.NewTimer(30 * time.Second)
	defer timeout.Stop()

	for {
		select {
		case resp, ok := <-c.responseChan:
			if !ok {
				c.debugLog("Response channel closed")
				return fmt.Errorf("response channel closed")
			}
			respID, ok := resp.ID.(float64)
			if ok && int64(respID) == id {
				c.debugLog("Received response for id=%d", id)
				if resp.Error != nil {
					c.debugLog("RPC error: %s (code: %d)", resp.Error.Message, resp.Error.Code)
					return fmt.Errorf("RPC error: %s (code: %d)", resp.Error.Message, resp.Error.Code)
				}
				if result != nil {
					if err := resp.UnmarshalResult(result); err != nil {
						c.debugLog("Failed to unmarshal result: %v", err)
						return fmt.Errorf("failed to unmarshal result: %w", err)
					}
				}
				c.debugLog("Request completed successfully")
				return nil
			} else if !ok {
				c.debugLog("Response ID type mismatch: expected int64, got %T (%v)", resp.ID, resp.ID)
			}
		case <-timeout.C:
			c.debugLog("Request timeout for method %s", method)
			return fmt.Errorf("request timeout for method %s after 30s", method)
		case <-c.ctx.Done():
			c.debugLog("Context cancelled")
			return c.ctx.Err()
		}
	}
}

// sendNotification sends a JSON-RPC notification.
//
// Parameters:
//   - method: The notification method name.
//   - params: The notification parameters.
//
// Returns:
//   - error: An error if sending fails.
func (c *Client) sendNotification(method string, params interface{}) error {
	c.debugLog("Sending notification: method=%s", method)

	msg, err := protocol.NewNotification(method, params)
	if err != nil {
		c.debugLog("Failed to create notification: %v", err)
		return fmt.Errorf("failed to create notification: %w", err)
	}

	data, err := msg.Marshal()
	if err != nil {
		c.debugLog("Failed to marshal notification: %v", err)
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	if c.debug {
		c.debugLog("Notification payload: %s", string(data))
	}

	if err := c.transport.Send(data); err != nil {
		c.debugLog("Failed to send notification: %v", err)
		return fmt.Errorf("failed to send notification: %w", err)
	}

	c.debugLog("Notification sent successfully")

	return nil
}

// handleMessage processes incoming messages from the transport.
//
// Parameters:
//   - data: The raw message data.
func (c *Client) handleMessage(data []byte) {
	c.debugLog("handleMessage received: %s", string(data))

	msg, err := protocol.ParseRequest(data)
	if err != nil {
		c.debugLog("Failed to parse message: %v", err)
		c.handleError(fmt.Errorf("failed to parse message: %w", err))
		return
	}

	c.debugLog("Message parsed: isResponse=%v isNotification=%v isRequest=%v id=%v", msg.IsResponse(), msg.IsNotification(), msg.IsRequest(), msg.ID)

	if msg.IsResponse() {
		c.debugLog("Sending response to channel, id=%d", msg.ID)
		select {
		case c.responseChan <- msg:
			c.debugLog("Response sent to channel, id=%d", msg.ID)
		default:
			c.debugLog("Response channel full, dropping message, id=%d", msg.ID)
			c.handleError(fmt.Errorf("response channel full, dropping message"))
		}
	} else if msg.IsNotification() {
		c.handleNotification(msg)
	} else if msg.IsRequest() {
		c.handleIncomingRequest(msg)
	}
}

// handleNotification processes incoming notifications.
//
// Parameters:
//   - msg: The notification message.
func (c *Client) handleNotification(msg *protocol.JSONRPCMessage) {
	switch msg.Method {
	case protocol.MethodToolsListChanged:
	case protocol.MethodResourcesListChanged:
	case protocol.MethodResourcesUpdated:
	case protocol.MethodPromptsListChanged:
	case protocol.MethodLoggingMessage:
		var notif protocol.LoggingMessageNotification
		if err := msg.UnmarshalParams(&notif); err == nil {
			c.handleError(fmt.Errorf("[%s] %s: %s", notif.Level, notif.Logger, notif.Data))
		}
	case protocol.MethodProgress:
	case protocol.MethodCancel:
	}
}

// handleIncomingRequest processes incoming requests from the server.
//
// Parameters:
//   - msg: The request message.
func (c *Client) handleIncomingRequest(msg *protocol.JSONRPCMessage) {
	if handler, exists := c.methodReg.GetHandler(msg.Method); exists {
		result, err := handler(msg.Params)
		var resp *protocol.JSONRPCMessage
		if err != nil {
			resp, _ = protocol.NewResponse(msg.ID, nil, protocol.NewError(protocol.InternalError, err.Error(), nil))
		} else {
			resp, _ = protocol.NewResponse(msg.ID, result, nil)
		}
		data, _ := resp.Marshal()
		c.transport.Send(data)
	}
}

// responseLoop handles background tasks for the client.
func (c *Client) responseLoop() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}
	}
}

// handleError logs or handles internal client errors.
//
// Parameters:
//   - err: The error to handle.
func (c *Client) handleError(err error) {
}

// Name returns the client name.
//
// Returns:
//   - string: The client name.
func (c *Client) Name() string {
	return c.name
}

// GetCapabilities returns the negotiated server capabilities.
//
// Returns:
//   - *protocol.ServerCapabilities: The server capabilities.
func (c *Client) GetCapabilities() *protocol.ServerCapabilities {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.capabilities
}

// GetServerInfo returns the server implementation details.
//
// Returns:
//   - *protocol.Implementation: The server info.
func (c *Client) GetServerInfo() *protocol.Implementation {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.serverInfo
}

// IsReady checks if the client is in the Ready state.
//
// Returns:
//   - bool: True if the client is ready.
func (c *Client) IsReady() bool {
	return c.lifecycle.IsReady()
}

// ListTools retrieves the list of available tools from the server.
//
// Parameters:
//   - cursor: Optional cursor for pagination.
//
// Returns:
//   - *protocol.ToolListResponse: The list of tools.
//   - error: An error if the request fails.
func (c *Client) ListTools(cursor string) (*protocol.ToolListResponse, error) {
	req := &protocol.ToolListRequest{Cursor: cursor}
	var result protocol.ToolListResponse
	if err := c.sendRequest(protocol.MethodToolsList, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CallTool invokes a tool on the server.
//
// Parameters:
//   - req: The tool call request parameters.
//
// Returns:
//   - *protocol.ToolCallResponse: The tool execution result.
//   - error: An error if the request fails.
func (c *Client) CallTool(req *protocol.ToolCallRequest) (*protocol.ToolCallResponse, error) {
	var result protocol.ToolCallResponse
	if err := c.sendRequest(protocol.MethodToolsCall, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListResources retrieves the list of available resources from the server.
//
// Parameters:
//   - cursor: Optional cursor for pagination.
//
// Returns:
//   - *protocol.ResourceListResponse: The list of resources.
//   - error: An error if the request fails.
func (c *Client) ListResources(cursor string) (*protocol.ResourceListResponse, error) {
	req := &protocol.ResourceListRequest{Cursor: cursor}
	var result protocol.ResourceListResponse
	if err := c.sendRequest(protocol.MethodResourcesList, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ReadResource reads the content of a resource.
//
// Parameters:
//   - req: The resource read request parameters.
//
// Returns:
//   - *protocol.ResourceReadResponse: The resource content.
//   - error: An error if the request fails.
func (c *Client) ReadResource(req *protocol.ResourceReadRequest) (*protocol.ResourceReadResponse, error) {
	var result protocol.ResourceReadResponse
	if err := c.sendRequest(protocol.MethodResourcesRead, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SubscribeResource subscribes to changes in a resource.
//
// Parameters:
//   - req: The subscription request parameters.
//
// Returns:
//   - error: An error if the request fails.
func (c *Client) SubscribeResource(req *protocol.ResourceSubscribeRequest) error {
	return c.sendRequest(protocol.MethodResourcesSubscribe, req, nil)
}

// UnsubscribeResource unsubscribes from a resource.
//
// Parameters:
//   - req: The unsubscription request parameters.
//
// Returns:
//   - error: An error if the request fails.
func (c *Client) UnsubscribeResource(req *protocol.ResourceUnsubscribeRequest) error {
	return c.sendRequest(protocol.MethodResourcesUnsubscribe, req, nil)
}

// ListResourceTemplates retrieves the list of resource templates from the server.
//
// Parameters:
//   - cursor: Optional cursor for pagination.
//
// Returns:
//   - *protocol.ResourceTemplateListResponse: The list of resource templates.
//   - error: An error if the request fails.
func (c *Client) ListResourceTemplates(cursor string) (*protocol.ResourceTemplateListResponse, error) {
	req := &protocol.ResourceTemplateListRequest{Cursor: cursor}
	var result protocol.ResourceTemplateListResponse
	if err := c.sendRequest(protocol.MethodResourcesTemplatesList, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListPrompts retrieves the list of available prompts from the server.
//
// Parameters:
//   - cursor: Optional cursor for pagination.
//
// Returns:
//   - *protocol.PromptListResponse: The list of prompts.
//   - error: An error if the request fails.
func (c *Client) ListPrompts(cursor string) (*protocol.PromptListResponse, error) {
	req := &protocol.PromptListRequest{Cursor: cursor}
	var result protocol.PromptListResponse
	if err := c.sendRequest(protocol.MethodPromptsList, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetPrompt retrieves a specific prompt.
//
// Parameters:
//   - req: The prompt get request parameters.
//
// Returns:
//   - *protocol.PromptGetResponse: The prompt details.
//   - error: An error if the request fails.
func (c *Client) GetPrompt(req *protocol.PromptGetRequest) (*protocol.PromptGetResponse, error) {
	var result protocol.PromptGetResponse
	if err := c.sendRequest(protocol.MethodPromptsGet, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SetLogLevel sets the logging level on the server.
//
// Parameters:
//   - level: The desired log level.
//
// Returns:
//   - error: An error if the request fails.
func (c *Client) SetLogLevel(level string) error {
	req := &protocol.SetLevelRequest{Level: level}
	return c.sendRequest(protocol.MethodLoggingSetLevel, req, nil)
}
