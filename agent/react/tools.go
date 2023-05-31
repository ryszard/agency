package react

import (
	"context"
	"fmt"
	"sort"
)

// Tool is a interface encapsulating a tool that can be used by the agent.
type Tool interface {
	// Name returns the name of the tool.
	Name() string
	// Description returns a description of the tool.
	Description() string
	// Input returns what kind of input the tool expects.
	Input() string
	// Work runs the tool. The observation is what is going to be passed to the
	// agent.
	Work(ctx context.Context, argument, content string) (observation string, err error)
}

type toolbox struct {
	tools map[string]Tool
}

func (box *toolbox) Tools() (tools []Tool) {
	tools = make([]Tool, 0, len(box.tools))
	for _, tool := range box.tools {
		tools = append(tools, tool)
	}

	// We want them sorted by name.
	sort.Slice(tools, func(i, j int) bool {
		return tools[i].Name() < tools[j].Name()
	})

	return tools
}

func newToolbox(tools ...Tool) *toolbox {
	tb := &toolbox{
		tools: make(map[string]Tool),
	}
	for _, tool := range tools {
		tb.tools[tool.Name()] = tool
	}
	return tb
}

func (box *toolbox) Work(ctx context.Context, argument, content string) (observation string, err error) {
	tool, ok := box.tools[argument]
	if !ok {
		return "", fmt.Errorf("unknown tool: %q", argument)
	}
	return tool.Work(ctx, argument, content)
}
