// Package bash implements a tool that allows the ReAct agent defined in
// agent/react/agent.go to run bash commands.
package bash

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// BashTool is a tool that allows the agent to run bash commands. Obviously,
// this is very dangerous, use it at your own risk.
type BashTool struct {
	Executor *BashExecutor
}

func (BashTool) Name() string {
	return "bash"
}

func (BashTool) Description() string {
	return `Description: This allows you to run bash commands. It interacts with the file system and the network, in an environent that is similar to the one you are in and that I created for you. You can use it to run any bash command you want, including running other tools, like the go tool. Yes, you can interact with the file system. Note that this tool passes to the code using "/bin/bash -c <your code>", so don't expect that environment variables you set will be available in the next command you run. Usage Limits: use it as much as you want.`
}

func (BashTool) Input() string {
	return "A bash script."
}

func (b BashTool) Work(_ context.Context, _, content string) (observation string, err error) {
	resp, err := b.Executor.Execute(content)
	return fmt.Sprintf("stdout:\n\n%q\nstderr:\n\n%q, error: %#v", resp.Out, resp.Err, err), nil
}

// New returns a prepared BashTool, which implements the Tool interface.
func New(path string) BashTool {
	return BashTool{Executor: NewExecutor(path)}
}

// Response is a struct that represents a response from Bash.
type Response struct {
	Out string `json:"out"`
	Err string `json:"err"`
}

type BashExecutor struct {
	Path string
}

// NewExecutor returns a prepared BashExecutor.
func NewExecutor(path string) *BashExecutor {
	return &BashExecutor{Path: path}
}

// Execute takes bash code as a string and executes it.
// It returns the outputs from stdout, stderr and an error if the execution fails.
func (b *BashExecutor) Execute(code string) (response Response, err error) {
	cmd := exec.Command(b.Path, "-c", code)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	response.Out = strings.TrimSpace(stdout.String())
	response.Err = strings.TrimSpace(stderr.String())

	return response, err
}

func (*BashExecutor) Close() error {
	return nil
}
