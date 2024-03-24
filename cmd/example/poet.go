package main

import (
	"context"
	"fmt"
	"os"

	"github.com/ryszard/agency/agent"
	"github.com/ryszard/agency/client/openai"
	log "github.com/sirupsen/logrus"
)

func main() {
	client := openai.New(os.Getenv("OPENAI_API_KEY"))

	// Initialize a poet agent and a critic agent
	poet := agent.New("poet",
		agent.WithClient(client),
		agent.WithModel(openai.GPT3Dot5Turbo),
		agent.WithMaxTokens(2000),
		agent.WithTemperature(0.7))

	critic := agent.New("critic",
		agent.WithClient(client),
		agent.WithModel(openai.GPT3Dot5Turbo),
		agent.WithMaxTokens(2000))

	// Set the topic for the haiku
	topic := "sunrise"

	// The poet writes a haiku about the given topic
	_, err := poet.Listen(fmt.Sprintf("Write a haiku about a %s", topic))
	if err != nil {
		log.Fatalf("Poet Listen failed: %v", err)
	}

	haiku, err := poet.Respond(context.Background())
	if err != nil {
		log.Fatalf("Poet Respond failed: %v", err)
	}

	fmt.Printf("Haiku:\n%s\n", haiku)

	// The critic critiques the haiku
	_, err = critic.Listen(fmt.Sprintf("Please critique this haiku: \n%s", haiku))
	if err != nil {
		log.Fatalf("Critic Listen failed: %v", err)
	}

	critique, err := critic.Respond(context.Background())
	if err != nil {
		log.Fatalf("Critic Respond failed: %v", err)
	}

	fmt.Printf("Critique: %s\n", critique)

	// Create a loop for the poet to improve and the critic to critique
	for i := 0; i < 10; i++ {
		// Poet takes into account the critique and tries to improve
		_, err = poet.Listen(fmt.Sprintf("Feedback received: '%s'. Please improve the haiku.", critique))
		if err != nil {
			log.Fatalf("Poet Listen failed: %v", err)
		}

		haiku, err = poet.Respond(context.Background())
		if err != nil {
			log.Fatalf("Poet Respond failed: %v", err)
		}

		// The critic critiques the improved haiku
		_, err = critic.Listen(fmt.Sprintf("Please critique this improved haiku: \n%s", haiku))
		if err != nil {
			log.Fatalf("Critic Listen failed: %v", err)
		}

		critique, err = critic.Respond(context.Background())
		if err != nil {
			log.Fatalf("Critic Respond failed: %v", err)
		}

		fmt.Printf("Improved Haiku: %s\nCritique: %s\n", haiku, critique)
	}
}
