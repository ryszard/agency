package client

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"

	log "github.com/sirupsen/logrus"
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

	// Stream is a writer to which the API should write the response as it
	// appears. The API will still return the response as a whole.
	Stream io.Writer `json:"-"` // This should not be used when hashing.
}

func (r ChatCompletionRequest) hash() ([]byte, error) {
	data, err := json.Marshal(r)
	if err != nil {
		log.WithError(err).Error("hash: failed to marshal messages")
		return nil, err
	}

	hash := sha256.Sum256(data)
	return []byte(hex.EncodeToString(hash[:])), nil

}

type ChatCompletionStream chan []string

// Client is an interface for the OpenAI API client. It's main purpose is to
// make testing easier.
type Client interface {
	CreateChatCompletion(ctx context.Context, req ChatCompletionRequest) (ChatCompletionResponse, error)

	// TODO(ryszard): Implement this.
	//SupportedParameters() []string
}
