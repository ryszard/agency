package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ryszard/agency/agent"
	"github.com/ryszard/agency/client"
	"github.com/ryszard/agency/client/exp/huggingface"
	"github.com/ryszard/agency/client/openai"
	log "github.com/sirupsen/logrus"
)

var (
	model       = flag.String("model", "gpt-3.5-turbo", "model to use")
	maxTokens   = flag.Int("max_tokens", 1000, "maximum context length")
	temperature = flag.Float64("temperature", 0.7, "temperature")
	logLevel    = flag.String("log_level", "error", "log level")
	platform    = flag.String("platform", "openai", "platform to use")
)

func main() {
	flag.Parse()

	level, err := log.ParseLevel(*logLevel)
	if err != nil {
		log.Fatal(err)
	}

	log.SetLevel(level)
	log.SetReportCaller(true)
	var cl client.Client
	switch *platform {
	case "openai":
		cl = openai.New(os.Getenv("OPENAI_API_KEY"))
	case "huggingface":
		cl = huggingface.New(os.Getenv("HUGGINGFACE_API_KEY"))
	default:
		log.Fatalf("unknown platform: %s", *platform)
	}

	cl = client.Retrying(cl, 1*time.Second, 30*time.Second, 20)

	bot := agent.New("assistant",
		agent.WithClient(cl),
		agent.WithModel(*model),
		agent.WithMaxTokens(*maxTokens),
		agent.WithTemperature(float32(*temperature)),
		agent.WithCustomParams(map[string]interface{}{
			"repetition_penalty": 40.0,
			"wait_for_model":     true,
		}),
		//agent.WithMemory(agent.SummarizerMemory(0.5)),
	)

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Printf("model: %s\n", *model)
	fmt.Printf("max_tokens: %d\n", *maxTokens)
	fmt.Printf("temperature: %f\n", *temperature)
	fmt.Println("Start")
	fmt.Print("You: ")

	for scanner.Scan() {
		input := scanner.Text()
		_, err := bot.Listen(input)
		if err != nil {
			fmt.Printf("An error occurred: %v\n", err)
			continue
		}
		fmt.Println("Bot:")
		_, err = bot.Respond(context.Background(), agent.WithStreaming(os.Stdout))
		if err != nil {
			fmt.Printf("An error occurred: %v\n", err)
			continue
		}
		fmt.Print("You: ")
	}

	if scanner.Err() != nil {
		fmt.Fprintf(os.Stderr, "Reading standard input: %s", scanner.Err())
	}

}
