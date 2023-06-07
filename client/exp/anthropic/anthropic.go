package anthropic

import (
	"context"
	"fmt"
	"io"

	"github.com/madebywelch/anthropic-go/pkg/anthropic"
	"github.com/ryszard/agency/client"
	log "github.com/sirupsen/logrus"
)

const (
	// Our largest model, ideal for a wide range of more complex tasks.
	ClaudeV1 = "claude-v1"

	// An enhanced version of ClaudeV1 with a 100,000 token context window.
	ClaudeV1_100k = "claude-v1-100k"

	// A smaller model with far lower latency, sampling at roughly 40 words/sec!
	ClaudeInstantV1 = "claude-instant-v1"

	// An enhanced version of ClaudeInstantV1 with a 100,000 token context window.
	ClaudeInstantV1_100k = "claude-instant-v1-100k"

	// More robust against red-team inputs, better at precise instruction-following,
	// better at code, and better and non-English dialogue and writing.
	ClaudeV1_3 = "claude-v1.3"

	// An enhanced version of ClaudeV1_3 with a 100,000 token context window.
	ClaudeV1_3_100k = "claude-v1.3-100k"

	// An improved version of ClaudeV1, slightly improved at general helpfulness,
	// instruction following, coding, and other tasks. It is also considerably
	// better with non-English languages.
	ClaudeV1_2 = "claude-v1.2"

	// An earlier version of ClaudeV1.
	ClaudeV1_0 = "claude-v1.0"

	// Latest version of ClaudeInstantV1. It is better at a wide variety of tasks
	// including writing, coding, and instruction following. It performs better on
	// academic benchmarks, including math, reading comprehension, and coding tests.
	ClaudeInstantV1_1 = "claude-instant-v1.1"

	// An enhanced version of ClaudeInstantV1_1 with a 100,000 token context window.
	ClaudeInstantV1_1_100k = "claude-instant-v1.1-100k"

	// An earlier version of ClaudeInstantV1.
	ClaudeInstantV1_0 = "claude-instant-v1.0"
)

type Client struct {
	client *anthropic.Client
}

func New(apiKey string) (*Client, error) {
	cl, err := anthropic.NewClient(apiKey)
	if err != nil {
		return nil, err
	}
	return &Client{
		client: cl,
	}, nil
}

func (cl *Client) CreateChatCompletion(ctx context.Context, req client.ChatCompletionRequest) (client.ChatCompletionResponse, error) {
	request, err := TranslateRequest(&req)
	if err != nil {
		return client.ChatCompletionResponse{}, err
	}
	if req.WantsStreaming() {
		return cl.createChatCompletionStream(ctx, req, req.Stream)
	}
	resp, err := cl.client.Complete(request, nil)
	if err != nil {
		return client.ChatCompletionResponse{}, err
	}
	return TranslateResponse(resp), nil
}

func (cl *Client) createChatCompletionStream(ctx context.Context, req client.ChatCompletionRequest, w io.Writer) (client.ChatCompletionResponse, error) {

	request, err := TranslateRequest(&req)
	if err != nil {
		return client.ChatCompletionResponse{}, err
	}

	request.Stream = true
	var response *anthropic.CompletionResponse

	callback := func(resp *anthropic.CompletionResponse) error {
		log.WithField("resp", fmt.Sprintf("%#v", resp)).Debug("Received response from server")
		response = resp
		if _, err := w.Write([]byte(resp.Delta)); err != nil {
			return err
		}
		return nil
	}

	_, err = cl.client.Complete(request, callback)
	if err != nil {
		return client.ChatCompletionResponse{}, err
	}
	return TranslateResponse(response), nil
}

func TranslateResponse(resp *anthropic.CompletionResponse) client.ChatCompletionResponse {
	return client.ChatCompletionResponse{
		Choices: []client.Message{{Role: client.Assistant, Content: resp.Completion}},
	}
}

func TranslateRequest(clientReq *client.ChatCompletionRequest) (*anthropic.CompletionRequest, error) {
	req := anthropic.CompletionRequest{
		Model:             anthropic.Model(clientReq.Model),
		MaxTokensToSample: clientReq.MaxTokens,
		Temperature:       float64(clientReq.Temperature),
		Prompt:            TranslateMessages(clientReq.Messages),
	}

	if stopSequences, ok := clientReq.CustomParams["stop_sequences"]; ok {
		req.StopSequences, ok = stopSequences.([]string)
		if !ok {
			return nil, fmt.Errorf("stop_sequences must be an array of strings")
		}
	}

	if topK, ok := clientReq.CustomParams["top_k"]; ok {
		req.TopK, ok = topK.(int)
		if !ok {
			return nil, fmt.Errorf("top_k must be an int")
		}
	}

	if topP, ok := clientReq.CustomParams["top_p"]; ok {
		req.TopP, ok = topP.(float64)
		if !ok {
			return nil, fmt.Errorf("top_p must be a float")
		}
	}

	// Add checks for other custom parameters as needed...

	return &req, nil
}

func TranslateMessages(messages []client.Message) string {
	var s string
	for _, m := range messages {
		if m.Role == client.User || m.Role == client.System {
			s += "Human: "
		} else {
			s += "Assistant: "
		}
		s += m.Content + "\n\n"
	}
	if len(messages) > 0 {
		s += "Assistant:"
	}

	return s
}
