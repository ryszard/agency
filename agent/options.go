package agent

import (
	"io"

	"github.com/ryszard/agency/client"
)

// Option can be used to configure an agent. Note that the order in which you
// pass options to an agent is important, as they will be applied in the same
// order. So, for example, if you pass WithModel twice with different models,
// the second one will overwrite the first one.
type Option func(*Config)

type Config struct {
	// openai.ChatCompletionRequest fields
	RequestTemplate client.ChatCompletionRequest

	// other fields
	Client client.Client `json:"-"`

	// Memory is the agent's memory.
	Memory Memory `json:"-"`
}

func (ac Config) chatCompletionRequest() client.ChatCompletionRequest {

	// FIXME(ryszard): handle Streaming.
	// return a deep copy of the req
	req := ac.RequestTemplate
	if ac.RequestTemplate.CustomParams != nil {

		req.CustomParams = make(map[string]interface{}, len(ac.RequestTemplate.CustomParams))
		for k, v := range ac.RequestTemplate.CustomParams {
			req.CustomParams[k] = v
		}
	}
	return req
}

func (ac Config) clone() Config {
	cfg := ac
	cfg.RequestTemplate = ac.chatCompletionRequest()
	return cfg
}

// WithConfig will make the agent use the given config.
func WithConfig(cfg Config) Option {
	return func(ac *Config) {
		*ac = cfg
	}
}

// WithModel configures the agent to use the given model. The model must be
// the ID of a completion model.
func WithModel(model string) Option {
	return func(ac *Config) {
		ac.RequestTemplate.Model = model
	}
}

// WithMaxTokens sets MaxTokens for the agent.
func WithMaxTokens(maxTokens int) Option {
	return func(ac *Config) {
		ac.RequestTemplate.MaxTokens = maxTokens
	}
}

// WithTemperature sets Temperature for the agent.
func WithTemperature(temperature float32) Option {
	return func(ac *Config) {
		ac.RequestTemplate.Temperature = temperature
	}
}

// WithClient will make the agent use the given client.
func WithClient(cl client.Client) Option {
	return func(ac *Config) {
		ac.Client = cl
	}
}

type nullWriter struct{}

func (nw nullWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

var _ io.Writer = nullWriter{}

// WithStreaming makes the agent to stream its responses to the provided writer.
// Note that this will cause the agent to use streaming calls to the OpenAI API.
func WithStreaming(w io.Writer) Option {
	return func(ac *Config) {
		ac.RequestTemplate.Stream = w
	}
}

// WithoutStreaming suppresses streaming of the agent's responses.
func WithoutStreaming() Option {
	return func(ac *Config) {
		ac.RequestTemplate.Stream = nil
	}
}

// WithMemory sets the agent's memory.
func WithMemory(memory Memory) Option {
	return func(ac *Config) {
		ac.Memory = memory
	}
}
func WithCustomParams(params map[string]interface{}) Option {
	return func(ac *Config) {
		ac.RequestTemplate.CustomParams = params
	}
}
