package openai

import (
	"testing"

	"github.com/ryszard/agency/client"
	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
)

func TestTranslateResponse(t *testing.T) {
	openaiResponse := openai.ChatCompletionResponse{
		ID:      "test-id",
		Object:  "test-object",
		Created: 123456789,
		Model:   "test-model",
		Choices: []openai.ChatCompletionChoice{
			{
				Index: 0,
				Message: openai.ChatCompletionMessage{
					Role:    "assistant",
					Content: "Hello, how can I assist you today?",
					Name:    "test-name",
				},
				FinishReason: "stop",
			},
		},
		Usage: openai.Usage{
			// Fill in with appropriate data
		},
	}

	expectedClientResponse := client.ChatCompletionResponse{
		Choices: []client.Message{
			{
				Content: "Hello, how can I assist you today?",
				Role:    "assistant",
			},
		},
	}

	actualClientResponse := TranslateResponse(openaiResponse)

	if len(actualClientResponse.Choices) != len(expectedClientResponse.Choices) {
		t.Errorf("Expected number of choices %v, but got %v", len(expectedClientResponse.Choices), len(actualClientResponse.Choices))
	}

	for i := range actualClientResponse.Choices {
		if actualClientResponse.Choices[i].Content != expectedClientResponse.Choices[i].Content {
			t.Errorf("Expected content '%s', but got '%s'", expectedClientResponse.Choices[i].Content, actualClientResponse.Choices[i].Content)
		}
		if actualClientResponse.Choices[i].Role != expectedClientResponse.Choices[i].Role {
			t.Errorf("Expected role '%s', but got '%s'", expectedClientResponse.Choices[i].Role, actualClientResponse.Choices[i].Role)
		}
	}
}

func TestTranslateRequest(t *testing.T) {
	// Setup
	clientReq := client.ChatCompletionRequest{
		Model:       "gpt-4",
		MaxTokens:   100,
		Temperature: 0.8,
		Messages: []client.Message{
			{
				Content: "you are an assistant",
				Role:    client.System,
			},
			{
				Content: "How much is 2 + 2?",
				Role:    client.User,
			},
		},
		CustomParams: map[string]interface{}{
			"top_p":             float32(0.9),
			"stop":              []string{"\n"},
			"presence_penalty":  float32(0.6),
			"frequency_penalty": float32(0.5),
			"logit_bias":        map[string]int{"50256": -100},
			"user":              "user123",
		},
	}

	// Expected result
	expected := openai.ChatCompletionRequest{
		Model: "gpt-4",
		Messages: []openai.ChatCompletionMessage{
			{
				Content: "you are an assistant",
				Role:    openai.ChatMessageRoleSystem,
			},
			{
				Content: "How much is 2 + 2?",
				Role:    openai.ChatMessageRoleUser,
			},
		},
		MaxTokens:        100,
		Temperature:      0.8,
		TopP:             0.9,
		Stop:             []string{"\n"},
		PresencePenalty:  0.6,
		FrequencyPenalty: 0.5,
		LogitBias:        map[string]int{"50256": -100},
		User:             "user123",
	}

	// Run the function
	res, err := TranslateRequest(clientReq)

	// Validate results
	assert.Nil(t, err)
	assert.Equal(t, expected, res)
}
