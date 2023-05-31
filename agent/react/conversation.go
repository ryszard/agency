package react

import (
	"fmt"
	"strings"
)

// Tag represents the type of an entry in the conversation.
type Tag string

var Tags = struct {
	Thought      Tag
	Action       Tag
	Observation  Tag
	Question     Tag
	Assumption   Tag
	Answer       Tag
	FinalAnswer  Tag
	Unrecognized Tag
}{
	Thought:      "Thought",
	Action:       "Action",
	Observation:  "Observation",
	Question:     "Question",
	Assumption:   "Assumption",
	Answer:       "Answer",
	FinalAnswer:  "Final Answer",
	Unrecognized: "",
}

func (et Tag) Prefix() string {
	return fmt.Sprintf("%s: ", et)
}

func (et Tag) IsRecognized() bool {
	return et != Tags.Unrecognized
}

func matchTag(line string) Tag {
	switch {
	case strings.HasPrefix(line, Tags.Thought.Prefix()):
		return Tags.Thought
	case strings.HasPrefix(line, Tags.Action.Prefix()):
		return Tags.Action
	case strings.HasPrefix(line, Tags.Assumption.Prefix()):
		return Tags.Assumption
	case strings.HasPrefix(line, Tags.Observation.Prefix()):
		return Tags.Observation
	case strings.HasPrefix(line, Tags.Question.Prefix()):
		return Tags.Question
	case strings.HasPrefix(line, Tags.Answer.Prefix()):
		return Tags.Answer
	case strings.HasPrefix(line, Tags.FinalAnswer.Prefix()):
		return Tags.FinalAnswer
	default:
		return Tags.Unrecognized
	}
}

// Entry represents an individual step in the conversation or process log, which includes
// a thought, action, action input, observation or final answer.
type Entry struct {
	Tag      Tag
	Content  string
	Argument string
}

func (entry Entry) String() string {
	if entry.Tag == Tags.Action {
		return fmt.Sprintf("%s: %s\n%s", entry.Tag, entry.Argument, entry.Content)
	}
	return fmt.Sprintf("%s: %s", entry.Tag, entry.Content)
}

// Parse parses a conversation from a string.
func Parse(text string) (entries []Entry, err error) {
	// Split the text into lines
	lines := strings.Split(text, "\n")

	currentEntry := Entry{}

	for _, line := range lines {
		tag := matchTag(line)

		if tag.IsRecognized() {
			// We found the beginning of a new step.
			if currentEntry.Tag.IsRecognized() {
				// There was a previous entry in progress, so we should finalize
				// it and add it to the list of entries.
				entries = append(entries, currentEntry)
			}
			currentEntry = Entry{Tag: tag}
			if tag != Tags.Action {
				currentEntry.Content = strings.TrimSpace(strings.TrimPrefix(line, tag.Prefix()))
			} else {
				// Split the line into the entry type and the argument the first
				// value returned by strings.Cut is going to be the entry tag.
				// `found` is always going to be true, as otherwise the entry
				// tag would have been Unrecognized
				_, arg, _ := strings.Cut(line, ": ")

				currentEntry.Argument = strings.TrimSpace(arg)
			}

		} else if currentEntry.Tag.IsRecognized() {
			// We are in the middle of a entry; add the line to its content.
			currentEntry.Content += "\n" + line
		} else {
			// No new entry, and no entry in progress. Unless this is a blank
			// we should return an error.
			if strings.TrimSpace(line) != "" {
				return nil, fmt.Errorf("unrecognized tag: %q", line)
			}
		}
	}

	entries = append(entries, currentEntry)

	for i, entry := range entries {

		entries[i].Content = strings.TrimSpace(entry.Content)

	}

	return entries, nil
}
