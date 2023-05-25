package agent

import (
	"io"

	"github.com/sashabaranov/go-openai"
)

// Option is a function that configures the agent.
type Option func(*Config)

type Config struct {
	// openai.ChatCompletionRequest fields
	Model            string
	MaxTokens        int
	Temperature      float32
	TopP             float32
	Stream           bool
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
}

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
		Stream:           ac.Stream,
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
		Stream:           ac.Stream,
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
func WithConfig(cfg Config) func(*Config) {
	return func(ac *Config) {
		*ac = cfg
	}
}

// WithModel configures the agent to use the given model. The model must be
// the ID of a completion model.
func WithModel(model string) func(*Config) {
	return func(ac *Config) {
		ac.Model = model
	}
}

// WithMaxTokens creates a function that sets the MaxTokens field of
// agentConfig. The returned function can be used as an option to configure the
// agent.
func WithMaxTokens(maxTokens int) func(*Config) {
	return func(ac *Config) {
		ac.MaxTokens = maxTokens
	}
}

// WithTemperature creates a function that sets the Temperature field of
// agentConfig. The returned function can be used as an option to configure the
// agent.
func WithTemperature(temperature float32) func(*Config) {
	return func(ac *Config) {
		ac.Temperature = temperature
	}
}

// WithTopP creates a function that sets the TopP field of agentConfig. The
// returned function can be used as an option to configure the agent.
func WithTopP(topP float32) func(*Config) {
	return func(ac *Config) {
		ac.TopP = topP
	}
}

// WithStream creates a function that sets the Stream field of agentConfig. The
// returned function can be used as an option to configure the agent.
func WithStream(stream bool) func(*Config) {
	return func(ac *Config) {
		ac.Stream = stream
	}
}

// WithStop creates a function that sets the Stop field of agentConfig. The
// returned function can be used as an option to configure the agent.
func WithStop(stop []string) func(*Config) {
	return func(ac *Config) {
		ac.Stop = stop
	}
}

// WithPresencePenalty creates a function that sets the PresencePenalty field of
// agentConfig. The returned function can be used as an option to configure the
// agent.
func WithPresencePenalty(presencePenalty float32) func(*Config) {
	return func(ac *Config) {
		ac.PresencePenalty = presencePenalty
	}
}

// WithFrequencyPenalty creates a function that sets the FrequencyPenalty field
// of agentConfig. The returned function can be used as an option to configure
// the agent.
func WithFrequencyPenalty(frequencyPenalty float32) func(*Config) {
	return func(ac *Config) {
		ac.FrequencyPenalty = frequencyPenalty
	}
}

// WithLogitBias creates a function that sets the LogitBias field of
// agentConfig. The returned function can be used as an option to configure the
// agent.
func WithLogitBias(logitBias map[string]int) func(*Config) {
	return func(ac *Config) {
		ac.LogitBias = logitBias
	}
}

// WithUser creates a function that sets the User field of agentConfig. The
// returned function can be used as an option to configure the agent.
func WithUser(user string) func(*Config) {
	return func(ac *Config) {
		ac.User = user
	}
}

// WithClient creates a function that sets the Client field of agentConfig. The
// returned function can be used as an option to configure the agent.
func WithClient(client Client) func(*Config) {
	return func(ac *Config) {
		ac.Client = client
	}
}

type nullWriter struct{}

func (nw nullWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

var _ io.Writer = nullWriter{}

func WithOutput(w io.Writer) func(*Config) {
	return func(ac *Config) {
		ac.Output = w
	}
}
