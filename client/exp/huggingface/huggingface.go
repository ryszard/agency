package huggingface

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/ryszard/agency/client"
	log "github.com/sirupsen/logrus"
)

type Client struct {
	token   string
	http    *http.Client
	baseURL string
}

func (Client) SupportsStreaming() bool {
	return false
}

func New(token string) *Client {
	return &Client{
		token:   token,
		http:    &http.Client{},
		baseURL: "https://api-inference.huggingface.co/models/",
	}
}

var _ client.Client = (*Client)(nil)

type Options struct {
	UseCache     bool `json:"use_cache,omitempty"`
	WaitForModel bool `json:"wait_for_model,omitempty"`
}
type Parameters struct {
	MinLength         int     `json:"min_length,omitempty"`
	MaxLength         int     `json:"max_length,omitempty"`
	TopK              int     `json:"top_k,omitempty"`
	TopP              float64 `json:"top_p,omitempty"`
	Temperature       float64 `json:"temperature,omitempty"`
	RepetitionPenalty float64 `json:"repetition_penalty,omitempty"`
	MaxTime           float64 `json:"max_time,omitempty"`
}

type ConversationalRequest struct {
	Model              string     `json:"-"`
	Text               string     `json:"text,omitempty"`
	PastUserInputs     []string   `json:"past_user_inputs,omitempty"`
	GeneratedResponses []string   `json:"generated_responses,omitempty"`
	Options            Options    `json:"options,omitempty"`
	Parameters         Parameters `json:"parameters,omitempty"`
}

type ConversationalResponse struct {
	GeneratedText string `json:"generated_text,omitempty"`
	Error         string `json:"error,omitempty"`
}

func (cl *Client) CreateChatCompletion(ctx context.Context, request client.ChatCompletionRequest) (client.ChatCompletionResponse, error) {
	log.WithField("request", request).Debug("huggingface client CreateChatCompletion")
	if request.WantsStreaming() {
		log.Warn("huggingface client does not support streaming")
	}
	payload, err := TranslateRequest(request)
	if err != nil {
		return client.ChatCompletionResponse{}, err
	}
	log.WithField("payload", fmt.Sprintf("%#v", payload)).Debug("huggingface request")

	var resp ConversationalResponse
	if err := cl.MakeRequest(ctx, cl.baseURL+payload.Model, payload, &resp); err != nil {

		return client.ChatCompletionResponse{}, err
	}
	log.WithField("resp", resp).Debug("huggingface response")
	request.Stream.Write([]byte(resp.GeneratedText + "\n"))
	return TranslateResponse(resp)
}

func (cl *Client) MakeRequest(ctx context.Context, urlStr string, payload any, out any) error {
	// The Hugging Face API expects to receive a JSON string containing the JSON
	// encoded body. Weird, but what can you do.
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	data, err = json.Marshal(string(data))
	if err != nil {
		return err
	}
	log.WithField("data", string(data)).Debug("huggingface request body")
	req, err := http.NewRequestWithContext(ctx, "POST", urlStr, bytes.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cl.token)

	resp, err := cl.http.Do(req)
	if err != nil {
		return err
	}
	log.WithField("status", resp.Status).WithField("status code", resp.StatusCode).Debug("huggingface response")
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	log.WithField("respBody", string(respBody)).Debug("huggingface response body")
	if err := json.Unmarshal(respBody, &out); err != nil {
		return err
	}
	return nil
}

func TranslateResponse(cr ConversationalResponse) (client.ChatCompletionResponse, error) {
	if cr.Error != "" {
		return client.ChatCompletionResponse{}, errors.New(cr.Error)
	}
	msg := client.Message{
		Content: cr.GeneratedText,
		Role:    client.Assistant,
	}

	ccr := client.ChatCompletionResponse{
		Choices: []client.Message{msg},
	}

	return ccr, nil
}

// reservedParams is the list of parameters that cannot be part of the custom
// params.
var reservedParams = []string{"max_length", "temperature"}

func TranslateMessages(messages []client.Message, req *ConversationalRequest) error {
	if len(messages) == 0 {
		return errors.New("no messages")
	}
	tail := messages[len(messages)-1]
	if tail.Role == client.Assistant {
		return fmt.Errorf("last message must be from user")
	}
	req.Text = tail.Content

	similar := func(left, right client.Role) bool {
		return left == right || (left == client.User && right == client.System) || (left == client.System && right == client.User)
	}

	lastRole := client.Role("")
	var current []string
	for _, msg := range messages {

		if msg.Role == "" {
			return fmt.Errorf("message role is empty: %#v", msg)
		}

		// Adjacent messages from the same role are concatenated.
		if similar(msg.Role, lastRole) {
			current = append(current, msg.Content)
			continue
		}

		// A change in role. Join current messages and add to the request.
		if len(current) > 0 {
			content := strings.Join(current, "\n\n")
			if !similar(msg.Role, client.User) {
				req.PastUserInputs = append(req.PastUserInputs, content)
			} else {
				req.GeneratedResponses = append(req.GeneratedResponses, content)
			}
		}
		current = []string{msg.Content}
		lastRole = msg.Role

	}
	// Note: we are not adding the last message to the request. It is used to
	// initialize the request text.
	return nil
}

func TranslateRequest(clientReq client.ChatCompletionRequest) (ConversationalRequest, error) {
	for _, param := range reservedParams {
		if _, ok := clientReq.CustomParams[param]; ok {
			return ConversationalRequest{}, fmt.Errorf("custom param %q is reserved", param)
		}
	}
	req := ConversationalRequest{
		Model: clientReq.Model,
		Parameters: Parameters{
			MaxLength:   clientReq.MaxTokens,
			Temperature: float64(clientReq.Temperature),
		},
	}

	if err := TranslateMessages(clientReq.Messages, &req); err != nil {
		return ConversationalRequest{}, err
	}

	if useCache, ok := clientReq.CustomParams["use_cache"]; ok {
		req.Options.UseCache, ok = useCache.(bool)
		if !ok {
			return ConversationalRequest{}, fmt.Errorf("use_cache must be a bool")
		}
	}

	if waitForModel, ok := clientReq.CustomParams["wait_for_model"]; ok {
		req.Options.WaitForModel, ok = waitForModel.(bool)
		if !ok {
			return ConversationalRequest{}, fmt.Errorf("wait_for_model must be a bool")
		}
	}

	if minLen, ok := clientReq.CustomParams["min_length"]; ok {
		req.Parameters.MinLength, ok = minLen.(int)
		if !ok {
			return ConversationalRequest{}, fmt.Errorf("min_length must be an int")
		}
	}

	if maxLen, ok := clientReq.CustomParams["max_length"]; ok {
		req.Parameters.MaxLength, ok = maxLen.(int)
		if !ok {
			return ConversationalRequest{}, fmt.Errorf("max_length must be an int")
		}
	}

	if topK, ok := clientReq.CustomParams["top_k"]; ok {
		req.Parameters.TopK, ok = topK.(int)
		if !ok {
			return ConversationalRequest{}, fmt.Errorf("top_k must be an int")
		}
	}

	if topP, ok := clientReq.CustomParams["top_p"]; ok {
		req.Parameters.TopP, ok = topP.(float64)
		if !ok {
			return ConversationalRequest{}, fmt.Errorf("top_p must be a float64")
		}
	}

	if repetitionPenalty, ok := clientReq.CustomParams["repetition_penalty"]; ok {
		req.Parameters.RepetitionPenalty, ok = repetitionPenalty.(float64)
		if !ok {
			return ConversationalRequest{}, fmt.Errorf("repetition_penalty must be a float64")
		}
	}

	if maxTime, ok := clientReq.CustomParams["max_time"]; ok {
		req.Parameters.MaxTime, ok = maxTime.(float64)
		if !ok {
			return ConversationalRequest{}, fmt.Errorf("max_time must be a float64")
		}
	}

	return req, nil
}
