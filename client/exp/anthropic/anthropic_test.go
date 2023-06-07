package anthropic

import (
	"reflect"
	"testing"

	"github.com/madebywelch/anthropic-go/pkg/anthropic"
	"github.com/ryszard/agency/client"
	"github.com/stretchr/testify/assert"
)

func TestTranslateRequest(t *testing.T) {
	testCases := []struct {
		name     string
		input    client.ChatCompletionRequest
		err      error
		expected anthropic.CompletionRequest
	}{
		{
			name: "Basic Translation",
			input: client.ChatCompletionRequest{
				Model:       "claude-v1",
				MaxTokens:   100,
				Temperature: 0.5,
				CustomParams: map[string]interface{}{
					"stop_sequences": []string{"stop1", "stop2"},
					"top_k":          10,
					"top_p":          0.9,
				},
			},
			err: nil,
			expected: anthropic.CompletionRequest{
				Model:             "claude-v1",
				MaxTokensToSample: 100,
				Temperature:       0.5,
				StopSequences:     []string{"stop1", "stop2"},
				TopK:              10,
				TopP:              0.9,
			},
		},
		{
			name: "Messages",
			input: client.ChatCompletionRequest{
				Model:     "claude-v1",
				MaxTokens: 100,
				Messages: []client.Message{
					{Role: client.User, Content: "1"},
					{Role: client.Assistant, Content: "2"},
					{Role: client.User, Content: "3"},
				},
			},
			err: nil,
			expected: anthropic.CompletionRequest{
				Prompt: `Human: 1

Assistant: 2

Human: 3

Assistant:`,
				Model:             "claude-v1",
				MaxTokensToSample: 100,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := TranslateRequest(&tc.input)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, &tc.expected, actual)
		})
	}
}

func TestTranslateMessages(t *testing.T) {

	for _, tc := range []struct {
		name     string
		input    []client.Message
		expected string
	}{
		{
			name: "Basic Translation",
			input: []client.Message{
				{Role: client.User, Content: "1"},
				{Role: client.Assistant, Content: "2"},
				{Role: client.User, Content: "3"},
			},
			expected: `Human: 1

Assistant: 2

Human: 3

Assistant:`},
		{
			name: "Basic Translation",
			input: []client.Message{
				{Role: client.System, Content: "1"},
				{Role: client.Assistant, Content: "2"},
				{Role: client.User, Content: "3"},
			},
			expected: `Human: 1

Assistant: 2

Human: 3

Assistant:`},
	} {

		actual := TranslateMessages(tc.input)

		assert.Equal(t, tc.expected, actual)
	}
}

func TestTranslateResponse(t *testing.T) {
	tests := []struct {
		name     string
		input    *anthropic.CompletionResponse
		expected client.ChatCompletionResponse
	}{
		{
			name: "Test 1: Basic translate",
			input: &anthropic.CompletionResponse{
				Completion: "Test completion 1",
				StopReason: "stop_sequence",
				Stop:       "Test stop 1",
			},
			expected: client.ChatCompletionResponse{
				Choices: []client.Message{
					{
						Content: "Test completion 1",
						Role:    client.Assistant,
					},
				},
			},
		},
		// More tests can be added here...
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := TranslateResponse(tt.input)
			if !reflect.DeepEqual(actual, tt.expected) {
				t.Errorf("TranslateResponse(%v) = %v, want %v", tt.input, actual, tt.expected)
			}
		})
	}
}
