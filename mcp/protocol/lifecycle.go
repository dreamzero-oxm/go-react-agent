package protocol

import (
	"context"
	"fmt"
	"sync"
)

// LifecycleState represents the state of the MCP connection lifecycle.
type LifecycleState string

// Available lifecycle states.
const (
	StateUninitialized LifecycleState = "uninitialized"
	StateInitializing  LifecycleState = "initializing"
	StateReady         LifecycleState = "ready"
	StateShuttingDown  LifecycleState = "shutting_down"
	StateClosed        LifecycleState = "closed"
)

// LifecycleManager manages the state transitions of the MCP connection.
type LifecycleManager struct {
	state      LifecycleState
	stateMutex sync.RWMutex
	eventChan  chan LifecycleEvent
}

// LifecycleEvent represents a change in the lifecycle state.
type LifecycleEvent struct {
	Type     LifecycleEventType
	OldState LifecycleState
	NewState LifecycleState
	Error    error
}

// LifecycleEventType represents the type of lifecycle event.
type LifecycleEventType string

// Available lifecycle event types.
const (
	EventStateChange LifecycleEventType = "state_change"
	EventError       LifecycleEventType = "error"
)

// NewLifecycleManager creates a new LifecycleManager.
//
// Returns:
//   - *LifecycleManager: The created lifecycle manager.
func NewLifecycleManager() *LifecycleManager {
	return &LifecycleManager{
		state:     StateUninitialized,
		eventChan: make(chan LifecycleEvent, 100),
	}
}

// GetState returns the current lifecycle state.
//
// Returns:
//   - LifecycleState: The current state.
func (lm *LifecycleManager) GetState() LifecycleState {
	lm.stateMutex.RLock()
	defer lm.stateMutex.RUnlock()
	return lm.state
}

// setState transitions the manager to a new state if the transition is valid.
//
// Parameters:
//   - newState: The target state.
//
// Returns:
//   - error: An error if the transition is invalid.
func (lm *LifecycleManager) setState(newState LifecycleState) error {
	lm.stateMutex.Lock()
	defer lm.stateMutex.Unlock()

	oldState := lm.state

	if !lm.isValidTransition(oldState, newState) {
		return fmt.Errorf("invalid state transition from %s to %s", oldState, newState)
	}

	lm.state = newState

	event := LifecycleEvent{
		Type:     EventStateChange,
		OldState: oldState,
		NewState: newState,
	}

	select {
	case lm.eventChan <- event:
	default:
	}

	return nil
}

// isValidTransition checks if a state transition is allowed.
//
// Parameters:
//   - from: The current state.
//   - to: The target state.
//
// Returns:
//   - bool: True if the transition is valid, false otherwise.
func (lm *LifecycleManager) isValidTransition(from, to LifecycleState) bool {
	validTransitions := map[LifecycleState][]LifecycleState{
		StateUninitialized: {StateInitializing, StateClosed},
		StateInitializing:  {StateReady, StateClosed},
		StateReady:         {StateShuttingDown, StateClosed},
		StateShuttingDown:  {StateClosed},
		StateClosed:        {},
	}

	for _, valid := range validTransitions[from] {
		if valid == to {
			return true
		}
	}
	return false
}

// Initialize transitions the state to Ready via Initializing.
//
// Returns:
//   - error: An error if the transition fails.
func (lm *LifecycleManager) Initialize() error {
	if err := lm.setState(StateInitializing); err != nil {
		return err
	}
	return lm.setState(StateReady)
}

// Shutdown transitions the state to Closed via ShuttingDown.
//
// Returns:
//   - error: An error if the transition fails.
func (lm *LifecycleManager) Shutdown() error {
	if err := lm.setState(StateShuttingDown); err != nil {
		return err
	}
	return lm.setState(StateClosed)
}

// Close transitions the state directly to Closed.
//
// Returns:
//   - error: An error if the transition fails.
func (lm *LifecycleManager) Close() error {
	return lm.setState(StateClosed)
}

// WaitForState waits until the lifecycle reaches the desired state or the context is cancelled.
//
// Parameters:
//   - ctx: The context for cancellation.
//   - desiredState: The state to wait for.
//
// Returns:
//   - error: An error if the context is cancelled or the connection is closed.
func (lm *LifecycleManager) WaitForState(ctx context.Context, desiredState LifecycleState) error {
	for {
		currentState := lm.GetState()
		if currentState == desiredState {
			return nil
		}

		if currentState == StateClosed {
			return fmt.Errorf("connection closed while waiting for state %s", desiredState)
		}

		select {
		case event := <-lm.eventChan:
			if event.Type == EventStateChange {
				continue
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Events returns a channel for listening to lifecycle events.
//
// Returns:
//   - <-chan LifecycleEvent: The event channel.
func (lm *LifecycleManager) Events() <-chan LifecycleEvent {
	return lm.eventChan
}

// IsReady checks if the lifecycle state is Ready.
//
// Returns:
//   - bool: True if the state is Ready.
func (lm *LifecycleManager) IsReady() bool {
	return lm.GetState() == StateReady
}

// IsClosed checks if the lifecycle state is Closed.
//
// Returns:
//   - bool: True if the state is Closed.
func (lm *LifecycleManager) IsClosed() bool {
	return lm.GetState() == StateClosed
}

// CloseEvents closes the event channel.
func (lm *LifecycleManager) CloseEvents() {
	close(lm.eventChan)
}
