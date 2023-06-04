# 🏢🤖Agency
🚀Robust LLM Agent Management with Go🚀

## 🎯 Overview

Agency is a Go package designed to provide an idiomatic interface to various LLM providers (currently supporting OpenAI and, experimentally, 🤗Hugging Face). The package aims to simplify the creation and management of Language Learning Model (LLM)-based agents, making it easier for you to work with multiple agents and manage their data flow. It hopes to enable easier implementation of autonomous agent systems, similar to AutoGPT or BabyAGI, to solve a variety of tasks.

It also provides some features one may need to deploy the code into production, like caching, retrying, or rate limiting.

The heart and soul of Agency is the `agent.Agent` type. Its interface features three core methods that allow for convenient communication with AI agents: 

- `Listen`: communicates text to the AI.
- `System`: sends a system message.
- `Respond`: receives responses from the agent.

Below is a simple usage example:

```go
client := openai.New(os.Getenv("OPENAI_API_KEY")) // This is github.com/ryszard/agency/client/openai
ag := agent.New("poet", agent.WithClient(client), agent.WithModel("gpt-4"))
_, err = ag.Listen("Please write a limerick about a Common Lisp programmer from Reno")
if err != nil {
    panic(err)
}
limerick, err := ag.Respond(context.Background())
if err != nil {
    panic(err)
}
fmt.Println(limerick)
```

As you can see, Agent behavior can be fine-tuned via Options, as exemplified by agent.WithModel. Options can also be passed to Agent.Respond to modify a specific request's behavior:

```go
message, err := ag.Respond(context.Background(), agent.WithStreaming(os.Stdout))
if err != nil {
    log.Fatalf("failed to respond: %v", err)
}
```

### 🧠Managing Agent Memory
LLMs have a token limitation within their context window. Agency addresses this by providing the option to give an agent a memory, using the agent.WithMemory option:

```go
ag := agent.New("Funes", agent.WithClient(client), agent.WithMemory(agent.TokenBufferMemory(0.9))
```

Agency offers several memory implementations (`agent.BufferMemory`, `agent.TokenBufferMemory`, and `agent.SummarizerMemory`). If these don't fit your needs, you can implement your own by adhering to the agent.Memory interface, which is simply a function that takes in a context, configuration, and list of openai.ChatCompletionMessage, and returns a modified list of messages.

### 🚀Optimizing Performance and Reliability

In order be more robust and efficient, Agency incorporates features such as retry mechanisms with exponential backoff, rate limiting, and caching. Caching especially beneficial when you need to modify prompts frequently during development and wish to avoid excessive latency or redundant API calls.

```go
var client client.Client = openai.New(os.Getenv("OPENAI_API_KEY"))

// If you fail, wait 1 second on the first retry, 2 seconds on the second, 
// and so on, until your reach either 20 retries or 5 minutes.
client = client.Retrying(client, 1 * time.Second, 5 * time.Minute, 20)

// golang.org/x/time/rate is a great rate limiting library.
limiter := rate.NewLimiter(10, 1)

client = client.RateLimiting(client, limiter)

ag := agent.New("hardened")

// github.com/ryszard/utils/cache provides an in-memory and a BoldDB-based cache.
cach := cache.Memory()
ag = agent.Cached(ag, cach)
```

### 🔍Delving Deeper
To fully appreciate the potential of Agency, consider exploring [Agency's implementation of the ReAct framework]((https://github.com/ryszard/agency/blob/main/agent/react/agent.go)). Agency is designed to facilitate the construction of such complex agent interactions.

# 🗺️ Roadmap

 - Agency currently only supports OpenAI's API. Add support for other LLMs, like Anthropic's Claude or completion models from Hugging Face.
 - More memory implementations, especially ones using vector databases like Faiss.



# ❓Frequently Asked Questions

## 🔄 Why not just use LangChain🦜️🔗?

While I hold immense respect for LangChain and its impressive development speed, it does bear a few drawbacks. Primarily, LangChain's code is somewhat cryptic - it offers appealing APIs but complicates their debugging and reasoning process. I've found that deviating slightly from the original authors' intent can lead to confusion. Moreover, due to its metaprogramming usage, understanding LangChain's code becomes a daunting task. Lastly, LangChain's interface doesn't resonate with me intuitively, although this is subjective.

## 🐍Why choose Go over Python?
Despite Python being the default choice for AI work, it is not without its issues. Writing high-performance concurrent code in Python is notoriously challenging. Additionally, dependency management is complicated, and AI libraries are usually demanding and tricky to install. As much AI work nowadays consists of calling APIs like OpenAI's, it has opened doors to using other languages. Hence, my preference for Go led to the creation of Agency.

## 🏗️How production-ready is Agency?
Currently, Agency is in its early development stage and not quite ready for production use. The APIs may undergo changes without prior notification. However, due to interest from some of my friends, I decided to make this a public repository for anyone keen on exploring its potential.

## 🔧Installation
Install Agency with the following command:

```bash
go get github.com/ryszard/agency
```
# 📚 Documentation

Comprehensive documentation for Agency is available on GoDoc. The GoDoc page provides a detailed overview of the package, including its functions, types, and methods, along with their usage examples.

To access the Agency documentation, follow this link: [GoDoc](https://godoc.org/github.com/ryszard/agency)

The GoDoc documentation should provide you with sufficient information to get started with Agency. If you have further questions, feel free to create an issue on this GitHub repository.

Remember to always keep your `OPENAI_API_KEY` secure and to adhere to the OpenAI use case policy when using the OpenAI API. *This line was ChatGPT's idea. I decided to humor it.*

## 👨‍💻Usage
Here's a full working example of how you can use Agency to create two agents, a poet and a critic, that interact with each other to create better content.

```go
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
	log.SetLevel(log.ErrorLevel)
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

```

In this example, the poet writes a haiku about a given topic, and the critic provides feedback on the poem. Then, in a loop, the poet attempts to improve the poem based on the critic's feedback, and the critic provides another round of critique. This iterative improvement and critique process continues for a set number of iterations.

## 📝Notes
Please note that this package is still in the early stages of development and is not production-ready. Use it at your own risk, and feel free to contribute to its development.

## 🤝Contributing
I welcome contributions! Please send me a pull request.

## 📄License
This project is licensed under the terms of the MIT license. See LICENSE for details.

## 📞Contact
If you have any questions, feel free to open an issue.

