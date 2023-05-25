package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/sashabaranov/go-openai"
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

// Cached wraps an Agent with a cache. The cache is used to store the
// responses to the agent's messages. If the agent is asked to respond to
// a message that it has already responded to, it will return the cached
// response instead of making a new request to the API. This can be useful
// while developing, as it will save you from using up your API requests.
func Cached(ag Agent, cache Cache) Agent {
	return &cachedAgent{
		Agent: ag,
		cache: cache,
	}
}

type cachedAgent struct {
	Agent
	cache Cache
}

func (ag *cachedAgent) hash(options ...Option) (string, Config, error) {

	messagesJSON, err := json.Marshal(ag.Messages())
	if err != nil {
		log.WithError(err).Error("failed to marshal messages")
		return "", Config{}, err
	}

	cfg := ag.Config()
	for _, opt := range options {
		opt(&cfg)
	}

	configJSON, err := json.Marshal(cfg)
	if err != nil {
		log.WithError(err).Error("failed to marshal config")
		return "", Config{}, err
	}

	data := append(messagesJSON, configJSON...)

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), cfg, nil
}

func (ag *cachedAgent) Respond(ctx context.Context, options ...Option) (message string, err error) {
	hash, _, err := ag.hash(options...)
	if err != nil {
		return "", err
	}
	cached, ok, err := ag.cache.Get([]byte(hash))
	if err != nil {
		return "", err
	}

	if ok {
		log.WithField("hash", hash).Debug("cached response")
		ag.Append(openai.ChatCompletionMessage{
			Content: string(cached),
			Role:    openai.ChatMessageRoleAssistant,
		})
		return string(cached), nil
	}

	message, err = ag.Agent.Respond(ctx, options...)
	if err != nil {
		return "", err
	}

	err = ag.cache.Set([]byte(hash), []byte(message))
	return message, err
}
