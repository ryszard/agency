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

// Agent is an interface for a chat agent. It is the main interface that
// you will use to interact with the AI. You can create an agent using
// the New function.
type Agent interface {
	// Name returns the name of the agent.
	Name() string
	// Listen sends a message to the agent and appends the response to the
	// agent's messages. An agent may also support passing additional data
	// to Listen, but this is not required. An Agent may return an error
	// if you pass data that it does not support.
	Listen(message string, data ...any) error

	// System sends a system message to the agent and appends the response
	// to the agent's messages. An agent may also support passing additional
	// data to System, but this is not required. An Agent may return an error
	// if you pass data that it does not support.
	System(message string, data ...any) error

	// Respond gets a response from the agent, basing on the current
	// conversation. The options passed to Respond will be applied for this
	// call, but won't affect subsequent calls.
	Respond(ctx context.Context, options ...Option) (message string, err error)

	// RespondStream gets a response from the agent, basing on the current
	// conversation. The options passed to Respond will be applied for this
	// call, but won't affect subsequent calls. The response will be streamed
	// to the writer, but also returned as a string.
	RespondStream(ctx context.Context, options ...Option) (message string, err error)

	// Messages returns all messages that the agent has sent and received.
	Messages() []openai.ChatCompletionMessage

	// Append appends messages to the agent's messages.
	Append(messages ...openai.ChatCompletionMessage)

	// Config returns the agent's config.
	Config() Config
}

var _ Agent = &BaseAgent{}

// BaseAgent is a basic implementation of the Agent interface.
type BaseAgent struct {
	name     string
	messages []openai.ChatCompletionMessage
	config   Config
}

func (ag *BaseAgent) Config() Config {
	return ag.config
}

func (ag *BaseAgent) Messages() []openai.ChatCompletionMessage {
	return ag.messages
}

func (ag *BaseAgent) Name() string {
	return ag.name
}

func New(name string, options ...Option) *BaseAgent {
	ag := &BaseAgent{
		name: name,
	}

	for _, opt := range options {
		opt(&ag.config)
	}

	return ag
}

func (ag *BaseAgent) Append(messages ...openai.ChatCompletionMessage) {
	ag.messages = append(ag.messages, messages...)
}

func (ag *BaseAgent) System(message string, data ...any) error {
	if len(data) > 0 {
		return errors.New("this agent does not support passing data to System")
	}
	ag.Append(openai.ChatCompletionMessage{
		Content: message,
		Role:    "system",
	})
	return nil
}

func (ag *BaseAgent) Listen(message string, data ...any) error {
	log.WithField("message", message).WithField("agent", ag.Name()).Trace("Listen")
	if len(data) > 0 {
		return errors.New("this agent does not support passing data to Listen")
	}
	ag.Append(openai.ChatCompletionMessage{
		Content: message,
		Role:    "user",
	})
	log.WithField("messages", ag.Messages()).WithField("agent", ag.Name()).Trace("Listen Messages")

	return nil
}

func (ag *BaseAgent) createRequest(options []Option) (Config, openai.ChatCompletionRequest) {
	cfg := ag.config.clone()
	for _, opt := range options {
		opt(&cfg)
	}
	req := cfg.chatCompletionRequest()
	req.Messages = ag.Messages()

	return cfg, req
}

func (ag *BaseAgent) Respond(ctx context.Context, options ...Option) (message string, err error) {

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
	ag.Append(msg)

	return msg.Content, nil
}

func (ag *BaseAgent) RespondStream(ctx context.Context, options ...Option) (string, error) {
	cfg, req := ag.createRequest(options)
	logger := log.WithField("actor", ag.name)

	req.Stream = true
	logger.WithFields(log.Fields{
		"request": fmt.Sprintf("%+v", req),
		"stream":  true,
	}).Info("RespondStream: Sending request")
	stream, err := cfg.Client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return "", err
	}

	defer stream.Close()

	if _, err := fmt.Fprintf(cfg.out(), "%v:\n\n", ag.name); err != nil {
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
		if _, err := cfg.out().Write([]byte(delta)); err != nil {
			return "", err
		}

	}
	cfg.out().Write([]byte("\n\n"))

	message := openai.ChatCompletionMessage{
		Content: b.String(),
		Role:    openai.ChatMessageRoleAssistant,
	}

	ag.Append(message)
	return b.String(), nil
}
