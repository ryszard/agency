package agent

import (
	"context"
	"testing"

	"github.com/ryszard/agency/util/cache"
	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHash(t *testing.T) {
	ag := &cachedAgent{
		Agent: New("test"),
		cache: nil,
	}

	// Test when messages and config are default
	hash1, err := ag.hash()
	assert.NoError(t, err)

	// Test when messages are the same, but config is different
	ag.Append(openai.ChatCompletionMessage{Role: "user", Content: "hello"})
	hash2, err := ag.hash(WithMaxTokens(10)) // Assuming WithMaxTokens is an Option to change max tokens
	assert.NoError(t, err)
	assert.NotEqual(t, hash1, hash2)

	// Test when messages are the same and config is the same, hash should be the same
	hash3, err := ag.hash(WithMaxTokens(10))
	assert.NoError(t, err)
	assert.Equal(t, hash2, hash3)

	// Test when messages are different, hash should be different
	ag.Append(openai.ChatCompletionMessage{Role: "user", Content: "goodbye"})
	hash4, err := ag.hash(WithMaxTokens(10))
	assert.NoError(t, err)
	assert.NotEqual(t, hash2, hash4)
	assert.NotEqual(t, hash1, hash4)
}

type mockClient struct {
	mock.Mock
}

func (m *mockClient) CreateChatCompletion(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(openai.ChatCompletionResponse), args.Error(1)
}

func (m *mockClient) CreateChatCompletionStream(ctx context.Context, req openai.ChatCompletionRequest) (*openai.ChatCompletionStream, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*openai.ChatCompletionStream), args.Error(1)
}

func TestCachedAgent(t *testing.T) {
	ctx := context.Background()
	client := new(mockClient)

	var (
		userMessage       = openai.ChatCompletionMessage{Role: "user", Content: "hello, assistant"}
		assistantResponse = openai.ChatCompletionMessage{Role: "assistant", Content: "hello, user"}
	)

	createAgent := func() Agent {
		ag := WithCache(New("test", WithClient(client)), cache.Memory())

		ag.Append(userMessage)

		return ag
	}

	ag := createAgent()

	client.On(
		"CreateChatCompletion",
		mock.Anything,
		openai.ChatCompletionRequest{
			Messages: []openai.ChatCompletionMessage{userMessage},
		},
	).Return(openai.ChatCompletionResponse{
		Choices: []openai.ChatCompletionChoice{
			{
				Message: assistantResponse,
			},
		},
	}, nil)

	client.On(
		"CreateChatCompletion",
		mock.Anything,
		openai.ChatCompletionRequest{
			Messages: []openai.ChatCompletionMessage{userMessage},
		},
	).Return(openai.ChatCompletionResponse{
		Choices: []openai.ChatCompletionChoice{
			{
				Message: openai.ChatCompletionMessage{
					Role:    "assistant",
					Content: "hello second time, too bad the test is failing",
				},
			},
		},
	}, nil)

	// The client should be called the first time
	respMsg, err := ag.Respond(ctx)
	assert.NoError(t, err)
	assert.Equal(t, assistantResponse.Content, respMsg)

	ag = createAgent()

	// The client should not be called the second time because the messages and options are the same
	respMsg, err = ag.Respond(ctx)
	assert.NoError(t, err)
	assert.Equal(t, assistantResponse.Content, respMsg)

	client.AssertExpectations(t)

	// If messages or options are different, the client should be called
	anotherMessage := openai.ChatCompletionMessage{Role: "user", Content: "goodbye, assistant"}
	ag.Append(anotherMessage)

	response := openai.ChatCompletionResponse{
		Choices: []openai.ChatCompletionChoice{
			{Message: openai.ChatCompletionMessage{
				Role:    "assistant",
				Content: "goodbye, user"}}}}

	client.On("CreateChatCompletion", ctx, openai.ChatCompletionRequest{Messages: []openai.ChatCompletionMessage{userMessage, assistantResponse, anotherMessage}}).Return(response, nil).Once()

	respMsg, err = ag.Respond(ctx)
	assert.NoError(t, err)
	assert.Equal(t, response.Choices[0].Message.Content, respMsg)

	client.AssertExpectations(t)
}
