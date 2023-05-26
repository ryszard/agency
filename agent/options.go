package agent

import (
	"io"

	"github.com/sashabaranov/go-openai"
)

// Option can be used to configure an agent. Note that the order in which you
// pass options to an agent is important, as they will be applied in the same
// order. So, for example, if you pass WithModel twice with different models,
// the second one will overwrite the first one.
type Option func(*Config)

type Config struct {
	// openai.ChatCompletionRequest fields
	Model            string
	MaxTokens        int
	Temperature      float32
	TopP             float32
	Stop             []string
	PresencePenalty  float32
	FrequencyPenalty float32
	LogitBias        map[string]int
	User             string

	// other fields
	Client Client `json:"-"`

	// Output is the writer to which the agent will write its output. If nil,
	// the agent's output will be discarded. This is useful for streaming.
	Output io.Writer `json:"-"`

	// Memory is the agent's memory.
	Memory Memory `json:"-"`
}

// Stream returns true if the agent is configured to stream its output.
func (cfg Config) Stream() bool {
	return cfg.Output != nil
}

// TODO(ryszard): Remove out.

func (ac Config) out() io.Writer {
	if ac.Output != nil {
		return ac.Output
	}
	return nullWriter{}
}

func (ac Config) chatCompletionRequest() openai.ChatCompletionRequest {
	return openai.ChatCompletionRequest{
		Model:            ac.Model,
		MaxTokens:        ac.MaxTokens,
		Temperature:      ac.Temperature,
		TopP:             ac.TopP,
		Stream:           ac.Stream(),
		Stop:             ac.Stop,
		PresencePenalty:  ac.PresencePenalty,
		FrequencyPenalty: ac.FrequencyPenalty,
		LogitBias:        ac.LogitBias,
		User:             ac.User,
	}
}

func (ac Config) clone() Config {
	return Config{
		Model:            ac.Model,
		MaxTokens:        ac.MaxTokens,
		Temperature:      ac.Temperature,
		TopP:             ac.TopP,
		Stop:             ac.Stop,
		PresencePenalty:  ac.PresencePenalty,
		FrequencyPenalty: ac.FrequencyPenalty,
		LogitBias:        ac.LogitBias,
		User:             ac.User,
		Client:           ac.Client,
		Output:           ac.Output,
	}
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
		ac.Model = model
	}
}

// WithMaxTokens sets MaxTokens for the agent.
func WithMaxTokens(maxTokens int) Option {
	return func(ac *Config) {
		ac.MaxTokens = maxTokens
	}
}

// WithTemperature sets Temperature for the agent.
func WithTemperature(temperature float32) Option {
	return func(ac *Config) {
		ac.Temperature = temperature
	}
}

// WithTopP sets
func WithTopP(topP float32) Option {
	return func(ac *Config) {
		ac.TopP = topP
	}
}

// WithStop sets Stop for the agent.
func WithStop(stop []string) Option {
	return func(ac *Config) {
		ac.Stop = stop
	}
}

// WithPresencePenalty sets
func WithPresencePenalty(presencePenalty float32) Option {
	return func(ac *Config) {
		ac.PresencePenalty = presencePenalty
	}
}

// WithFrequencyPenalty sets FrequencyPenalty for the agent.
func WithFrequencyPenalty(frequencyPenalty float32) Option {
	return func(ac *Config) {
		ac.FrequencyPenalty = frequencyPenalty
	}
}

// WithLogitBias sets LogitBias for the agent.
func WithLogitBias(logitBias map[string]int) Option {
	return func(ac *Config) {
		ac.LogitBias = logitBias
	}
}

// WithUser sets User for the agent.
func WithUser(user string) Option {
	return func(ac *Config) {
		ac.User = user
	}
}

// WithClient will make the agent use the given client.
func WithClient(client Client) Option {
	return func(ac *Config) {
		ac.Client = client
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
		ac.Output = w
	}
}

// WithoutStreaming suppresses streaming of the agent's responses.
func WithoutStreaming() Option {
	return func(ac *Config) {
		ac.Output = nil
	}
}

// WithMemory sets the agent's memory.
func WithMemory(memory Memory) Option {
	return func(ac *Config) {
		ac.Memory = memory
	}
}
