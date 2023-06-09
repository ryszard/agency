package client

import (
	"context"
	"fmt"
	"io"
)

// RetryableError is an error from the API that can be retried.
type RetryableError struct {
	originalError error
}

func (r RetryableError) Error() string {
	return fmt.Sprintf("retryable: %v", r.originalError)
}

func (r RetryableError) Unwrap() error {
	return r.originalError
}

func Retryable(err error) error {
	return &RetryableError{
		originalError: err,
	}
}

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

	// If Stream is not nil, the client will use the streaming API. The client
	// should write the message content from the server as it appears on the
	// wire to Stream, and then still return the whole message.
	Stream io.Writer `json:"-"` // This should not be used when hashing.
}

func (r ChatCompletionRequest) WantsStreaming() bool {
	return r.Stream != nil
}

// Client is an interface for the LLM API client. Any methods that return errors
// should return a RetryableError (by calling Retryable) if the error is
// retryable, or any other error if it is not.
type Client interface {
	CreateChatCompletion(ctx context.Context, req ChatCompletionRequest) (ChatCompletionResponse, error)

	// TODO(ryszard): Implement this.
	//SupportedParameters() []string
}
