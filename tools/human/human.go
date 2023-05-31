package human

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
)

type Tool struct {
	reader io.Reader
}

// New returns a tool that allows the agent to ask a human a question, which
// will be read from the reader (usually os.Stdin).
func NewFromReader(reader io.Reader) Tool {
	return Tool{reader: reader}
}

// New returns a tool that allows the agent to ask a human a question, which
// will be read from os.Stdin.
func New() Tool {
	return NewFromReader(os.Stdin)
}

func (t Tool) Name() string {
	return "human"
}

func (t Tool) Description() string {
	return "A tool that allows you to ask a human a question."
}

func (Tool) Input() string {
	return "A natural language question."
}

func (Tool) Work(_ context.Context, _, content string) (observation string, err error) {
	fmt.Printf("Question to Human: %s\n", content)
	// Read the answer from standard input
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Answer: ")
	answer, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Observation: Answer from human: %s\n", answer), nil
}
