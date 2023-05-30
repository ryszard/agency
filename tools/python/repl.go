package python

import (
	"bufio"
	_ "embed"
	"encoding/json"
	"io"
	"os/exec"

	log "github.com/sirupsen/logrus"
)

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

// New creates a new Python REPL.
func New(command string) (*PythonREPL, error) {
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
