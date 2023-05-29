package react

import (
	"fmt"
	"strings"
)

// StepType represents the type of step in the conversation.
type StepType string

const (
	ThoughtStep     StepType = "Thought"
	ActionStep      StepType = "Action"
	ObservationStep StepType = "Observation"
	QuestionStep    StepType = "Question"
	AssumptionStep  StepType = "Assumption"
	AnswerStep      StepType = "Answer"
	FinalAnswerStep StepType = "Final Answer"
	Unrecognized    StepType = ""
)

func (s StepType) Prefix() string {
	return fmt.Sprintf("%s: ", s)
}

func (s StepType) IsRecognized() bool {
	return s != Unrecognized
}

func matchStep(line string) StepType {
	switch {
	case strings.HasPrefix(line, ThoughtStep.Prefix()):
		return ThoughtStep
	case strings.HasPrefix(line, ActionStep.Prefix()):
		return ActionStep
	case strings.HasPrefix(line, AssumptionStep.Prefix()):
		return AssumptionStep
	case strings.HasPrefix(line, ObservationStep.Prefix()):
		return ObservationStep
	case strings.HasPrefix(line, QuestionStep.Prefix()):
		return QuestionStep
	case strings.HasPrefix(line, AnswerStep.Prefix()):
		return AnswerStep
	case strings.HasPrefix(line, FinalAnswerStep.Prefix()):
		return FinalAnswerStep
	default:
		return Unrecognized
	}
}

// Step represents an individual step in the conversation or process log, which includes
// a thought, action, action input, observation or final answer.
type Step struct {
	Type     StepType
	Content  string
	Argument string
}

func (s Step) String() string {
	if s.Type == ActionStep {
		return fmt.Sprintf("%s: %s\n%s", s.Type, s.Argument, s.Content)
	}
	return fmt.Sprintf("%s: %s", s.Type, s.Content)
}

// Conversation represents a sequence of conversation steps
type Conversation struct {
	Question string
	Steps    []Step
}

// NewConversation returns an empty conversation for a given question
func NewConversation(question string) *Conversation {
	return &Conversation{Question: question}
}

// Parse parses a conversation from a string.
func Parse(text string) (steps []Step, err error) {
	// Split the text into lines
	lines := strings.Split(text, "\n")

	currentStep := Step{}

	for _, line := range lines {
		stepType := matchStep(line)

		if stepType.IsRecognized() {
			// We found the beginning of a new step.
			if currentStep.Type.IsRecognized() {
				// There was a previous step in progress, so we should finalize
				// it and add it to the list of steps.
				steps = append(steps, currentStep)
			}
			currentStep = Step{Type: stepType}
			if stepType != ActionStep {
				currentStep.Content = strings.TrimSpace(strings.TrimPrefix(line, stepType.Prefix()))
			} else {
				// Split the line into the step type and the argument the first
				// value returned by strings.Cut is going to be the step type.
				// `found` is always going to be true, as otherwise the step
				// type would have been Unrecognized
				_, arg, _ := strings.Cut(line, ": ")

				currentStep.Argument = strings.TrimSpace(arg)
			}

		} else if currentStep.Type.IsRecognized() {
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

// FinalAnswer is a method on Conversation that returns the final answer from the conversation steps
func (c *Conversation) FinalAnswer() (string, error) {
	// implementation goes here
	return "", nil
}
