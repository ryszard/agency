package bash

import (
	"bytes"
	"os/exec"
	"strings"
)

// Response is a struct that represents a response from the REPL.
type Response struct {
	Out string `json:"out"`
	Err string `json:"err"`
}

type BashExecutor struct {
	Path string
}

// New returns a prepared BashExecutor.
func New(path string) *BashExecutor {
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
