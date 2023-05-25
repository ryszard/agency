package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"

	"github.com/sashabaranov/go-openai"
	log "github.com/sirupsen/logrus"
)

type Cache interface {
	Get(key []byte) ([]byte, error)
	Set(key []byte, value []byte) error
}

// WithCache wraps an Agent with a cache. The cache is used to store the
// responses to the agent's messages. If the agent is asked to respond to
// a message that it has already responded to, it will return the cached
// response instead of making a new request to the API. This can be useful
// while developing, as it will save you from using up your API requests.
func WithCache(ag Agent, cache Cache) Agent {
	return &cachedAgent{
		Agent: ag,
		cache: cache,
	}
}

type cachedAgent struct {
	Agent
	cache Cache
}

func (ag *cachedAgent) hash(options ...Option) (string, error) {

	// convert ag.Messages() to JSON and put it in data
	messagesJSON, err := json.Marshal(ag.Messages())
	if err != nil {
		log.WithError(err).Error("failed to marshal messages")
		return "", err
	}

	cfg := ag.Config()
	for _, opt := range options {
		opt(&cfg)
	}

	configJSON, err := json.Marshal(cfg)
	if err != nil {
		log.WithError(err).Error("failed to marshal config")
		return "", err
	}

	data := append(messagesJSON, configJSON...)

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

func (ag *cachedAgent) Respond(ctx context.Context, options ...Option) (message string, err error) {
	hash, err := ag.hash(options...)
	if err != nil {
		return "", err
	}
	if cached, err := ag.cache.Get([]byte(hash)); err == nil {
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

func (ag *cachedAgent) RespondStream(ctx context.Context, w io.Writer, options ...Option) (message string, err error) {
	hash, err := ag.hash(options...)
	if err != nil {
		return "", err
	}
	if cached, err := ag.cache.Get([]byte(hash)); err == nil {
		ag.Append(openai.ChatCompletionMessage{
			Content: string(cached),
			Role:    openai.ChatMessageRoleAssistant,
		})
		w.Write([]byte(cached))
		w.Write([]byte("\n"))
		return string(cached), nil
	}

	message, err = ag.Agent.RespondStream(ctx, w, options...)
	if err != nil {
		return "", err
	}

	err = ag.cache.Set([]byte(hash), []byte(message))
	return message, err
}
