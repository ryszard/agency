package client

import (
	"context"

	log "github.com/sirupsen/logrus"
)

// Limiter is an interface for rate limiters. A recommended implementation is
// golang.org/x/time/rate.Limiter .
type Limiter interface {
	Wait(ctx context.Context) error
}

type rateLimitingClient struct {
	client  Client
	limiter Limiter
}

// RateLimiting wraps a Client and rate limits requests using the provided
// limiter. If the limiter returns an error, RateLimiting will return that
// error, otherwise it will return the error from the wrapped Client.
func RateLimiting(client Client, limiter Limiter) Client {
	return &rateLimitingClient{
		client:  client,
		limiter: limiter,
	}
}

func (client *rateLimitingClient) CreateChatCompletion(ctx context.Context, req ChatCompletionRequest) (ChatCompletionResponse, error) {
	log.WithField("req", req).Info("CreateChatCompletion")
	if err := client.limiter.Wait(ctx); err != nil {
		return ChatCompletionResponse{}, err
	}
	return client.client.CreateChatCompletion(ctx, req)
}

func (client *rateLimitingClient) CreateChatCompletionStream(ctx context.Context, req ChatCompletionStreamRequest) (ChatCompletionStream, error) {
	log.WithField("req", req).Info("CreateChatCompletionStream")
	if err := client.limiter.Wait(ctx); err != nil {
		return nil, err
	}
	return client.client.CreateChatCompletionStream(ctx, req)
}
