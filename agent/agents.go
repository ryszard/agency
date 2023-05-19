package agent

import (
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai"
	log "github.com/sirupsen/logrus"
)

type Actor struct {
	name     string
	client   Client
	Messages []openai.ChatCompletionMessage
}

func New(name string, client *openai.Client) *Actor {
	return &Actor{
		name:   name,
		client: client,
	}
}

// System sends a System message to the actor.
func (ac *Actor) System(message string) {
	ac.Messages = append(ac.Messages, openai.ChatCompletionMessage{
		Content: message,
		Role:    "system",
	})
}

// Listen sends a User message to the actor.
func (ac *Actor) Listen(message string) {
	ac.Messages = append(ac.Messages, openai.ChatCompletionMessage{
		Content: message,
		Role:    "user",
	})
}

// Respond gets a response from the actor, basing on the current conversation.
func (ac *Actor) Respond(ctx context.Context) (message string, err error) {

	logger := log.WithField("actor", ac.name)
	req := openai.ChatCompletionRequest{
		Model:     openai.GPT3Dot5Turbo,
		Messages:  ac.Messages,
		MaxTokens: 300,
	}
	logger.WithField("request", fmt.Sprintf("%+v", req)).Info("Sending request")
	resp, err := ac.client.CreateChatCompletion(ctx, req)
	logger.WithError(err).WithField("response", fmt.Sprintf("%+v", resp)).Debug("Received response from OpenAI API")
	if err != nil {
		logger.WithError(err).Error("Failed to send request to OpenAI API")
		return "", err
	}
	logger.WithField("response", fmt.Sprintf("%+v", resp)).Info("Received response from OpenAI API")

	msg := resp.Choices[0].Message
	ac.Messages = append(ac.Messages, msg)

	return msg.Content, nil
}
