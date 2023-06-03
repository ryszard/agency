package client

import (
	"context"
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
)

// Cache is an interface for a cache that you need to pass to Cached. It is used
// to store the responses to the agent's messages.
type Cache interface {
	// Get returns the value for the given key. If the key does not exist,
	// ok will be false.
	Get(key []byte) (value []byte, ok bool, err error)
	Set(key []byte, value []byte) error
}

type CachedClient struct {
	client Client
	cache  Cache
}

// Cached wraps a client with a cache.
func Cached(client Client, cache Cache) *CachedClient {
	return &CachedClient{
		client: client,
		cache:  cache,
	}
}

// CreateChatCompletion implements Client
func (c *CachedClient) CreateChatCompletion(ctx context.Context, req ChatCompletionRequest) (ChatCompletionResponse, error) {
	log.SetLevel(log.DebugLevel)
	hash, err := req.hash()
	log.WithField("hash", fmt.Sprintf("%x", hash)).Debug("hashing request")
	if err != nil {
		return ChatCompletionResponse{}, err
	}

	val, ok, err := c.cache.Get(hash)
	if err != nil {
		return ChatCompletionResponse{}, err
	}

	if ok {
		log.Debug("cache hit")
		var resp ChatCompletionResponse
		if err := json.Unmarshal(val, &resp); err != nil {
			return ChatCompletionResponse{}, err
		}
		return resp, nil
	}
	log.Debug("cache miss")
	resp, err := c.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return ChatCompletionResponse{}, err
	}

	val, err = json.Marshal(resp)
	if err != nil {
		return ChatCompletionResponse{}, err
	}
	log.Debug("setting cache")
	if err := c.cache.Set(hash, val); err != nil {
		return ChatCompletionResponse{}, err
	}

	return resp, nil

}

// CreateChatCompletionStream implements Client
func (*CachedClient) CreateChatCompletionStream(ctx context.Context, req ChatCompletionStreamRequest) (ChatCompletionStream, error) {
	return nil, nil
}

var _ Client = (*CachedClient)(nil)
