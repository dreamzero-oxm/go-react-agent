// Package agent provides the core ReAct agent implementation for building
// intelligent AI agents with tool-using capabilities.
//
// The ReAct (Reasoning + Acting) pattern enables agents to iteratively reason
// about a task, take actions using available tools, observe results, and
// eventually provide a final answer.
package agent

// Thought represents a single reasoning step in the agent's thought process.
type Thought struct {
	// Content is the text content of the thought
	Content string `json:"content"`
}

// Action represents an action the agent wants to perform using a tool.
type Action struct {
	// Name is the name of the tool to use
	Name string `json:"name"`
	// Input contains the parameters to pass to the tool
	Input map[string]interface{} `json:"input"`
}

// Observation represents the result returned from executing an action.
type Observation struct {
	// Content is the text content of the observation
	Content string `json:"content"`
}

// Step represents a single step in the agent's execution, containing
// a thought, action, and observation.
type Step struct {
	// Thought is the reasoning step (optional)
	Thought *Thought `json:"thought,omitempty"`
	// Action is the action to perform (optional)
	Action *Action `json:"action,omitempty"`
	// Observation is the result from the action (optional)
	Observation *Observation `json:"observation,omitempty"`
	// Error contains any error message from the action execution
	Error string `json:"error,omitempty"`
}

// ReActResponse represents the response from the ReAct agent containing
// thoughts, actions, and the final answer.
type ReActResponse struct {
	// Thoughts is the sequence of reasoning steps
	Thoughts []Thought `json:"thoughts"`
	// Action is the next action to perform (if not done)
	Action *Action `json:"action,omitempty"`
	// Answer is the final answer when the agent is done
	Answer string `json:"answer,omitempty"`
	// Done indicates whether the agent has reached a final answer
	Done bool `json:"done"`
}

// StructuredResponse wraps the structured output result from the agent.
//
// It combines the raw ReActResponse with a parsed structured output of type T.
type StructuredResponse[T any] struct {
	// ReActResponse is the raw response from the agent
	ReActResponse *ReActResponse
	// Output is the parsed structured output
	Output *T
}
