package react

import (
	"bufio"
	"context"
	_ "embed"
	"fmt"
	"os"

	"github.com/ryszard/agency/agent"
	"github.com/ryszard/agency/util/python"
	log "github.com/sirupsen/logrus"
)

//go:embed react_prompt.md
var systemPrompt string

func Work(ctx context.Context, client agent.Client, pythonPath string, cache agent.Cache, question string, options ...agent.Option) error {
	ag := agent.New("pythonista", options...)

	ag = agent.Cached(ag, cache)

	err := ag.System(systemPrompt)
	if err != nil {
		return err
	}

	python, err := python.New(pythonPath)
	if err != nil {
		return err
	}

	defer python.Close()

	steps := []Step{}

	_, err = ag.Listen(fmt.Sprintf("Question: %s", question))
	if err != nil {
		return err
	}

	for {
		msg, err := ag.Respond(context.Background())
		if err != nil {
			return err
		}

		log.WithField("msg", msg).Info("received message")

		newSteps, err := Parse(msg)
		if err != nil {
			return err
		}
		log.WithField("newSteps", fmt.Sprintf("%+v", newSteps)).Info("parsed message")
		//steps = append(steps, newSteps...)
		for _, step := range newSteps {
			fmt.Printf("%s\n", step)
		}

		steps = append(steps, newSteps...)
		lastStep := steps[len(steps)-1]

		if lastStep.Type == FinalAnswerStep {
			return nil
		} else if lastStep.Type != ActionStep {
			_, err := ag.Listen("Please continue.")
			if err != nil {
				return err
			}
			continue
		}
		var observation string
		switch lastStep.Argument {
		case "python":
			stdout, stderr, err := python.Execute(lastStep.Content)
			if err != nil {
				return err
			}
			fmt.Printf("stdout: %s\nstderr: %s\n", stdout, stderr)
			observation = fmt.Sprintf("Observation: \nStandard Output: %s\nStandardError:\n%s\n", stdout, stderr)

		case "human":
			// Print the question
			fmt.Printf("Question to Human: %s\n", question)
			// Read the answer from standard input
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Answer: ")
			answer, err := reader.ReadString('\n')
			if err != nil {
				return err
			}
			observation = fmt.Sprintf("Observation: Answer from human: %s\n", answer)

		}

		if _, err := ag.Listen(observation); err != nil {
			return err
		}

		fmt.Println("\n" + observation + "\n")

	}
	return nil
}
