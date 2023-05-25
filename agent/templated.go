package agent

import (
	"fmt"
	"strings"
	"text/template"

	log "github.com/sirupsen/logrus"
)

// TemplatedAgent is an agent that supports templating. It wraps another agent
// and uses templates to format messages before sending them to the wrapped
// agent.
type TemplatedAgent struct {
	Agent
	Templates map[string]*template.Template
}

// Templated wraps an agent with a templating layer (the returned agent's
// real type is *TemplatedAgent). The templates are parsed from the given map
// of template names to template texts.
func Templated(ag Agent, templatesText map[string]string) (Agent, error) {
	templates := make(map[string]*template.Template)
	for name, text := range templatesText {
		tmpl, err := template.New(name).Parse(text)
		if err != nil {
			return nil, err
		}
		templates[name] = tmpl
	}

	return &TemplatedAgent{
		Agent:     ag,
		Templates: templates,
	}, nil
}

var _ Agent = &TemplatedAgent{}

// Listen implements the Agent interface. Instead of sending the message
// directly to the wrapped agent, it first formats it using the template with
// the given name and the given data. The data is passed to the template's
// Execute method.
func (ag *TemplatedAgent) Listen(templateName string, data ...any) error {
	if len(data) > 1 {
		return fmt.Errorf("templated agent only supports one data argument")
	}
	datum := data[0]
	template := ag.Templates[templateName]
	if template == nil {
		return fmt.Errorf("template %s not found", templateName)
	}

	var message strings.Builder
	if err := template.Execute(&message, datum); err != nil {
		return err
	}

	ag.Agent.Listen(message.String())
	log.WithField("templated agent messages", ag.Messages()).Debug("agent messages")

	out := ag.Config().out()
	fmt.Fprintf(out, "User ➡️ %s\n", ag.Name())
	out.Write([]byte(message.String()))
	out.Write([]byte("\n"))
	return nil
}
