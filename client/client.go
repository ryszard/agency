package client

import (
	"context"
)

type Role string

const (
	System    Role = "system"
	User      Role = "user"
	Assistant Role = "assistant"
)

type ChatCompletionResponse struct {
	Choices []Message `json:"choices"`
}

type Message struct {
	Content string `json:"content"`
	Role    Role   `json:"role"`
}

// FIXME(ryszard): What about N?

type ChatCompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature float32   `json:"temperature"`
	// This is an escape hatch for passing arbitrary parameters to the APIs. It
	// is the client's responsibility to ensure that the parameters are valid
	// for the model.
	CustomParams map[string]interface{} `json:"params"`
}

type ChatCompletionStreamRequest interface{}

type ChatCompletionStream chan []string

// Client is an interface for the OpenAI API client. It's main purpose is to
// make testing easier.
type Client interface {
	CreateChatCompletion(ctx context.Context, req ChatCompletionRequest) (ChatCompletionResponse, error)
	CreateChatCompletionStream(ctx context.Context, req ChatCompletionStreamRequest) (ChatCompletionStream, error)
	SupportsStreaming() bool

	// TODO(ryszard): Implement this.
	//SupportedParameters() []string
}
