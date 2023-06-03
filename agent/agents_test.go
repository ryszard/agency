package agent

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/ryszard/agency/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAgentSystem(t *testing.T) {
	t.Run("System message is added to the actor's messages", func(t *testing.T) {
		ag := &BaseAgent{}
		systemMessage := "Test system message"

		ag.System(systemMessage)

		want := []client.Message{
			{
				Content: systemMessage,
				Role:    "system",
			},
		}

		if !reflect.DeepEqual(ag.Messages(), want) {
			t.Errorf("got %v, want %v", ag.Messages(), want)
		}
	})
}

type MockClient struct {
	mock.Mock
}

func (m *MockClient) CreateChatCompletion(ctx context.Context, req client.ChatCompletionRequest) (client.ChatCompletionResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(client.ChatCompletionResponse), args.Error(1)
}

func TestRespond(t *testing.T) {
	ctx := context.Background()
	messages := []client.Message{
		{Content: "Message 1", Role: "user"},
		{Content: "Message 2", Role: "assistant"},
	}

	expectedResp := client.ChatCompletionResponse{
		Choices: []client.Message{
			{Content: "Response", Role: "assistant"},
		},
	}

	mockClient := &MockClient{}
	mockCall := mockClient.On("CreateChatCompletion", ctx, mock.Anything).Return(expectedResp, nil)

	ac := NewBaseAgent("Test", WithClient(mockClient))
	ac.messages = messages

	// Test success case
	respMsg, err := ac.Respond(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "Response", respMsg)
	assert.Len(t, ac.Messages(), 3)

	expectedReq := client.ChatCompletionRequest{
		Messages: messages,
	}
	mockClient.AssertCalled(t, "CreateChatCompletion", ctx, expectedReq)

	// Test error case
	mockCall.Unset()
	expectedErr := errors.New("API error")
	mockClient.On("CreateChatCompletion", ctx, mock.Anything).Return(client.ChatCompletionResponse{}, expectedErr)

	_, err = ac.Respond(ctx)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
}
