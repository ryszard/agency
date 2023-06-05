package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/ryszard/agency/agent"
	"github.com/ryszard/agency/agent/react"
	"github.com/ryszard/agency/client"
	"github.com/ryszard/agency/client/openai"
	"github.com/ryszard/agency/tools/bash"
	"github.com/ryszard/agency/tools/human"
	"github.com/ryszard/agency/tools/python"
	"github.com/ryszard/agency/util/cache"

	log "github.com/sirupsen/logrus"
)

var (
	question     = flag.String("question", `For the following names, please concatenate the first letter of the name, and the last letter of the surname: "Ryszard Szopa", "Bill Clinton", "Sam Harris", "Barack Obama". To that, concatenate the length of the name and the Python interpreter version.`, "question to ask")
	model        = flag.String("model", "gpt-3.5-turbo", "model to use")
	maxTokens    = flag.Int("max_tokens", 2500, "maximum context length. Note that this should be enough to fit the question and the system prompt.")
	memoryTokens = flag.Int("memory_tokens", 1000, "number of tokens to keep in memory")
	temperature  = flag.Float64("temperature", 0.7, "temperature")
	logLevel     = flag.String("log_level", "info", "log level")

	pythonPath = flag.String("python_path", "/opt/homebrew/anaconda3/bin/python", "path to a python interpreter")
)

func main() {
	flag.Parse()

	level, err := log.ParseLevel(*logLevel)
	if err != nil {
		log.Fatal(err)
	}

	log.SetLevel(level)

	var cl client.Client = openai.New(os.Getenv("OPENAI_API_KEY"))

	cach, err := cache.BoltDB("./cache.db")
	if err != nil {
		log.WithError(err).Fatal("error")
	}

	cl = client.Cached(cl, cach)

	ag := agent.New("pythonista",
		agent.WithClient(cl),
		agent.WithMaxTokens(*maxTokens),
		agent.WithTemperature(float32(*temperature)),
		agent.WithModel(*model),
		agent.WithStreaming(os.Stdout),
		agent.WithMemory(agent.TokenBufferMemory(*memoryTokens)),
	)

	bash := bash.New("/bin/bash")
	python, err := python.New(*pythonPath)
	if err != nil {
		log.WithError(err).Fatal("error")
	}

	defer python.Close()

	human := human.New()

	reactor := react.New(ag, bash, python, human)

	if *question != "" {

		if err := reactor.Answer(context.Background(), *question); err != nil {
			log.WithError(err).Fatal("error")
		}
	}
	reader := bufio.NewReader(os.Stdin)

	for {
		// Read the question from stdin.
		fmt.Print("Question: ")
		newQuestion, err := reader.ReadString('\n')
		if err != nil {
			log.WithError(err).Fatal("error reading from stdin")
		}

		if err := reactor.Answer(context.Background(), newQuestion); err != nil {
			log.WithError(err).Fatal("error")
		}

	}
}
