// Package react provides a framework for creating and managing reactive agents,
// inspired by the paper "Reactive Agents: A Conversational AI Approach".
//
// The core of the package is the ReAct struct, which encapsulates a reactive
// agent. Agents can be created with default or custom system prompts and can
// answer questions in a context-aware manner.
//
// See github.com/ryszard/agency/cmd/react/ for an example of ReAct in use.
package react

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"

	"github.com/ryszard/agency/agent"
	log "github.com/sirupsen/logrus"
)

//go:embed templates/react_prompt.md
var SystemPromptTemplate string

// ReAct is a reactive agent.
type ReAct struct {
	agent          agent.Agent
	box            *toolbox
	systemTemplate *template.Template
	initialized    bool
	writer         io.Writer
}

// New creates a new ReAct agent with the default system prompt and writing to
// os.Stdout.
func New(ag agent.Agent, tools ...Tool) *ReAct {
	tpl := template.Must(template.New("system_prompt").Parse(SystemPromptTemplate))
	return NewReAct(ag, os.Stdout, tpl, tools...)
}

// NewWithTemplate creates a new reactive agent with a custom system prompt. The
// provided template must use the same data as the default template. The writer will be used to
// write the conversation.
func NewReAct(ag agent.Agent, w io.Writer, tpl *template.Template, tools ...Tool) *ReAct {
	return &ReAct{
		agent:          ag,
		box:            newToolbox(tools...),
		systemTemplate: tpl,
		writer:         w,
	}
}

// Answer ask the agent a question and returns the answer.
func (reactor *ReAct) Answer(ctx context.Context, question string, options ...agent.Option) error {

	if !reactor.initialized {
		var sb strings.Builder
		if err := reactor.systemTemplate.Execute(&sb, reactor.box); err != nil {
			return err
		}

		_, err := reactor.agent.System(sb.String())
		if err != nil {
			return err
		}
	}
	entries := []Entry{}

	_, err := reactor.agent.System(fmt.Sprintf("Question: %s", question))
	if err != nil {
		return err
	}

	for {
		msg, err := reactor.agent.Respond(context.Background(), options...)
		if err != nil {
			return err
		}

		log.WithField("msg", msg).Info("received message")

		newEntries, err := Parse(msg)
		if err != nil {
			return err
		}
		log.WithField("newEntries", fmt.Sprintf("%+v", newEntries)).Info("parsed message")
		actionNotLast := false
		observationsOutput := false
		for i, step := range newEntries {
			fmt.Printf("%s\n", step)
			if step.Tag == Tags.Action && i != len(newEntries)-1 {
				actionNotLast = true
			} else if step.Tag == Tags.Observation {
				observationsOutput = true
			}
		}

		if actionNotLast || observationsOutput {
			var scolding string
			if actionNotLast {
				scolding = "Please provide an Action as the last entry!"
			}
			if observationsOutput {
				scolding += " You are not allowed to provide your own observations!"
			}
			_, err := reactor.agent.Listen(scolding)
			if err != nil {
				return err
			}
			continue
		}

		entries = append(entries, newEntries...)
		lastEntry := entries[len(entries)-1]

		if lastEntry.Tag == Tags.FinalAnswer {
			return nil
		} else if lastEntry.Tag != Tags.Action {
			_, err := reactor.agent.Listen("Please continue.")
			if err != nil {
				return err
			}
			continue
		}

		observation, err := reactor.box.Work(ctx, lastEntry.Argument, lastEntry.Content)
		if err != nil {
			return err
		}

		if _, err := reactor.agent.Listen(observation); err != nil {
			return err
		}

		fmt.Println("\n" + observation + "\n")

	}
}
