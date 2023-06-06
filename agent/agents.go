package agent

import (
	"context"
	"errors"
	"fmt"

	"github.com/ryszard/agency/client"
	log "github.com/sirupsen/logrus"
)

// Agent is an interface for a chat agent. It is the main interface that
// you will use to interact with the AI. You can create an agent using
// the New function.
type Agent interface {
	// Name returns the name of the agent.
	Name() string
	// Listen sends a message to the agent and appends the response to the
	// agent's messages. An agent may also support passing additional data to
	// Listen, but this is not required. An Agent may return an error if you
	// pass data that it does not support. Listen will return the message that
	// was passed to the agent. This will be identical to the message that you
	// passed to Listen in the most basic case, but may be different if the
	// agent modifies the message.
	Listen(message string, data ...any) (string, error)

	// System sends a system message to the agent and appends the response
	// to the agent's messages. An agent may also support passing additional
	// data to System, but this is not required. An Agent may return an error
	// if you pass data that it does not support. System will return the
	// message that was passed to the agent. This will be identical to the
	// message that you passed to System in the most basic case, but may be
	// different if the agent modifies the message.
	System(message string, data ...any) (string, error)

	// Respond gets a response from the agent, basing on the current
	// conversation. The options passed to Respond will be applied for this
	// call, but won't affect subsequent calls.
	Respond(ctx context.Context, options ...Option) (message string, err error)

	// Inject introduces a message into the ongoing conversation, giving the
	// impression that the agent produced it. This method returns the message as
	// processed by the agent, which will be identical to the input in simple
	// cases. However, depending on the agent's behavior, the returned message
	// may be different from the input.
	Inject(message string, data ...any) (string, error)

	// Messages returns all messages that the agent has sent and received.
	Messages() []client.Message

	// Append appends messages to the agent's messages.
	Append(messages ...client.Message)

	// Config returns the agent's config.
	Config() Config
}

// New returns a new Agent with the given name and options. It will be backed by
// a BaseAgent.
func New(name string, options ...Option) Agent {
	return NewBaseAgent(name, options...)
}

var _ Agent = &BaseAgent{}

// BaseAgent is a basic implementation of the Agent interface. You most likely
// want to use it as a base for your own agents.
type BaseAgent struct {
	name     string
	messages []client.Message
	config   Config
}

func (ag *BaseAgent) Config() Config {
	return ag.config
}

func (ag *BaseAgent) Messages() []client.Message {
	return ag.messages
}

func (ag *BaseAgent) Name() string {
	return ag.name
}

// NewBaseAgent returns a BaseAgent with the given name and options.
func NewBaseAgent(name string, options ...Option) *BaseAgent {
	ag := &BaseAgent{
		name: name,
	}

	for _, opt := range options {
		opt(&ag.config)
	}

	return ag
}

func (ag *BaseAgent) Append(messages ...client.Message) {
	ag.messages = append(ag.messages, messages...)
}

func (ag *BaseAgent) System(message string, data ...any) (string, error) {
	if len(data) > 0 {
		return "", errors.New("this agent does not support passing data to System")
	}
	ag.Append(client.Message{
		Content: message,
		Role:    client.System,
	})
	return message, nil
}

func (ag *BaseAgent) Listen(message string, data ...any) (string, error) {
	log.WithField("message", message).WithField("agent", ag.Name()).Trace("Listen")
	if len(data) > 0 {
		return "", errors.New("this agent does not support passing data to Listen")
	}
	ag.Append(client.Message{
		Content: message,
		Role:    client.User,
	})
	log.WithField("messages", ag.Messages()).WithField("agent", ag.Name()).Trace("Listen Messages")

	return message, nil
}

func (ag *BaseAgent) Inject(message string, data ...any) (string, error) {
	if len(data) > 0 {
		return "", errors.New("this agent does not support passing data to Inject")
	}
	ag.Append(client.Message{
		Content: message,
		Role:    client.Assistant,
	})
	return message, nil
}

func (ag *BaseAgent) createRequest(options []Option) (Config, client.ChatCompletionRequest) {
	cfg := ag.config.clone()
	for _, opt := range options {
		opt(&cfg)
	}
	req := cfg.chatCompletionRequest()
	req.Messages = ag.Messages()

	return cfg, req
}

func (ag *BaseAgent) Respond(ctx context.Context, options ...Option) (message string, err error) {
	logger := log.WithField("agent", ag.name)
	logger.Debug("Responding to message")
	cfg, req := ag.createRequest(options)

	if cfg.Memory != nil {
		log.Debug("Using memory")
		newMessages, err := cfg.Memory(ctx, cfg, ag.messages)
		if err != nil {
			log.WithError(err).Error("Failed to use memory")
			return "", err
		}
		ag.messages = newMessages
	}

	logger.WithField("request", fmt.Sprintf("%+v", req)).Info("Sending request")
	resp, err := cfg.Client.CreateChatCompletion(ctx, req)
	logger.WithError(err).WithField("response", fmt.Sprintf("%+v", resp)).Debug("Received response from client")
	if err != nil {
		logger.WithError(err).Error("Failed to send request to OpenAI API")
		return "", err
	}
	logger.WithField("response", fmt.Sprintf("%+v", resp)).Info("Received response from client")

	msg := resp.Choices[0]
	ag.Append(msg)

	return msg.Content, nil
}
