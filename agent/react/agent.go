package react

import (
	"bufio"
	"context"
	_ "embed"
	"fmt"
	"os"

	"github.com/ryszard/agency/agent"
	"github.com/ryszard/agency/tools/bash"
	"github.com/ryszard/agency/tools/python"
	log "github.com/sirupsen/logrus"
)

//go:embed react_prompt.md
var SystemPrompt string

func Work(ctx context.Context, client agent.Client, pythonPath string, cache agent.Cache, question string, options ...agent.Option) error {
	ag := agent.New("pythonista", options...)

	ag = agent.Cached(ag, cache)

	_, err := ag.System(SystemPrompt)
	if err != nil {
		return err
	}

	python, err := python.New(pythonPath)
	if err != nil {
		return err
	}

	defer python.Close()

	// FIXME(ryszard): Pass this from the outside.
	bash := bash.New("/bin/bash")
	defer bash.Close()

	steps := []Entry{}

	_, err = ag.System(fmt.Sprintf("Question: %s", question))
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
		actionNotLast := false
		observationsOutput := false
		for i, step := range newSteps {
			fmt.Printf("%s\n", step)
			if step.Tag == Tags.Action && i != len(newSteps)-1 {
				actionNotLast = true
			} else if step.Tag == Tags.Observation {
				observationsOutput = true
			}
		}

		if actionNotLast || observationsOutput {
			var scolding string
			if actionNotLast {
				scolding = "Please provide an Action as the last step!"
			}
			if observationsOutput {
				scolding += "You are not allowed to provide your own observations!"
			}
			_, err := ag.Listen(scolding)
			if err != nil {
				return err
			}
			continue
		}

		steps = append(steps, newSteps...)
		lastStep := steps[len(steps)-1]

		if lastStep.Tag == Tags.FinalAnswer {
			return nil
		} else if lastStep.Tag != Tags.Action {
			_, err := ag.Listen("Please continue.")
			if err != nil {
				return err
			}
			continue
		}
		var observation string
		switch lastStep.Argument {
		case "python":
			resp, err := python.Execute(lastStep.Content)
			if err != nil {
				return err
			}
			fmt.Printf("stdout: %s\nstderr: %s\n", resp.Out, resp.Err)
			observation = fmt.Sprintf("Observation: \nStandard Output: %s\nStandardError:\n%s\n", resp.Out, resp.Err)

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

		case "bash":
			fmt.Printf("Running bash command: %s\n", lastStep.Content)

			resp, err := bash.Execute(lastStep.Content)
			observation = fmt.Sprintf("Observation: \nStandard Output:\n%s\nStandardError:\n%s\n\nExecution Error: %#v", resp.Out, resp.Err, err)

		}

		if _, err := ag.Listen(observation); err != nil {
			return err
		}

		fmt.Println("\n" + observation + "\n")

	}
	return nil
}
