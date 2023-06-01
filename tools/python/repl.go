// Package python provides a tool that allows ReAct agent from
// github.com/ryszard/agency/agent/react to run Python code in a repl.
package python

import (
	"bufio"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"

	log "github.com/sirupsen/logrus"
)

// Tool allows the agent to run Python code in a repl. Obviously, this is very
// dangerous, use it at your own risk.
type Tool struct {
	*PythonREPL
}

func (Tool) Name() string {
	return "python"
}

func (Tool) Description() string {
	return "A Python process you can use to run Python code. Usage limites: use it as much as you want."
}

func (Tool) Input() string {
	return "Python code"
}

func (b Tool) Work(_ context.Context, _, content string) (observation string, err error) {
	resp, err := b.Execute(content)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("stdout:\n\n%q\nstderr:\n\n%q", resp.Out, resp.Err), nil
}

// New returns a tool that allows the agent to run Python code.
func New(path string) (Tool, error) {
	repl, err := NewREPL(path)
	if err != nil {
		return Tool{}, err
	}
	return Tool{
		PythonREPL: repl,
	}, nil
}

//go:embed repl.py
var pythonScript string

type REPL interface {
	Execute(code string) (Response, error)
	Close() error
}

// PythonCode is a struct that represents a Python code snippet to be sent to
// the REPL.
type PythonCode struct {
	Code string `json:"code"`
}

// Response is a struct that represents a response from the REPL.
type Response struct {
	Out string `json:"out"`
	Err string `json:"err"`
}

// PythonREPL represents a Python REPL.
type PythonREPL struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
}

// NewREPL creates a new Python REPL.
func NewREPL(command string) (*PythonREPL, error) {
	cmd := exec.Command(command, "-c", pythonScript)
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return &PythonREPL{
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
	}, nil
}

// Execute executes a Python code snippet in the REPL.
func (repl *PythonREPL) Execute(code string) (response Response, err error) {
	pythonCode := PythonCode{Code: code}
	jsonData, _ := json.Marshal(pythonCode)
	jsonData = append(jsonData, '\n')
	if _, err := repl.stdin.Write(jsonData); err != nil {
		return response, err
	}
	log.WithField("json", string(jsonData)).Debug("Sent code to Python REPL")

	scanner := bufio.NewScanner(repl.stdout)
	scanner.Scan()

	line := scanner.Text()
	log.WithField("line", line).Debug("Received response from Python REPL")
	if err := json.Unmarshal([]byte(line), &response); err != nil {
		return Response{}, err
	}

	return response, nil
}

func (repl *PythonREPL) Close() {
	repl.stdin.Close()
	repl.cmd.Wait()
}
