package main

import (
	"fmt"
	"io"
	"os"

	"github.com/ryszard/agency/agent"
	"github.com/ryszard/agency/util/cache"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type AgentConfig struct {
	Config    agent.Config      `yaml:"config"`
	Templates map[string]string `yaml:"templates"`
}

func (cfg *AgentConfig) Agent(name string, client agent.Client, out io.Writer, cache *cache.BoltDBCache) (ag agent.Agent, err error) {

	ag = agent.New(name,
		agent.WithConfig(cfg.Config),
		agent.WithClient(client),
		agent.WithStreaming(out))

	log.WithField("name", name).WithField("agent", fmt.Sprintf("%+v", ag)).Debug("agent created")

	ag, err = agent.Templated(ag, cfg.Templates)
	if err != nil {
		return nil, err
	}

	if cache != nil {
		ag = agent.Cached(ag, cache)
	}

	return ag, nil

}

type FlowConfig struct {
	IntentionEvaluator AgentConfig `yaml:"intention_evaluator"`

	Author AgentConfig `yaml:"author"`

	Critic AgentConfig `yaml:"critic"`
	cache  *cache.BoltDBCache
}

type Flow struct {
	Config             FlowConfig
	IntentionEvaluator agent.Agent
	Author             agent.Agent
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
