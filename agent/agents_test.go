package agent

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestActorSystem(t *testing.T) {
	t.Run("System message is added to the actor's messages", func(t *testing.T) {
		actor := &Actor{}
		systemMessage := "Test system message"

		actor.System(systemMessage)

		want := []openai.ChatCompletionMessage{
			{
				Content: systemMessage,
				Role:    "system",
			},
		}

		if !reflect.DeepEqual(actor.Messages, want) {
			t.Errorf("got %v, want %v", actor.Messages, want)
		}
	})
}

type MockClient struct {
	mock.Mock
}

func (m *MockClient) CreateChatCompletion(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(openai.ChatCompletionResponse), args.Error(1)
}

func TestRespond(t *testing.T) {
	ctx := context.Background()
	messages := []openai.ChatCompletionMessage{
		{Content: "Message 1", Role: "user"},
		{Content: "Message 2", Role: "assistant"},
	}

	expectedResp := openai.ChatCompletionResponse{
		Choices: []openai.ChatCompletionChoice{
			{Message: openai.ChatCompletionMessage{Content: "Response", Role: "assistant"}},
		},
	}

	mockClient := &MockClient{}
	mockCall := mockClient.On("CreateChatCompletion", ctx, mock.Anything).Return(expectedResp, nil)

	ac := &Actor{
		name:     "Test",
		client:   mockClient,
		Messages: messages,
	}
	// Test success case
	respMsg, err := ac.Respond(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "Response", respMsg)
	assert.Len(t, ac.Messages, 3)

	expectedReq := openai.ChatCompletionRequest{
		Model:     openai.GPT3Dot5Turbo,
		Messages:  messages,
		MaxTokens: 300,
	}
	mockClient.AssertCalled(t, "CreateChatCompletion", ctx, expectedReq)

	// Test error case
	mockCall.Unset()
	expectedErr := errors.New("API error")
	mockClient.On("CreateChatCompletion", ctx, mock.Anything).Return(openai.ChatCompletionResponse{}, expectedErr)

	_, err = ac.Respond(ctx)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
}
