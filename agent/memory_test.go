package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/ryszard/agency/client"
)

// Test case where there are more relevant messages than the buffer size.
func TestBufferMemoryOverflow(t *testing.T) {
	messages := []client.Message{
		{Role: "system", Content: "Hello, world!"},
		{Role: "user", Content: "Hi, system!"},
		{Role: "system", Content: "Hello again, world!"},
		{Role: "assistant", Content: "Hi, user!"},
	}

	memory := BufferMemory(2)
	cfg := Config{}

	if _, err := memory(context.TODO(), cfg, messages); err == nil {
		t.Error("Expected error, got nil")
	}
}

// Test case where there are just enough messages to fit in the buffer.
func TestBufferMemoryExactFit(t *testing.T) {
	messages := []client.Message{
		{Role: "system", Content: "Hello, world!"},
		{Role: "user", Content: "Hi, system!"},
	}

	memory := BufferMemory(2)
	cfg := Config{}

	bufferedMessages, err := memory(context.TODO(), cfg, messages)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(bufferedMessages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(bufferedMessages))
	}

	// Check that the messages are correct and in the right order.
	if bufferedMessages[0].Role != "system" || bufferedMessages[0].Content != "Hello, world!" {
		t.Errorf("Unexpected first message: %v", bufferedMessages[0])
	}
	if bufferedMessages[1].Role != "user" || bufferedMessages[1].Content != "Hi, system!" {
		t.Errorf("Unexpected second message: %v", bufferedMessages[1])
	}
}

// Test case where there are system messages, user messages, and assistant messages
// in a random order.
// Test case where there are system messages, user messages, and assistant messages
// in a random order.
func TestBufferMemoryRandomOrder(t *testing.T) {
	messages := []client.Message{
		{Role: "system", Content: "Hello, world!"},
		{Role: "user", Content: "Hi, system!"},
		{Role: "assistant", Content: "Hi, user!"},
		{Role: "system", Content: "Hello again, world!"},
		{Role: "assistant", Content: "How can I assist you today?"},
	}

	memory := BufferMemory(4)
	cfg := Config{}

	bufferedMessages, err := memory(context.TODO(), cfg, messages)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(bufferedMessages) != 4 {
		t.Errorf("Expected 4 messages, got %d", len(bufferedMessages))
	}

	// Check that the messages are correct and in the right order.
	// Expecting the messages "system1", "assistant1", "system2", "assistant2"
	expectedMessages := []client.Message{messages[0], messages[2], messages[3], messages[4]}
	for i, msg := range expectedMessages {
		if bufferedMessages[i].Role != msg.Role || bufferedMessages[i].Content != msg.Content {
			t.Errorf("Unexpected message at index %d: %v", i, bufferedMessages[i])
		}
	}
}

func TestPartitionByTokenLimit(t *testing.T) {
	messages := []client.Message{
		{Role: "system", Content: " "},
		{Role: "user", Content: " "},
		{Role: "assistant", Content: " "},
		{Role: "user", Content: " "},
		{Role: "assistant", Content: " "},
		{Role: "user", Content: " "},
		{Role: "assistant", Content: " "},
	}

	tokenCount := func(s string) (int, error) {
		return len(s), nil
	}

	retainedMessages, droppedMessages, _ := partitionByTokenLimit(Config{}, messages, 4, tokenCount)

	if len(retainedMessages) != 4 {
		t.Errorf("Expected 4 retained messages, got %d", len(retainedMessages))
		t.Errorf("Retained messages: %v", retainedMessages)
	}

	if len(droppedMessages) != 3 {
		t.Errorf("Expected 3 dropped messages, got %d", len(droppedMessages))
		t.Errorf("dropped messages: %v", droppedMessages)

	}

	// Check that the messages are correct and in the right order.
	// Expecting the messages "0", "4", "5","6"
	expectedMessages := []client.Message{messages[0], messages[4], messages[5], messages[6]}
	for i, msg := range expectedMessages {
		if retainedMessages[i].Role != msg.Role || retainedMessages[i].Content != msg.Content {
			t.Errorf("Unexpected message at index %d: %v", i, retainedMessages[i])
		}
	}

}

func TestPartitionByTokenLimitLongerMessages(t *testing.T) {
	messages := []client.Message{
		{Role: "system", Content: " "},
		{Role: "user", Content: " "},
		{Role: "assistant", Content: " "},
		{Role: "user", Content: " "},
		{Role: "assistant", Content: " "},
		{Role: "user", Content: "55555555555555"},
		{Role: "assistant", Content: " "},
	}

	tokenCount := func(s string) (int, error) {
		return len(s), nil
	}

	retainedMessages, _, _ := partitionByTokenLimit(Config{}, messages, 4, tokenCount)

	// Check that the messages are correct and in the right order.
	// Expecting the messages "0","6" - 5 is too long to fit in the buffer.
	expectedMessages := []client.Message{messages[0], messages[6]}
	for i, msg := range expectedMessages {
		if retainedMessages[i].Role != msg.Role || retainedMessages[i].Content != msg.Content {
			t.Errorf("Unexpected message at index %d: %v", i, retainedMessages[i])
		}
	}

}

func TestTokenBufferMemory(t *testing.T) {
	messages := []client.Message{
		{Role: client.System, Content: "Hello, world!"},
		{Role: client.User, Content: "Hi, system!"},
		{Role: client.Assistant, Content: "Hi, user!"},
		{Role: client.System, Content: "Hello again, world!"},
		{Role: client.Assistant, Content: "How can I assist you today?"},
	}

	cfg := Config{RequestTemplate: client.ChatCompletionRequest{Model: "gpt-4"}}

	tokenCount := func(s string) (int, error) {
		// split the s into words and count the number of words
		return len(strings.Split(s, " ")), nil
	}

	// We will set MaxTokens to drop the user1 message
	allTokens := 0
	for _, msg := range messages {
		tokenLen, err := tokenCount(msg.Content)
		if err != nil {
			t.Errorf("Unexpected error in tokenCount: %v", err)
		}
		allTokens += tokenLen
	}

	droppedMessage := messages[1]
	droppedTokens, err := tokenCount(droppedMessage.Content)
	if err != nil {
		t.Errorf("Unexpected error in tokenCount: %v", err)
	}

	memoryLimit := allTokens - droppedTokens + 1
	memory := TokenBufferMemory(memoryLimit, tokenCount)

	bufferedMessages, err := memory(context.TODO(), cfg, messages)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Total token count in bufferedMessages should not exceed cfg.MaxTokens
	totalTokens := 0
	for _, msg := range bufferedMessages {
		tokenLen, err := tokenCount(msg.Content)
		if err != nil {
			t.Errorf("Unexpected error in tokenCount: %v", err)
		}
		totalTokens += tokenLen
	}

	if totalTokens > memoryLimit {
		t.Errorf("Total tokens (%d) exceed MaxTokens (%d)", totalTokens, cfg.RequestTemplate.MaxTokens)
	}

	// Last message of input should be the last message of output
	if bufferedMessages[len(bufferedMessages)-1].Content != messages[len(messages)-1].Content {
		t.Errorf("Last message of output does not match last message of input")
	}

	// All system messages in input should be present in output
	for _, inputMsg := range messages {
		if inputMsg.Role == "system" {
			found := false
			for _, outputMsg := range bufferedMessages {
				if outputMsg.Content == inputMsg.Content {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("System message '%s' from input not found in output", inputMsg.Content)
			}
		}
	}

	// Messages in output should maintain their original order from input
	lastIndex := -1
	for _, inputMsg := range messages {
		for i, outputMsg := range bufferedMessages {
			if outputMsg.Content == inputMsg.Content {
				if i < lastIndex {
					t.Errorf("Message order has changed in output")
				}
				lastIndex = i
				break
			}
		}
	}
}
