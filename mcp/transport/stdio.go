package transport

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
)

// StdioConfig holds configuration for StdioTransport.
type StdioConfig struct {
	Command string
	Args    []string
	Env     []string
	Timeout int
	Debug   bool
}

// StdioTransport implements Transport using standard input/output of a subprocess.
type StdioTransport struct {
	*BaseTransport
	config     *StdioConfig
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	stdout     io.Reader
	stderr     io.Reader
	cancelFunc context.CancelFunc
	mu         sync.Mutex
	closed     bool
}

// NewStdioTransport creates a new StdioTransport.
//
// Parameters:
//   - config: The configuration for the subprocess.
//
// Returns:
//   - *StdioTransport: The created transport.
func NewStdioTransport(config *StdioConfig) *StdioTransport {
	return &StdioTransport{
		BaseTransport: NewBaseTransport(),
		config:        config,
		closed:        false,
	}
}

// Type returns the transport type.
//
// Returns:
//   - TransportType: TransportStdio.
func (st *StdioTransport) Type() TransportType {
	return TransportStdio
}

// Start starts the subprocess and the read loops.
//
// Parameters:
//   - ctx: The context for cancellation.
//
// Returns:
//   - error: An error if starting the subprocess fails.
func (st *StdioTransport) Start(ctx context.Context) error {
	st.mu.Lock()
	defer st.mu.Unlock()

	if st.cmd != nil {
		return fmt.Errorf("transport already started")
	}

	ctx, st.cancelFunc = context.WithCancel(ctx)

	if st.config.Debug {
		fmt.Printf("[DEBUG StdioTransport] Starting command: %s %v\n", st.config.Command, st.config.Args)
	}

	st.cmd = exec.CommandContext(ctx, st.config.Command, st.config.Args...)

	parentEnv := os.Environ()
	for _, env := range st.config.Env {
		parentEnv = append(parentEnv, env)
	}
	st.cmd.Env = parentEnv

	if st.config.Debug {
		fmt.Printf("[DEBUG StdioTransport] Environment variables count: %d\n", len(parentEnv))
		for _, e := range st.config.Env {
			fmt.Printf("[DEBUG StdioTransport] Config env: %s\n", e)
		}
	}

	stdin, err := st.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	st.stdin = stdin

	stdout, err := st.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	st.stdout = stdout

	stderr, err := st.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	st.stderr = stderr

	if err := st.cmd.Start(); err != nil {
		if st.config.Debug {
			fmt.Printf("[DEBUG StdioTransport] Failed to start command: %v\n", err)
		}
		return fmt.Errorf("failed to start command: %w", err)
	}

	if st.config.Debug {
		fmt.Printf("[DEBUG StdioTransport] Command started successfully, PID: %d\n", st.cmd.Process.Pid)
	}

	go st.readLoop()
	go st.readErrorLoop()

	return nil
}

// Send writes data to the subprocess's stdin.
//
// Parameters:
//   - data: The data to send.
//
// Returns:
//   - error: An error if writing fails or the transport is closed.
func (st *StdioTransport) Send(data []byte) error {
	st.mu.Lock()
	defer st.mu.Unlock()

	if st.closed || st.stdin == nil {
		return fmt.Errorf("transport is closed")
	}

	_, err := st.stdin.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write to stdin: %w", err)
	}

	_, err = st.stdin.Write([]byte("\n"))
	if err != nil {
		return fmt.Errorf("failed to write newline to stdin: %w", err)
	}

	return nil
}

// Receive reads the next message from the subprocess's stdout.
//
// Returns:
//   - []byte: The received message.
//   - error: An error if reading fails.
func (st *StdioTransport) Receive() ([]byte, error) {
	return readMessage(st.stdout)
}

// Close terminates the subprocess and closes pipes.
//
// Returns:
//   - error: An error if closing fails.
func (st *StdioTransport) Close() error {
	st.mu.Lock()
	defer st.mu.Unlock()

	if st.closed {
		return nil
	}

	st.closed = true

	if st.cancelFunc != nil {
		st.cancelFunc()
	}

	if st.stdin != nil {
		st.stdin.Close()
	}

	if st.cmd != nil && st.cmd.Process != nil {
		st.cmd.Process.Signal(syscall.SIGTERM)
		st.cmd.Wait()
	}

	return nil
}

// readLoop reads lines from stdout and invokes the message handler.
func (st *StdioTransport) readLoop() {
	if st.config.Debug {
		fmt.Printf("[DEBUG StdioTransport] readLoop started\n")
	}
	scanner := bufio.NewScanner(st.stdout)
	for scanner.Scan() {
		line := scanner.Bytes()
		if st.config.Debug {
			fmt.Printf("[DEBUG StdioTransport] stdout received: %s\n", string(line))
		}
		st.handleMessage(line)
	}

	if err := scanner.Err(); err != nil {
		if st.config.Debug {
			fmt.Printf("[DEBUG StdioTransport] stdout read error: %v\n", err)
		}
		st.handleError(fmt.Errorf("stdout read error: %w", err))
	}
	if st.config.Debug {
		fmt.Printf("[DEBUG StdioTransport] readLoop ended\n")
	}
}

// readErrorLoop reads lines from stderr and invokes the error handler.
func (st *StdioTransport) readErrorLoop() {
	if st.config.Debug {
		fmt.Printf("[DEBUG StdioTransport] readErrorLoop started\n")
	}
	scanner := bufio.NewScanner(st.stderr)
	for scanner.Scan() {
		text := scanner.Text()
		if st.config.Debug {
			fmt.Printf("[DEBUG StdioTransport] stderr received: %s\n", text)
		}
		st.handleError(fmt.Errorf("stderr: %s", text))
	}

	if err := scanner.Err(); err != nil {
		if st.config.Debug {
			fmt.Printf("[DEBUG StdioTransport] stderr read error: %v\n", err)
		}
		st.handleError(fmt.Errorf("stderr read error: %w", err))
	}
	if st.config.Debug {
		fmt.Printf("[DEBUG StdioTransport] readErrorLoop ended\n")
	}
}

// IsRunning checks if the subprocess is currently running.
//
// Returns:
//   - bool: True if the process is running.
func (st *StdioTransport) IsRunning() bool {
	st.mu.Lock()
	defer st.mu.Unlock()

	return !st.closed && st.cmd != nil && st.cmd.Process != nil
}

// PID returns the process ID of the subprocess.
//
// Returns:
//   - int: The PID, or 0 if not running.
func (st *StdioTransport) PID() int {
	st.mu.Lock()
	defer st.mu.Unlock()

	if st.cmd != nil && st.cmd.Process != nil {
		return st.cmd.Process.Pid
	}
	return 0
}

// NewStdioTransportForTest creates a StdioTransport with provided I/O for testing.
//
// Parameters:
//   - stdin: Reader to simulate stdin (output from transport).
//   - stdout: Writer to simulate stdout (input to transport).
//   - stderr: Reader to simulate stderr.
//
// Returns:
//   - *StdioTransport: The created test transport.
func NewStdioTransportForTest(stdin io.Reader, stdout io.WriteCloser, stderr io.Reader) *StdioTransport {
	return &StdioTransport{
		BaseTransport: NewBaseTransport(),
		stdin:         stdout,
		stdout:        stdin,
		stderr:        stderr,
		closed:        false,
	}
}
