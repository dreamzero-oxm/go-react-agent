package agent

type Thought struct {
	Content string `json:"content"`
}

type Action struct {
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

type Observation struct {
	Content string `json:"content"`
}

type Step struct {
	Thought     *Thought     `json:"thought,omitempty"`
	Action      *Action      `json:"action,omitempty"`
	Observation *Observation `json:"observation,omitempty"`
	Error       string       `json:"error,omitempty"`
}

type ReActResponse struct {
	Thoughts []Thought `json:"thoughts"`
	Action   *Action   `json:"action,omitempty"`
	Answer   string    `json:"answer,omitempty"`
	Done     bool      `json:"done"`
}

type Tool struct {
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Parameters  map[string]Parameter `json:"parameters"`
	Execute     func(input map[string]interface{}) (string, error)
}

type Parameter struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}
