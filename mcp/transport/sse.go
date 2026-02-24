package transport

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// SSEConfig holds configuration for SSETransport.
type SSEConfig struct {
	URL       string
	Headers   map[string]string
	Timeout   int
	KeepAlive bool
	Debug     bool
}

// SSETransport implements Transport using Server-Sent Events (SSE).
// Note: Send is not supported as SSE is a one-way channel from server to client.
type SSETransport struct {
	*BaseTransport
	config     *SSEConfig
	client     *http.Client
	cancelFunc context.CancelFunc
	mu         sync.Mutex
	closed     bool
	eventChan  chan SSEEvent
}

// SSEEvent represents a single Server-Sent Event.
type SSEEvent struct {
	ID    string
	Event string
	Data  string
}

// NewSSETransport creates a new SSETransport.
//
// Parameters:
//   - config: The configuration for the SSE connection.
//
// Returns:
//   - *SSETransport: The created transport.
func NewSSETransport(config *SSEConfig) *SSETransport {
	timeout := time.Duration(config.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &SSETransport{
		BaseTransport: NewBaseTransport(),
		config:        config,
		client: &http.Client{
			Timeout: timeout,
		},
		closed:    false,
		eventChan: make(chan SSEEvent, 100),
	}
}

// Type returns the transport type.
//
// Returns:
//   - TransportType: TransportSSE.
func (st *SSETransport) Type() TransportType {
	return TransportSSE
}

// Start initiates the SSE connection.
//
// Parameters:
//   - ctx: The context for cancellation.
//
// Returns:
//   - error: An error if connection setup fails.
func (st *SSETransport) Start(ctx context.Context) error {
	st.mu.Lock()
	defer st.mu.Unlock()

	if st.cancelFunc != nil {
		return fmt.Errorf("transport already started")
	}

	if st.config.Debug {
		fmt.Printf("[DEBUG SSETransport] Starting SSE connection to: %s\n", st.config.URL)
	}

	ctx, st.cancelFunc = context.WithCancel(ctx)

	req, err := http.NewRequestWithContext(ctx, "GET", st.config.URL, nil)
	if err != nil {
		if st.config.Debug {
			fmt.Printf("[DEBUG SSETransport] Failed to create request: %v\n", err)
		}
		return fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range st.config.Headers {
		req.Header.Set(key, value)
		if st.config.Debug {
			fmt.Printf("[DEBUG SSETransport] Setting header: %s = %s\n", key, value)
		}
	}

	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	go st.connectAndRead(req)

	return nil
}

// connectAndRead handles the connection logic and reading loop.
// It implements reconnection logic if KeepAlive is enabled.
func (st *SSETransport) connectAndRead(req *http.Request) {
	for {
		if st.config.Debug {
			fmt.Printf("[DEBUG SSETransport] Connecting to SSE endpoint...\n")
		}
		resp, err := st.client.Do(req)
		if err != nil {
			if st.config.Debug {
				fmt.Printf("[DEBUG SSETransport] HTTP request failed: %v\n", err)
			}
			st.handleError(fmt.Errorf("HTTP request failed: %w", err))
			return
		}

		if st.config.Debug {
			fmt.Printf("[DEBUG SSETransport] Received response status: %d\n", resp.StatusCode)
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			if st.config.Debug {
				fmt.Printf("[DEBUG SSETransport] Unexpected status code: %d\n", resp.StatusCode)
			}
			st.handleError(fmt.Errorf("unexpected status code: %d", resp.StatusCode))
			return
		}

		st.readSSE(resp.Body)
		resp.Body.Close()

		if st.IsClosed() || !st.config.KeepAlive {
			break
		}

		select {
		case <-req.Context().Done():
			return
		case <-time.After(5 * time.Second):
			req, err = http.NewRequestWithContext(context.Background(), "GET", st.config.URL, nil)
			if err != nil {
				st.handleError(fmt.Errorf("failed to recreate request: %w", err))
				return
			}
			for key, value := range st.config.Headers {
				req.Header.Set(key, value)
			}
			req.Header.Set("Accept", "text/event-stream")
		}
	}
}

// readSSE reads SSE events from the response body.
func (st *SSETransport) readSSE(body io.Reader) {
	scanner := bufio.NewScanner(body)
	var currentEvent SSEEvent

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			if currentEvent.Data != "" {
				st.handleMessage([]byte(currentEvent.Data))
				st.eventChan <- currentEvent
				currentEvent = SSEEvent{}
			}
			continue
		}

		if strings.HasPrefix(line, "id:") {
			currentEvent.ID = strings.TrimSpace(strings.TrimPrefix(line, "id:"))
		} else if strings.HasPrefix(line, "event:") {
			currentEvent.Event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "data:") {
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if currentEvent.Data != "" {
				currentEvent.Data += "\n"
			}
			currentEvent.Data += data
		}
	}

	if err := scanner.Err(); err != nil {
		st.handleError(fmt.Errorf("SSE read error: %w", err))
	}
}

// Send implements Transport.Send but returns error as SSE is read-only.
//
// Parameters:
//   - data: The data to send.
//
// Returns:
//   - error: Always returns an error indicating SSE is read-only.
func (st *SSETransport) Send(data []byte) error {
	return fmt.Errorf("SSE transport is read-only, cannot send data")
}

// Receive waits for and returns the next SSE event data.
//
// Returns:
//   - []byte: The event data.
//   - error: An error on timeout or if closed.
func (st *SSETransport) Receive() ([]byte, error) {
	select {
	case event := <-st.eventChan:
		return []byte(event.Data), nil
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("receive timeout")
	}
}

// Close closes the SSE connection.
//
// Returns:
//   - error: Always returns nil.
func (st *SSETransport) Close() error {
	st.mu.Lock()
	defer st.mu.Unlock()

	if st.closed {
		return nil
	}

	st.closed = true

	if st.cancelFunc != nil {
		st.cancelFunc()
	}

	close(st.eventChan)

	return nil
}

// IsClosed checks if the transport is closed.
//
// Returns:
//   - bool: True if closed.
func (st *SSETransport) IsClosed() bool {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.closed
}

// Events returns the channel of raw SSE events.
//
// Returns:
//   - <-chan SSEEvent: The event channel.
func (st *SSETransport) Events() <-chan SSEEvent {
	return st.eventChan
}
