package agent

import (
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai"
	log "github.com/sirupsen/logrus"
)

type Agent struct {
	name     string
	client   Client
	Messages []openai.ChatCompletionMessage
	config   agentConfig
}

func New(name string, options ...Option) *Agent {
	ag := &Agent{
		name: name,
	}

	for _, opt := range options {
		opt(&ag.config)
	}

	return ag
}

// System sends a System message to the actor.
func (ag *Agent) System(message string) {
	ag.Messages = append(ag.Messages, openai.ChatCompletionMessage{
		Content: message,
		Role:    "system",
	})
}

// Listen sends a User message to the actor.
func (ag *Agent) Listen(message string) {
	ag.Messages = append(ag.Messages, openai.ChatCompletionMessage{
		Content: message,
		Role:    "user",
	})
}

// Respond gets a response from the actor, basing on the current conversation.
func (ag *Agent) Respond(ctx context.Context, options ...Option) (message string, err error) {

	logger := log.WithField("actor", ag.name)

	cfg := ag.config.clone()
	for _, opt := range options {
		opt(&cfg)
	}
	req := cfg.chatCompletionRequest()
	req.Messages = ag.Messages

	logger.WithField("request", fmt.Sprintf("%+v", req)).Info("Sending request")
	resp, err := cfg.Client.CreateChatCompletion(ctx, req)
	logger.WithError(err).WithField("response", fmt.Sprintf("%+v", resp)).Debug("Received response from OpenAI API")
	if err != nil {
		logger.WithError(err).Error("Failed to send request to OpenAI API")
		return "", err
	}
	logger.WithField("response", fmt.Sprintf("%+v", resp)).Info("Received response from OpenAI API")

	msg := resp.Choices[0].Message
	ag.Messages = append(ag.Messages, msg)

	return msg.Content, nil
}
