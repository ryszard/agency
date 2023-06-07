package anthropic

import (
	"context"
	"fmt"

	"github.com/madebywelch/anthropic-go/pkg/anthropic"
	"github.com/ryszard/agency/client"
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

	resp, err := cl.client.Complete(request, nil)
	if err != nil {
		return client.ChatCompletionResponse{}, err
	}
	return TranslateResponse(resp), nil
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
