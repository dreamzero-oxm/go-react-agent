package agent

import "time"

type Config struct {
	MaxIterations int           `json:"max_iterations"`
	Timeout       time.Duration `json:"timeout"`
	Temperature   float64       `json:"temperature"`
	MaxTokens     int           `json:"max_tokens"`
	Parser        ResponseParser `json:"-"` // Response parser for LLM output
}

func DefaultConfig() *Config {
	return &Config{
		MaxIterations: 10,
		Timeout:       5 * time.Minute,
		Temperature:   0.7,
		MaxTokens:     2000,
		Parser:        NewJSONParser(),
	}
}
