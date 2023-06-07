package openai

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/ryszard/agency/client"
	"github.com/sashabaranov/go-openai"
	log "github.com/sirupsen/logrus"
)

type Model string

const (
	GPT432K0314       Model = "gpt-4-32k-0314"
	GPT432K           Model = "gpt-4-32k"
	GPT40314          Model = "gpt-4-0314"
	GPT4                    = "gpt-4"
	GPT3Dot5Turbo0301       = "gpt-3.5-turbo-0301"
	GPT3Dot5Turbo           = "gpt-3.5-turbo"
)

type Client struct {
	client *openai.Client
}

func New(apiKey string) *Client {
	return &Client{
		client: openai.NewClient(apiKey),
	}
}

func NewClient(cl *openai.Client) *Client {
	return &Client{
		client: cl,
	}
}

func (Client) SupportsStreaming() bool {
	return true
}

var _ client.Client = (*Client)(nil)

func (cl *Client) CreateChatCompletion(ctx context.Context, request client.ChatCompletionRequest) (client.ChatCompletionResponse, error) {
	req, err := TranslateRequest(request)
	if err != nil {
		return client.ChatCompletionResponse{}, err
	}
	if request.WantsStreaming() {
		return cl.createChatCompletionStream(ctx, req, request.Stream)
	}
	resp, err := cl.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return client.ChatCompletionResponse{}, err
	}
	return TranslateResponse(resp), nil
}

func (cl *Client) createChatCompletionStream(ctx context.Context, req openai.ChatCompletionRequest, w io.Writer) (client.ChatCompletionResponse, error) {
	req.Stream = true

	log.WithFields(log.Fields{
		"request": fmt.Sprintf("%+v", req),
		"stream":  true,
	}).Info("RespondStream: Sending request")
	stream, err := cl.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return client.ChatCompletionResponse{}, err
	}

	defer stream.Close()

	var b strings.Builder

	for {
		r, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return client.ChatCompletionResponse{}, err
		}
		//logger.WithField("stream response", fmt.Sprintf("%+v", r)).Trace("Received response from OpenAI API")
		delta := r.Choices[0].Delta.Content
		if _, err := b.WriteString(delta); err != nil {
			return client.ChatCompletionResponse{}, err
		}
		if _, err := w.Write([]byte(delta)); err != nil {
			return client.ChatCompletionResponse{}, err
		}

	}
	w.Write([]byte("\n\n"))

	message := client.Message{
		Content: b.String(),
		Role:    client.Assistant,
	}

	return client.ChatCompletionResponse{Choices: []client.Message{message}}, nil
}

var roleMapping = map[client.Role]string{
	client.User:      openai.ChatMessageRoleUser,
	client.System:    openai.ChatMessageRoleSystem,
	client.Assistant: openai.ChatMessageRoleAssistant,
}

func TranslateRequest(clientReq client.ChatCompletionRequest) (openai.ChatCompletionRequest, error) {
	req := openai.ChatCompletionRequest{
		Model:       clientReq.Model,
		Messages:    []openai.ChatCompletionMessage{},
		MaxTokens:   clientReq.MaxTokens,
		Temperature: clientReq.Temperature,
	}

	if topP, ok := clientReq.CustomParams["top_p"]; ok {
		req.TopP, ok = topP.(float32)
		if !ok {
			return openai.ChatCompletionRequest{}, fmt.Errorf("top_p must be a float32")
		}
	}

	if presencePenalty, ok := clientReq.CustomParams["presence_penalty"]; ok {
		req.PresencePenalty, ok = presencePenalty.(float32)
		if !ok {
			return openai.ChatCompletionRequest{}, fmt.Errorf("presence_penalty must be a float32")
		}
	}

	if frequencyPenalty, ok := clientReq.CustomParams["frequency_penalty"]; ok {
		req.FrequencyPenalty, ok = frequencyPenalty.(float32)
		if !ok {
			return openai.ChatCompletionRequest{}, fmt.Errorf("frequency_penalty must be a float32")
		}
	}

	if stop, ok := clientReq.CustomParams["stop"]; ok {
		req.Stop = stop.([]string)
	}

	if logitBias, ok := clientReq.CustomParams["logit_bias"]; ok {
		logitBiasMap, convOk := logitBias.(map[string]int)
		if !convOk {
			return openai.ChatCompletionRequest{}, fmt.Errorf("logit_bias must be a map[string]int")
		}
		req.LogitBias = logitBiasMap
	}

	// FIXME(ryszrd): rethink how should I handle n.
	if n, ok := clientReq.CustomParams["n"]; ok {
		req.N = n.(int)
	}

	if user, ok := clientReq.CustomParams["user"]; ok {
		req.User = user.(string)
	}

	for _, message := range clientReq.Messages {
		req.Messages = append(req.Messages, openai.ChatCompletionMessage{
			Content: message.Content,
			Role:    roleMapping[message.Role],
		})
	}

	return req, nil

}

// TranslateResponse translates a ChatCompletionResponse from the openai package to one from the client package
func TranslateResponse(openaiResp openai.ChatCompletionResponse) client.ChatCompletionResponse {
	// Create a new slice to hold the translated messages
	clientMessages := make([]client.Message, len(openaiResp.Choices))

	// Loop over the choices in the openai response
	for i, choice := range openaiResp.Choices {
		// Translate each choice's message to the client's Message type
		clientMessages[i] = client.Message{
			Content: choice.Message.Content,
			Role:    client.Role(choice.Message.Role),
		}
	}

	// Return a new ChatCompletionResponse from the client package, using the translated messages
	return client.ChatCompletionResponse{
		Choices: clientMessages,
	}
}
