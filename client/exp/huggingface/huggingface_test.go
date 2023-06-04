package huggingface

import (
	"errors"
	"testing"

	"github.com/ryszard/agency/client"
	"github.com/stretchr/testify/assert"
)

func TestTranslateResponse(t *testing.T) {
	for _, tc := range []struct {
		name     string
		convResp ConversationalResponse
		expected client.ChatCompletionResponse
		err      error
	}{
		{
			name: "basic",
			convResp: ConversationalResponse{
				GeneratedText: "Hello, I'm an AI.",
			},
			expected: client.ChatCompletionResponse{
				Choices: []client.Message{
					{
						Content: "Hello, I'm an AI.",
						Role:    "assistant",
					},
				},
			},
		},

		{
			name: "error",
			convResp: ConversationalResponse{
				Error: "error",
			},
			err:      errors.New("error"),
			expected: client.ChatCompletionResponse{},
		},
	} {
		result, err := TranslateResponse(tc.convResp)
		if tc.err != nil {
			if tc.err.Error() != err.Error() {
				t.Fatalf("Expected error %v, but got %v", tc.err, err)
			} else if err == nil {
				t.Fatal("Expected error, but got nil")
			}
		}

		if len(result.Choices) != len(tc.expected.Choices) {
			t.Errorf("Expected %v choice(s), but got %v", len(tc.expected.Choices), len(result.Choices))
		}

		for i, message := range result.Choices {
			if message.Content != tc.expected.Choices[i].Content || message.Role != tc.expected.Choices[i].Role {
				t.Errorf("Expected message %v to be %v, but got %v", i, tc.expected.Choices[i], message)
			}
		}
	}

}

func TestTranslateRequest(t *testing.T) {
	clientReq := client.ChatCompletionRequest{
		Model: "test-model",
		Messages: []client.Message{
			{Content: "Hello", Role: "system"},
			{Content: "How are you?", Role: "user"},
		},
		MaxTokens:   50,
		Temperature: 0.8,
		CustomParams: map[string]interface{}{
			"use_cache":          true,
			"wait_for_model":     false,
			"min_length":         10,
			"top_k":              50,
			"top_p":              0.9,
			"repetition_penalty": 1.2,
			"max_time":           60.0,
		},
	}

	hfReq, err := TranslateRequest(clientReq)
	assert.NoError(t, err)

	assert.Equal(t, clientReq.Model, hfReq.Model)
	assert.Equal(t, clientReq.MaxTokens, hfReq.Parameters.MaxLength) // Assuming MaxTokens corresponds to MaxLength
	assert.InDelta(t, clientReq.Temperature, hfReq.Parameters.Temperature, 0.01)

	assert.Equal(t, clientReq.CustomParams["use_cache"], hfReq.Options.UseCache)
	assert.Equal(t, clientReq.CustomParams["wait_for_model"], hfReq.Options.WaitForModel)
	assert.Equal(t, clientReq.CustomParams["min_length"], hfReq.Parameters.MinLength)
	assert.Equal(t, clientReq.CustomParams["top_k"], hfReq.Parameters.TopK)
	assert.Equal(t, clientReq.CustomParams["top_p"], hfReq.Parameters.TopP)
	assert.Equal(t, clientReq.CustomParams["repetition_penalty"], hfReq.Parameters.RepetitionPenalty)
	assert.Equal(t, clientReq.CustomParams["max_time"], hfReq.Parameters.MaxTime)
}

func TestTranslateRequestWithReservedCustomParams(t *testing.T) {
	clientReq := client.ChatCompletionRequest{
		Model:       "someModel",
		MaxTokens:   5,
		Temperature: 0.5,
		CustomParams: map[string]interface{}{
			"max_length":  5,
			"temperature": 0.5,
		},
	}

	_, err := TranslateRequest(clientReq)
	if err == nil {
		t.Fatalf("Expected error due to reserved keys in CustomParams, got nil")
	}
}

func TestTranslateRequestWithMessages(t *testing.T) {
	cases := []struct {
		name string
		req  client.ChatCompletionRequest
		want ConversationalRequest
	}{
		{
			name: "simple case",
			req: client.ChatCompletionRequest{
				Messages: []client.Message{
					{Content: "Hello", Role: client.User},
					{Content: "Hi. How are you?", Role: client.Assistant},
					{Content: "I'm fine, thanks.", Role: client.User},
				},
			},
			want: ConversationalRequest{
				Text:               "I'm fine, thanks.",
				PastUserInputs:     []string{"Hello"},
				GeneratedResponses: []string{"Hi. How are you?"},
			},
		},
		{
			name: "system and user messages",
			req: client.ChatCompletionRequest{
				Messages: []client.Message{
					{Content: "You are the assistant", Role: client.System},
					{Content: "Hello", Role: client.User},
					{Content: "Hi. How are you?", Role: client.Assistant},
					{Content: "I'm fine, thanks.", Role: client.User},
				},
			},
			want: ConversationalRequest{
				Text:               "I'm fine, thanks.",
				PastUserInputs:     []string{"You are the assistant\n\nHello"},
				GeneratedResponses: []string{"Hi. How are you?"},
			},
		},
	}

	for _, c := range cases {
		got, err := TranslateRequest(c.req)
		assert.NoError(t, err, c.name)
		assert.Equal(t, c.want.Text, got.Text, c.name)
		assert.Equal(t, c.want.PastUserInputs, got.PastUserInputs, "%s: PastUserInputs", c.name)
		assert.Equal(t, c.want.GeneratedResponses, got.GeneratedResponses, "%s: GeneratedResponses", c.name)
	}

}
