package react

import (
	"fmt"
	"strings"
)

// Tag represents the type of step in the conversation.
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
func Parse(text string) (steps []Entry, err error) {
	// Split the text into lines
	lines := strings.Split(text, "\n")

	currentStep := Entry{}

	for _, line := range lines {
		stepType := matchTag(line)

		if stepType.IsRecognized() {
			// We found the beginning of a new step.
			if currentStep.Tag.IsRecognized() {
				// There was a previous step in progress, so we should finalize
				// it and add it to the list of steps.
				steps = append(steps, currentStep)
			}
			currentStep = Entry{Tag: stepType}
			if stepType != Tags.Action {
				currentStep.Content = strings.TrimSpace(strings.TrimPrefix(line, stepType.Prefix()))
			} else {
				// Split the line into the step type and the argument the first
				// value returned by strings.Cut is going to be the step type.
				// `found` is always going to be true, as otherwise the step
				// type would have been Unrecognized
				_, arg, _ := strings.Cut(line, ": ")

				currentStep.Argument = strings.TrimSpace(arg)
			}

		} else if currentStep.Tag.IsRecognized() {
			// We are in the middle of a step; add the line to its content.
			currentStep.Content += "\n" + line
		} else {
			// No new step, and no step in progress. Unless this is a blank
			// we should return an error.
			if strings.TrimSpace(line) != "" {
				return nil, fmt.Errorf("unrecognized step: %q", line)
			}
		}
	}

	steps = append(steps, currentStep)

	for i, step := range steps {

		steps[i].Content = strings.TrimSpace(step.Content)

	}

	return steps, nil
}
