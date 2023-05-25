package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"

	"github.com/ryszard/agency/agent"
	"github.com/ryszard/agency/util/cache"
	"gopkg.in/yaml.v3"
)

type AgentConfig struct {
	Config    agent.Config      `yaml:"config"`
	Templates map[string]string `yaml:"templates"`
}

func (ac *AgentConfig) ParseTemplates() (map[string]*template.Template, error) {
	templates := make(map[string]*template.Template)
	for name, text := range ac.Templates {
		tmpl, err := template.New(name).Parse(text)
		if err != nil {
			return nil, err
		}
		templates[name] = tmpl
	}
	return templates, nil
}

func (cfg *AgentConfig) Agent(name string, client agent.Client, out io.Writer, cache *cache.BoltDBCache) (*TemplatedAgent, error) {
	templates, err := cfg.ParseTemplates()
	if err != nil {
		return nil, err
	}

	var ag agent.Agent = agent.FromConfig(name, cfg.Config, agent.WithClient(client))

	if cache != nil {
		ag = agent.WithCache(ag, cache)
	}
	return &TemplatedAgent{
		Agent:     ag,
		Templates: templates,
		out:       out,
	}, nil

}

type FlowConfig struct {
	IntentionEvaluator AgentConfig `yaml:"intention_evaluator"`

	Author AgentConfig `yaml:"author"`

	Critic AgentConfig `yaml:"critic"`
	cache  *cache.BoltDBCache
}

type Flow struct {
	Config             FlowConfig
	IntentionEvaluator *TemplatedAgent
	Author             *TemplatedAgent
}

func Load(path string, client agent.Client, out io.Writer, cach *cache.BoltDBCache) (*Flow, error) {

	data, err := os.ReadFile(*config)
	if err != nil {
		return nil, err
	}

	cfg := FlowConfig{cache: cach}

	// load its contents into the flow struct.
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	ie, err := cfg.IntentionEvaluator.Agent("intention_evaluator", client, out, cach)
	if err != nil {
		return nil, err
	}

	author, err := cfg.Author.Agent("author", client, out, cach)
	if err != nil {
		return nil, err
	}

	flow := Flow{
		Config:             cfg,
		IntentionEvaluator: ie,
		Author:             author,
	}

	return &flow, nil

}

type TemplatedAgent struct {
	agent.Agent
	Templates map[string]*template.Template
	out       io.Writer
}

func (ta *TemplatedAgent) ListenFromTemplate(templateName string, data any) error {
	template := ta.Templates[templateName]
	if template == nil {
		return fmt.Errorf("template %s not found", templateName)
	}

	var message strings.Builder
	if err := template.Execute(&message, data); err != nil {
		return err
	}

	ta.Agent.Listen(message.String())
	fmt.Fprintf(ta.out, "User ➡️ %s\n", ta.Name())
	ta.out.Write([]byte(message.String()))
	ta.out.Write([]byte("\n"))
	return nil
}
