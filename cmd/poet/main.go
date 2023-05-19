package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/ryszard/agency/agent"
	"github.com/sashabaranov/go-openai"
	log "github.com/sirupsen/logrus"
)

func parseRating(ratingResponse string) (int, error) {
	// parse a number from 1 to 10 from a ratingResponse
	i, err := strconv.Atoi(ratingResponse)
	if err != nil {
		return 0, err
	}
	return i, nil

}

var criticSystem = `
Please act as a literary critic. You will receive a poem with the genre "%s" 
and the theme "%s". Here are some notes that could be helpful (may be empty): "%s".
You'll also be provided with the author's explanation of the poem.
Please provide actionable feedback on how the author can make it better.
The goal is high artistic value and originality - but the poem must also adhere
to the provided theme. Please make sure that the poem adheres to the genre as well. 
Make sure that it meets any rhyme or meter requirements. Feel free to challenge the
author to make the poem better, both in terms of artistic value, how the theme is 
expressed, and how well it adheres to the genre, but please be polite. Constructive
feedback is how great art is made.

Please do not provide your own revision.

Please ALWAYS respond in the JSON format, with the following fields:
 - feedback (string): your feedback on the poem. You should ALWAYS provide some 
   actionable feedback on how the poem can be improved. If you think the poem is
   perfect, please say so, but also provide some feedback on how it can be improved.
 - needs_more_work (float): how much you feel the poem needs more work, with
   0 meaning that the work is truly excellent and trying to change it would make 
   it worse rather than better, and 1 meaning that it definitely needs more work. 
   A value of 0 should be very rare. Note that if this is not 0, you should provide
   some feedback on how the poem can be improved.
`

type CriticResponse struct {
	Feedback      string  `json:"feedback"`
	NeedsMoreWork float64 `json:"needs_more_work"`
}

var poetSystem = `
Put yourself in the role of a poet. The user may ask you to write a poem about
some particular topic. Later, you will receive feedback on your work. Using that feedback,
please write a new poem. You will keep repeating this process.

No matter what happens, you should ALWAYS respond in the JSON format, with the following fields:
 - text (string): the text of your poem. This MUST be the poem as specified by the user. 
   NEVER respond with some sort of commentary on the feedback you might have received.
 - explanation (string): an explanation of the artistic choices you made.

For example, if the user asks you to write a haiku about a squirrel falling from a tree,
please return something like this (and nothing else):
"""
{
  "text": "A squirrel falls\nFrom the tree, and I wonder\nIf it is okay",
  "explanation": "Your explanation here"
}
"""

Please remember to always return a JSON object with the two fields above.
`

type PoetResponse struct {
	Text        string `json:"text"`
	Explanation string `json:"explanation"`
}

var poetUser = `
Please write a poem with the theme being \"%s\" and the genre described as \"%s\". 
Here are some notes that could be helpful in your writing: \"%s\" (but you are not
required to use them; they can also be empty).

Please do not forget to return JSON with the two fields: text and explanation.
`

var criticUser = `
Poem:

"""%s"""

Explanation:

"""%s"""

Notes:

"""%s"""

Please do not forget to return JSON with the two fields: feedback (string) and needs_more_work (float).
`

func main() {
	logLevel := flag.String("log_level", "debug", "log level (trace, debug, info, warning, error, fatal and panic)")

	poetTemperature := flag.Float64("poet_temperature", 0.99, "temperature for the creator")
	criticTemperature := flag.Float64("critic_temperature", 0, "temperature for the critic")

	poetMaxTokens := flag.Int("poet_max_tokens", 3000, "maximum tokens for the creator")
	criticMaxTokens := flag.Int("critic_max_tokens", 3000, "maximum tokens for the critic")

	poetModel := flag.String("poet_model", "gpt-4", "model to use for the creator")
	criticModel := flag.String("critic_model", "gpt-4", "model to use for the critic")

	genre := flag.String("genre", "haiku", "genre to use for the poem")
	theme := flag.String("theme", "squirrel falling from a tree", "theme to use for the poem")
	notes := flag.String("notes", "", "notes to use for the poem")

	needsMoreWorkThreshold := flag.Float64("needs_more_work_threshold", 0.0, "threshold for the critic to say that the work needs more work")

	flag.Parse()
	log.SetFormatter(&log.JSONFormatter{PrettyPrint: true})
	level, err := log.ParseLevel(*logLevel)
	if err != nil {
		log.Fatal(err)
	}

	log.SetLevel(level)
	client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))

	poet := agent.New(
		"poet",
		agent.WithClient(client),
		agent.WithModel(*poetModel),
		agent.WithTemperature(float32(*poetTemperature)),
		agent.WithMaxTokens(*poetMaxTokens))
	poet.System(poetSystem)
	critic := agent.New("critic",
		agent.WithClient(client),
		agent.WithModel(*criticModel),
		agent.WithTemperature(float32(*criticTemperature)),
		agent.WithMaxTokens(*criticMaxTokens))

	critic.System(fmt.Sprintf(criticSystem, *genre, *theme, *notes))

	poet.Listen(fmt.Sprintf(poetUser, *theme, *genre, *notes))

	poem, err := poet.RespondStream(context.Background(), os.Stdout)
	if err != nil {
		log.Fatal(err)
	}

	poetResponse := PoetResponse{}
	if err := json.Unmarshal([]byte(poem), &poetResponse); err != nil {
		log.Fatal(err)
	}

	needsMoreWork := 1.0

	for needsMoreWork > *needsMoreWorkThreshold {
		critic.Listen(fmt.Sprintf(criticUser, poetResponse.Text, poetResponse.Explanation, *notes))
		feedback, err := critic.Respond(context.Background())
		if err != nil {
			log.Fatal(err)
		}

		criticResponse := CriticResponse{}
		if err := json.Unmarshal([]byte(feedback), &criticResponse); err != nil {
			log.Fatal(err)
		}
		needsMoreWork = criticResponse.NeedsMoreWork

		poet.Listen(criticResponse.Feedback)
		poem, err = poet.Respond(context.Background())
		if err != nil {
			log.Fatal(err)
		}

		if err := json.Unmarshal([]byte(poem), &poetResponse); err != nil {
			log.Fatal(err)
		}
	}

}
