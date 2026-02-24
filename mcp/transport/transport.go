package transport

import (
	"context"
	"io"
)

// TransportType represents the type of transport.
type TransportType string

// Supported transport types.
const (
	TransportStdio TransportType = "stdio"
	TransportSSE   TransportType = "sse"
)

// Transport defines the interface for MCP transport layer.
type Transport interface {
	// Type returns the transport type.
	Type() TransportType

	// Send sends data over the transport.
	//
	// Parameters:
	//   - data: The data to send.
	//
	// Returns:
	//   - error: An error if sending fails.
	Send(data []byte) error

	// Receive waits for and returns the next message.
	//
	// Returns:
	//   - []byte: The received message.
	//   - error: An error if receiving fails.
	Receive() ([]byte, error)

	// Close closes the transport.
	//
	// Returns:
	//   - error: An error if closing fails.
	Close() error

	// Start starts the transport.
	//
	// Parameters:
	//   - ctx: The context for cancellation.
	//
	// Returns:
	//   - error: An error if starting fails.
	Start(ctx context.Context) error

	// SetMessageHandler sets the handler for incoming messages.
	//
	// Parameters:
	//   - handler: The message handler function.
	SetMessageHandler(handler func([]byte))

	// SetErrorHandler sets the handler for transport errors.
	//
	// Parameters:
	//   - handler: The error handler function.
	SetErrorHandler(handler func(error))
}

// BaseTransport provides common functionality for transports.
type BaseTransport struct {
	messageHandler func([]byte)
	errorHandler   func(error)
}

// NewBaseTransport creates a new BaseTransport.
//
// Returns:
//   - *BaseTransport: The created base transport.
func NewBaseTransport() *BaseTransport {
	return &BaseTransport{}
}

// SetMessageHandler sets the message handler.
//
// Parameters:
//   - handler: The function to handle incoming messages.
func (bt *BaseTransport) SetMessageHandler(handler func([]byte)) {
	bt.messageHandler = handler
}

// SetErrorHandler sets the error handler.
//
// Parameters:
//   - handler: The function to handle transport errors.
func (bt *BaseTransport) SetErrorHandler(handler func(error)) {
	bt.errorHandler = handler
}

// handleMessage invokes the message handler if set.
//
// Parameters:
//   - data: The message data.
func (bt *BaseTransport) handleMessage(data []byte) {
	if bt.messageHandler != nil {
		bt.messageHandler(data)
	}
}

// handleError invokes the error handler if set.
//
// Parameters:
//   - err: The error to handle.
func (bt *BaseTransport) handleError(err error) {
	if bt.errorHandler != nil {
		bt.errorHandler(err)
	}
}

// readMessage reads a full message from a reader.
// It handles potential fragmentation and message boundaries.
//
// Parameters:
//   - reader: The reader to read from.
//
// Returns:
//   - []byte: The read message.
//   - error: An error if reading fails.
func readMessage(reader io.Reader) ([]byte, error) {
	buf := make([]byte, 1)
	var message []byte

	for {
		n, err := reader.Read(buf)
		if err != nil {
			if err == io.EOF {
				if len(message) > 0 {
					return message, nil
				}
				return nil, err
			}
			return nil, err
		}

		if n > 0 {
			if buf[0] == '\n' {
				if len(message) > 0 {
					return message, nil
				}
			} else if buf[0] != '\r' {
				message = append(message, buf[0])
			}
		}
	}
}
