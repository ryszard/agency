package agent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/sashabaranov/go-openai"
	log "github.com/sirupsen/logrus"
)

type Agent struct {
	name     string
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

func (ag *Agent) createRequest(options []Option) (agentConfig, openai.ChatCompletionRequest) {
	cfg := ag.config.clone()
	for _, opt := range options {
		opt(&cfg)
	}
	req := cfg.chatCompletionRequest()
	req.Messages = ag.Messages

	return cfg, req
}

// Respond gets a response from the actor, basing on the current conversation.
func (ag *Agent) Respond(ctx context.Context, options ...Option) (message string, err error) {

	logger := log.WithField("actor", ag.name)

	cfg, req := ag.createRequest(options)

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

// RespondStream gets a response from the actor, basing on the current
// conversation. It will stream the response to the provided writer.
func (ag *Agent) RespondStream(ctx context.Context, w io.Writer, options ...Option) (message string, err error) {
	cfg, req := ag.createRequest(options)
	logger := log.WithField("actor", ag.name)

	req.Stream = true

	stream, err := cfg.Client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return "", err
	}

	defer stream.Close()

	if _, err := fmt.Fprintf(w, "%v:\n\n", ag.name); err != nil {
		return "", err
	}

	var b strings.Builder

	for {
		r, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return "", err
		}
		logger.WithField("stream response", fmt.Sprintf("%+v", r)).Trace("Received response from OpenAI API")
		delta := r.Choices[0].Delta.Content
		if _, err := b.WriteString(delta); err != nil {
			return "", err
		}
		if _, err := w.Write([]byte(delta)); err != nil {
			return "", err
		}

	}
	w.Write([]byte("\n\n"))
	return b.String(), nil
}
