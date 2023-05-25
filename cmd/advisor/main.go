package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ryszard/agency/agent"
	"github.com/ryszard/agency/util/cache"
	"github.com/sashabaranov/go-openai"
	log "github.com/sirupsen/logrus"
)

var (
	config      = flag.String("config", "", "path to config file")
	messagePath = flag.String("message", "", "message from the user")
	logLevel    = flag.String("log_level", "debug", "log level")

	cachePath              = flag.String("cache", "", "path to cache file")
	includeCallingLocation = flag.Bool("include_calling_location", false, "include calling location in log messages")
)

func main() {
	flag.Parse()

	level, err := log.ParseLevel(*logLevel)
	if err != nil {
		log.Fatal(err)
	}

	log.SetLevel(level)

	if *includeCallingLocation {
		log.SetReportCaller(true)
	}

	log.Info("Starting advisor")

	client := agent.Retrying(openai.NewClient(os.Getenv("OPENAI_API_KEY")), time.Second, 60*time.Second, 5)

	var cach *cache.BoltDBCache = nil
	if *cachePath != "" {
		cach, err = cache.BoltDB(*cachePath)
		if err != nil {
			log.WithError(err).Fatal("can't create cache")
		}
	}
	flow, err := Load(*config, client, os.Stdout, cach)
	if err != nil {
		log.WithError(err).Fatal("can't load config file")
	}

	// load message from messagePath
	messageBytes, err := os.ReadFile(*messagePath)
	if err != nil {
		log.WithError(err).Fatal("can't load message file")
	}

	message := string(messageBytes)

	flow.IntentionEvaluator.Listen("emotional_state", message)

	emotionalState, err := flow.IntentionEvaluator.RespondStream(context.Background())
	if err != nil {
		log.WithError(err).Fatal("can't respond")
	}

	flow.IntentionEvaluator.Listen("desired_response", nil)
	desiredResponse, err := flow.IntentionEvaluator.RespondStream(context.Background())
	if err != nil {
		log.WithError(err).Fatal("can't respond")
	}

	flow.IntentionEvaluator.Listen("criteria", desiredResponse)
	criteria, err := flow.IntentionEvaluator.RespondStream(context.Background())
	if err != nil {
		log.WithError(err).Fatal("can't respond")
	}

	fmt.Fprintf(os.Stdout, "IntentionEvaluator:\n\nEmotional State: %s\n\nDesired Response: %s\n\nCriteria: %s\n\n", emotionalState, desiredResponse, criteria)
	log.SetLevel(log.TraceLevel)
	log.WithField("author messages", flow.Author.Messages()).Debug("author messages before Listen")

	if err := flow.Author.Listen("reply", struct {
		EmotionalState  string
		DesiredResponse string
		Criteria        string
		Message         string
	}{emotionalState, desiredResponse, criteria, message}); err != nil {
		log.WithError(err).Fatal("can't listen")
	}

	log.WithField("author messages", flow.Author.Messages()).Debug("author messages after Listen")

	reply, err := flow.Author.RespondStream(context.Background())
	if err != nil {
		log.WithError(err).Fatal("can't respond")
	}

	fmt.Fprintf(os.Stdout, "Author:\n\nReply: %s\n\n", reply)

}
