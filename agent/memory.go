package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"text/template"

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
type Memory func(context.Context, Config, []openai.ChatCompletionMessage) ([]openai.ChatCompletionMessage, error)

// BufferMemory is a memory that keeps the last n messages. All the system
// messages will be kept. If the buffer size is too small, it will return an
// error.
func BufferMemory(n int) Memory {
	return func(ctx context.Context, cfg Config, messages []openai.ChatCompletionMessage) ([]openai.ChatCompletionMessage, error) {

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

type tokenCounter func(cfg Config, message openai.ChatCompletionMessage) (int, error)

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

func partitionByTokenLimit(
	cfg Config,
	messages []openai.ChatCompletionMessage,
	maxTokens int,
	tokenCount tokenCounter,
) ([]openai.ChatCompletionMessage, []openai.ChatCompletionMessage, error) {
	// get the number of tokens in each messages
	tokens := make([]int, len(messages))
	total := 0
	for i, message := range messages {
		count, err := tokenCount(cfg, message)
		if err != nil {
			return nil, nil, err
		}
		tokens[i] = count
		total += count
	}
	log.WithField("total", total).Debug("Token count")

	// count the number of tokens in the system messages
	systemTokens := 0
	for i, message := range messages {
		if message.Role == "system" {
			systemTokens += tokens[i]
		}
	}

	if systemTokens > maxTokens {
		return nil, nil, fmt.Errorf("system messages are too long (%d tokens), MaxTokens * fillRatio is %d", systemTokens, maxTokens)
	}

	nonSystemTokens := maxTokens - systemTokens

	// if the last message is non-system, and it's too long, return an error
	if messages[len(messages)-1].Role != "system" && tokens[len(messages)-1] > nonSystemTokens {
		return nil, nil, fmt.Errorf("last message and system messages are too long (%d tokens), MaxTokens * fillRatio is %d", tokens[len(messages)-1]+systemTokens, maxTokens)
	}

	// initialize the new messages slice
	newMessages := make([]openai.ChatCompletionMessage, 0, len(messages))
	droppedMessages := make([]openai.ChatCompletionMessage, 0)

	startedDropping := false

	// iterate over messages from the end.
	for i := len(messages) - 1; i >= 0; i-- {
		log.WithField("message", messages[i]).Trace("Checking message")

		message := messages[i]
		tokenCount := tokens[i]

		if message.Role == openai.ChatMessageRoleSystem {
			log.WithField("message", messages[i]).
				WithField("nonSystemTokens", nonSystemTokens).
				WithField("tokenCount", tokenCount).
				WithField("startedDropping", startedDropping).
				Trace("Retaining (System)")

			// Never drop a system message
			newMessages = append(newMessages, message)
		} else if !startedDropping && nonSystemTokens >= tokenCount {
			log.WithField("message", messages[i]).
				WithField("nonSystemTokens", nonSystemTokens).
				WithField("tokenCount", tokenCount).
				WithField("startedDropping", startedDropping).
				Trace("Retaining")

			newMessages = append(newMessages, message)
			nonSystemTokens -= tokenCount

		} else {
			log.
				WithField("message", message).
				WithField("nonSystemTokens", nonSystemTokens).
				WithField("tokenCount", tokenCount).
				WithField("startedDropping", startedDropping).
				Trace("Dropping")
			startedDropping = true
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

	return func(ctx context.Context, cfg Config, messages []openai.ChatCompletionMessage) ([]openai.ChatCompletionMessage, error) {
		maxTokens := int(float64(cfg.MaxTokens) * fillRatio)

		newMessages, droppedMessages, err := partitionByTokenLimit(cfg, messages, maxTokens, tokenCount)
		if err != nil {
			return nil, err
		}

		log.WithField("messages", droppedMessages).Debug("Dropped messages: ", droppedMessages)

		return newMessages, nil

	}
}

type SummarizerTemplateValues struct {
	Messages        []openai.ChatCompletionMessage
	PreviousSummary string
}

var summaryMessageTemplate = template.Must(template.New("summaryMessage").Parse(`
You are the assistant. Part of this conversation has been truncated. Here is the summary of the conversation so far:

SUMMARY:
"{{js .}}"
END SUMMARY
`))

// parseSummary looks for the string "SUMMARY:" in the provided message, takes
// all the lines after it up to "END SUMMARY", parses them as a JSON string, and
// returns the resulting object. If there's no summary, return the empty string.
func parseSummary(message string) (string, error) {
	// find the start of the summary
	start := strings.Index(message, "SUMMARY:")
	if start == -1 {
		return "", nil
	}

	// find the end of the summary
	end := strings.Index(message, "END SUMMARY")
	if end == -1 {
		return "", fmt.Errorf("no END SUMMARY found")
	}

	// get the summary string
	summary := message[start+len("SUMMARY:") : end]

	// decode the summary from JSON
	var summaryObj string
	err := json.Unmarshal([]byte(summary), &summaryObj)
	if err != nil {
		log.WithError(err).WithField("summary", summary).Error("Error parsing summary")
		return "", err
	}

	return summaryObj, nil
}

// Summarizer memory is a memory implementation that when the number of tokens
// approaches MaxTokens * fillRatio, it will summarize the messages that it is
// dropping. It will never drop system messages.
func SummarizerMemoryWithTemplate(fillRatio float64, tmpl *template.Template, options ...Option) Memory {
	if fillRatio <= 0 || fillRatio > 1 {
		panic("fillRatio must be in the range (0, 1]")
	}

	return func(ctx context.Context, cfg Config, messages []openai.ChatCompletionMessage) ([]openai.ChatCompletionMessage, error) {
		maxTokens := int(float64(cfg.MaxTokens) * fillRatio)

		retainedMessages, droppedMessages, err := partitionByTokenLimit(cfg, messages, maxTokens, tokenCount)
		if err != nil {
			return nil, err
		}

		if len(droppedMessages) == 0 {
			log.Debug("No messages dropped")
			return messages, nil
		}

		log.WithField("messages", droppedMessages).Trace("Dropped messages")
		log.WithField("messages", retainedMessages).Trace("Retained messages")

		summarizerOptions := []Option{
			WithConfig(cfg),
		}
		summarizerOptions = append(summarizerOptions, options...)
		summarizerOptions = append(summarizerOptions, []Option{WithMemory(nil), WithoutStreaming()}...)

		// FIXME: allow the caller to override this.
		summarizerOptions = append(summarizerOptions, WithMaxTokens(1000))

		// Find if there is a previous summary. If there is, it's going to be in
		// the first message in retainedMessages, which is going to be a system message.
		var previousSummary string
		firstMessage := retainedMessages[0]
		if firstMessage.Role == openai.ChatMessageRoleSystem {
			previousSummary, err = parseSummary(firstMessage.Content)
			if err != nil {
				return nil, err
			}
			// drop the first message, we'll write a better one
			retainedMessages = retainedMessages[1:]
		}

		summarizer := New("summarizer", summarizerOptions...)

		sb := strings.Builder{}

		if err := tmpl.Execute(&sb, SummarizerTemplateValues{
			Messages:        droppedMessages,
			PreviousSummary: previousSummary,
		}); err != nil {
			return nil, err
		}

		_, err = summarizer.Listen(sb.String())
		if err != nil {
			return nil, err
		}

		summary, err := summarizer.Respond(ctx)
		if err != nil {
			return nil, err
		}

		wr := &strings.Builder{}
		if err := summaryMessageTemplate.Execute(wr, summary); err != nil {
			return nil, err
		}

		log.WithField("messages", droppedMessages).Debug("Dropped messages.")
		log.WithField("summary", wr.String()).Debug("Summary message.")

		newMessages := make([]openai.ChatCompletionMessage, 0, len(retainedMessages)+1)
		newMessages = append(newMessages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: wr.String(),
		})

		newMessages = append(newMessages, retainedMessages...)

		return newMessages, nil

	}
}

const summarizerTemplate = `
As the assistant, your role is to maintain an ongoing, concise summary of the entire conversation so far. This includes incorporating key actions, requests, and responses from both the previous summary and new conversation lines. 

Remember to highlight significant user actions, especially any change in user's requests, instructions, or themes. These changes are as important as the content of the conversation.

Here's the information you'll need to update the summary:

PREVIOUS SUMMARY: "{{js .PreviousSummary}}"
END PREVIOUS SUMMARY

NEW LINES:
{{range .Messages}}
{{.Role}}: "{{js .Content}}"
{{end}}
END NEW LINES

You're not just adding to the previous summary, but integrating the new information into it, particularly noting any changes or additions to the user's requests or instructions. Your aim is to create a concise, evolving record of the conversation that tracks user's requests and the assistant's responses, while incorporating all relevant information.
`

func SummarizerMemory(fillRatio float64, options ...Option) Memory {
	return SummarizerMemoryWithTemplate(fillRatio, template.Must(template.New("summarizer").Parse(summarizerTemplate)), options...)
}
