package agent

import (
	"errors"
	"fmt"

	"github.com/sashabaranov/go-openai"
	log "github.com/sirupsen/logrus"
	"github.com/tiktoken-go/tokenizer"
)

// Memory serves as the agent's memory. It receives a config and the agent's
// list of messages, and returns a new list of messages, that can be changed to
// provide the agent's memories. These functions are most likely to be called in
// Agent.Respond, right before sending an API request.
//
// A memory function should not drop any system messages, and should not drop
// the last user message. If this is impossible, it should return an error.
type Memory func(Config, []openai.ChatCompletionMessage) ([]openai.ChatCompletionMessage, error)

// BufferMemory is a memory that keeps the last n messages. All the system
// messages will be kept. If the buffer size is too small, it will return an
// error.
func BufferMemory(n int) Memory {
	return func(cfg Config, messages []openai.ChatCompletionMessage) ([]openai.ChatCompletionMessage, error) {

		// if there's less or equal to n messages, return all the messages
		if len(messages) <= n {
			return messages, nil
		}

		// count the number of system messages
		systemMessages := 0
		for _, message := range messages {
			if message.Role == "system" {
				systemMessages++
			}
		}

		nonSystemMessages := n - systemMessages

		if (messages[len(messages)-1].Role == "system" && nonSystemMessages < 0) || nonSystemMessages < 1 {
			return nil, errors.New("buffer size is too small")
		}

		// initialize the new messages slice
		newMessages := make([]openai.ChatCompletionMessage, 0, n)

		// iterate over messages from the end. If you encounter a system
		// message, add it to the new messages slice, and decrease
		// systemMessages by 1. If you encounter a non-system message, add it to
		// the new messages slice, and decrease nonSystemMessages by 1. If
		// nonSystemMessages is 0, drop any user messages.
		for i := len(messages) - 1; i >= 0; i-- {
			message := messages[i]

			if message.Role == "system" {
				newMessages = append(newMessages, message)
				systemMessages--
			} else if nonSystemMessages > 0 {
				newMessages = append(newMessages, message)
				nonSystemMessages--
			} else {
				log.WithField("message", message).Debug("Dropping user message")
			}

			if nonSystemMessages+systemMessages == 0 {
				break
			}
		}

		// reverse the new messages slice
		for i := len(newMessages)/2 - 1; i >= 0; i-- {
			opp := len(newMessages) - 1 - i
			newMessages[i], newMessages[opp] = newMessages[opp], newMessages[i]
		}

		return newMessages, nil

	}
}

// tokenCount returns the number of tokens in a message.
func tokenCount(cfg Config, message openai.ChatCompletionMessage) (int, error) {
	codec, err := tokenizer.ForModel(tokenizer.Model(cfg.Model))
	if err != nil {
		return 0, err
	}

	ids, _, err := codec.Encode(message.Content)
	if err != nil {
		return 0, err
	}

	return len(ids), nil
}

func partitionByTokenLimit(cfg Config, messages []openai.ChatCompletionMessage, maxTokens int) ([]openai.ChatCompletionMessage, []openai.ChatCompletionMessage, error) {
	// get the number of tokens in each messages
	tokens := make([]int, len(messages))
	for i, message := range messages {
		count, err := tokenCount(cfg, message)
		if err != nil {
			return nil, nil, err
		}
		tokens[i] = count
	}

	// count the number of tokens in the system messages
	systemTokens := 0
	for i, message := range messages {
		if message.Role == "system" {
			systemTokens += tokens[i]
		}
	}

	if systemTokens > maxTokens {
		return nil, nil, fmt.Errorf("system messages are too long (%d tokens), MaxTokens * fillRatio is %d", systemTokens, cfg.MaxTokens)
	}

	nonSystemTokens := maxTokens - systemTokens

	// if the last message is non-system, and it's too long, return an error
	if messages[len(messages)-1].Role != "system" && tokens[len(messages)-1] > nonSystemTokens {
		return nil, nil, fmt.Errorf("last message and system messages are too long (%d tokens), MaxTokens * fillRatio is %d", tokens[len(messages)-1]+systemTokens, maxTokens)
	}

	// initialize the new messages slice
	newMessages := make([]openai.ChatCompletionMessage, 0, len(messages))
	droppedMessages := make([]openai.ChatCompletionMessage, 0)

	// iterate over messages from the end.
	for i := len(messages) - 1; i >= 0; i-- {
		message := messages[i]
		tokenCount := tokens[i]
		if message.Role == "system" {
			newMessages = append(newMessages, message)
		} else if nonSystemTokens >= tokenCount {
			newMessages = append(newMessages, message)
			nonSystemTokens -= tokenCount
		} else {
			log.WithField("message", message).Debug("Dropping user message")
			droppedMessages = append(droppedMessages, message)
		}
	}

	// reverse the new messages slice
	for i := len(newMessages)/2 - 1; i >= 0; i-- {
		opp := len(newMessages) - 1 - i
		newMessages[i], newMessages[opp] = newMessages[opp], newMessages[i]
	}

	// reverse the dropped messages slice
	for i := len(droppedMessages)/2 - 1; i >= 0; i-- {
		opp := len(droppedMessages) - 1 - i
		droppedMessages[i], droppedMessages[opp] = droppedMessages[opp], droppedMessages[i]
	}

	return newMessages, droppedMessages, nil
}

// TokenBufferMemory will keep messages that contain at most MaxTokens *
// fillRatio tokens (taken form the config). It will keep all the system
// messages, and the last message. If this is impossible, it will return an
// error. If fillRatio is not in the range (0, 1], this function will panic.
func TokenBufferMemory(fillRatio float64) Memory {
	if fillRatio <= 0 || fillRatio > 1 {
		panic("fillRatio must be in the range (0, 1]")
	}

	return func(cfg Config, messages []openai.ChatCompletionMessage) ([]openai.ChatCompletionMessage, error) {
		maxTokens := int(float64(cfg.MaxTokens) * fillRatio)

		newMessages, droppedMessages, err := partitionByTokenLimit(cfg, messages, maxTokens)

		if err != nil {
			return nil, err
		}

		log.WithField("messages", droppedMessages).Debug("Dropped messages: ", droppedMessages)

		return newMessages, nil

	}
}
