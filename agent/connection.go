package agent

import (
	"context"

	"github.com/sashabaranov/go-openai"
)

// Client is an interface for the OpenAI API client. It's main purpose is to
// make testing easier.
type Client interface {
	CreateChatCompletion(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error)
	CreateChatCompletionStream(context.Context, openai.ChatCompletionRequest) (*openai.ChatCompletionStream, error)
}
