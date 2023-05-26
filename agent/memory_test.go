package agent

import (
	"testing"

	openai "github.com/sashabaranov/go-openai"
)

// Test case where there are more relevant messages than the buffer size.
func TestBufferMemoryOverflow(t *testing.T) {
	messages := []openai.ChatCompletionMessage{
		{Role: "system", Content: "Hello, world!", Name: "system1"},
		{Role: "user", Content: "Hi, system!", Name: "user1"},
		{Role: "system", Content: "Hello again, world!", Name: "system2"},
		{Role: "assistant", Content: "Hi, user!", Name: "assistant1"},
	}

	memory := BufferMemory(2)
	cfg := Config{}

	if _, err := memory(cfg, messages); err == nil {
		t.Error("Expected error, got nil")
	}
}

// Test case where there are just enough messages to fit in the buffer.
func TestBufferMemoryExactFit(t *testing.T) {
	messages := []openai.ChatCompletionMessage{
		{Role: "system", Content: "Hello, world!", Name: "system1"},
		{Role: "user", Content: "Hi, system!", Name: "user1"},
	}

	memory := BufferMemory(2)
	cfg := Config{}

	bufferedMessages, err := memory(cfg, messages)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(bufferedMessages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(bufferedMessages))
	}

	// Check that the messages are correct and in the right order.
	if bufferedMessages[0].Role != "system" || bufferedMessages[0].Content != "Hello, world!" || bufferedMessages[0].Name != "system1" {
		t.Errorf("Unexpected first message: %v", bufferedMessages[0])
	}
	if bufferedMessages[1].Role != "user" || bufferedMessages[1].Content != "Hi, system!" || bufferedMessages[1].Name != "user1" {
		t.Errorf("Unexpected second message: %v", bufferedMessages[1])
	}
}

// Test case where there are system messages, user messages, and assistant messages
// in a random order.
// Test case where there are system messages, user messages, and assistant messages
// in a random order.
func TestBufferMemoryRandomOrder(t *testing.T) {
	messages := []openai.ChatCompletionMessage{
		{Role: "system", Content: "Hello, world!", Name: "system1"},
		{Role: "user", Content: "Hi, system!", Name: "user1"},
		{Role: "assistant", Content: "Hi, user!", Name: "assistant1"},
		{Role: "system", Content: "Hello again, world!", Name: "system2"},
		{Role: "assistant", Content: "How can I assist you today?", Name: "assistant2"},
	}

	memory := BufferMemory(4)
	cfg := Config{}

	bufferedMessages, err := memory(cfg, messages)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(bufferedMessages) != 4 {
		t.Errorf("Expected 4 messages, got %d", len(bufferedMessages))
	}

	// Check that the messages are correct and in the right order.
	// Expecting the messages "system1", "assistant1", "system2", "assistant2"
	expectedMessages := []openai.ChatCompletionMessage{messages[0], messages[2], messages[3], messages[4]}
	for i, msg := range expectedMessages {
		if bufferedMessages[i].Role != msg.Role || bufferedMessages[i].Content != msg.Content || bufferedMessages[i].Name != msg.Name {
			t.Errorf("Unexpected message at index %d: %v", i, bufferedMessages[i])
		}
	}
}

func TestTokenBufferMemory(t *testing.T) {
	messages := []openai.ChatCompletionMessage{
		{Role: "system", Content: "Hello, world!", Name: "system1"},
		{Role: "user", Content: "Hi, system!", Name: "user1"},
		{Role: "assistant", Content: "Hi, user!", Name: "assistant1"},
		{Role: "system", Content: "Hello again, world!", Name: "system2"},
		{Role: "assistant", Content: "How can I assist you today?", Name: "assistant2"},
	}

	fillRatio := 0.9

	memory := TokenBufferMemory(fillRatio)
	cfg := Config{Model: "gpt-4"}

	// We will set MaxTokens to drop the user1 message
	allTokens := 0
	for _, msg := range messages {
		tokenLen, err := tokenCount(cfg, msg)
		if err != nil {
			t.Errorf("Unexpected error in tokenCount: %v", err)
		}
		allTokens += tokenLen
	}

	allTokens = int(float64(allTokens) * fillRatio)

	droppedMessage := messages[1]
	droppedTokens, err := tokenCount(cfg, droppedMessage)
	if err != nil {
		t.Errorf("Unexpected error in tokenCount: %v", err)
	}

	cfg.MaxTokens = allTokens - droppedTokens + 1

	bufferedMessages, err := memory(cfg, messages)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Total token count in bufferedMessages should not exceed cfg.MaxTokens
	totalTokens := 0
	for _, msg := range bufferedMessages {
		tokenLen, err := tokenCount(cfg, msg)
		if err != nil {
			t.Errorf("Unexpected error in tokenCount: %v", err)
		}
		totalTokens += tokenLen
	}

	if totalTokens > cfg.MaxTokens {
		t.Errorf("Total tokens (%d) exceed MaxTokens (%d)", totalTokens, cfg.MaxTokens)
	}

	// Last message of input should be the last message of output
	if bufferedMessages[len(bufferedMessages)-1].Name != messages[len(messages)-1].Name {
		t.Errorf("Last message of output does not match last message of input")
	}

	// All system messages in input should be present in output
	for _, inputMsg := range messages {
		if inputMsg.Role == "system" {
			found := false
			for _, outputMsg := range bufferedMessages {
				if outputMsg.Name == inputMsg.Name {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("System message '%s' from input not found in output", inputMsg.Name)
			}
		}
	}

	// Messages in output should maintain their original order from input
	lastIndex := -1
	for _, inputMsg := range messages {
		for i, outputMsg := range bufferedMessages {
			if outputMsg.Name == inputMsg.Name {
				if i < lastIndex {
					t.Errorf("Message order has changed in output")
				}
				lastIndex = i
				break
			}
		}
	}
}
