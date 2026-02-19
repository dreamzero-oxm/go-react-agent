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

// StructuredResponse 包装结构化输出结果
type StructuredResponse[T any] struct {
	ReActResponse *ReActResponse
	Output        *T // 解析后的结构体输出
}
