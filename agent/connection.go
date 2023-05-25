package agent

import (
	"context"
	"errors"
	"time"

	"github.com/sashabaranov/go-openai"
	log "github.com/sirupsen/logrus"
)

// Client is an interface for the OpenAI API client. It's main purpose is to
// make testing easier.
type Client interface {
	CreateChatCompletion(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error)
	CreateChatCompletionStream(context.Context, openai.ChatCompletionRequest) (*openai.ChatCompletionStream, error)
}

func retry(client *retryingClient, fn func() (any, error)) (any, error) {
	waitMultiplier := 1
	for i := 0; i < client.maxRetries; i++ {
		resp, err := fn()
		if err == nil {
			return resp, nil
		}
		log.WithError(err).Error("Error from OpenAI API")
		wait := time.Duration(waitMultiplier) * client.baseWait
		log.WithField("wait", wait).Info("Waiting before retrying")
		if wait > client.maxWait {
			wait = client.maxWait
		}
		time.Sleep(wait)
		waitMultiplier *= 2
	}

	return nil, errors.New("max retries exceeded")
}

type retryingClient struct {
	client     Client
	baseWait   time.Duration
	maxWait    time.Duration
	maxRetries int
}

// Retrying wraps a Client and retries requests if they fail, using exponential
// backoff. After the first failure, it will wait baseWait, then twice that,
// until it reaches either maxWait, or has made maxRetries attempts. Pass -1 to
// maxRetries to retry forever. Retrying will panic if maxRetries is 0.
func Retrying(client Client, baseWait time.Duration, maxWait time.Duration, maxRetries int) Client {
	if maxRetries == 0 {
		panic("maxRetries must not be 0")
	}
	return &retryingClient{
		client:     client,
		baseWait:   baseWait,
		maxWait:    maxWait,
		maxRetries: maxRetries,
	}
}

func (client *retryingClient) CreateChatCompletion(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
	log.WithField("req", req).Info("CreateChatCompletion")
	resp, err := retry(client, func() (any, error) {
		return client.client.CreateChatCompletion(ctx, req)
	})
	if err != nil {
		return openai.ChatCompletionResponse{}, err
	}
	return resp.(openai.ChatCompletionResponse), nil
}

func (client *retryingClient) CreateChatCompletionStream(ctx context.Context, req openai.ChatCompletionRequest) (*openai.ChatCompletionStream, error) {
	log.WithField("req", req).Info("CreateChatCompletionStream")
	resp, err := retry(client, func() (any, error) {
		return client.client.CreateChatCompletionStream(ctx, req)
	})
	if err != nil {
		return nil, err
	}
	return resp.(*openai.ChatCompletionStream), nil
}
