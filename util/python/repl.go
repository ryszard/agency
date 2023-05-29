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

type PythonCode struct {
	Code string `json:"code"`
}

type PythonResponse struct {
	Out string `json:"out"`
	Err string `json:"err"`
}

type PythonREPL struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
}

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

func (repl *PythonREPL) Execute(code string) (string, string, error) {
	pythonCode := PythonCode{Code: code}
	jsonData, _ := json.Marshal(pythonCode)
	jsonData = append(jsonData, '\n')
	if _, err := repl.stdin.Write(jsonData); err != nil {
		return "", "", err
	}
	log.WithField("json", string(jsonData)).Debug("Sent code to Python REPL")

	scanner := bufio.NewScanner(repl.stdout)
	scanner.Scan()

	line := scanner.Text()
	log.WithField("line", line).Debug("Received response from Python REPL")
	response := PythonResponse{}
	if err := json.Unmarshal([]byte(line), &response); err != nil {
		return "", "", err
	}

	return response.Out, response.Err, nil
}

func (repl *PythonREPL) Close() {
	repl.stdin.Close()
	repl.cmd.Wait()
}
