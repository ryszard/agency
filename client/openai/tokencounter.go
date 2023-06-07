package openai

import (
	"github.com/tiktoken-go/tokenizer"
)

func TokenCounter(model string) (func(string) (int, error), error) {
	codec, err := tokenizer.ForModel(tokenizer.Model(model))
	if err != nil {
		return nil, err
	}

	return func(s string) (int, error) {
		ids, _, err := codec.Encode(s)
		if err != nil {
			return 0, err
		}

		return len(ids), nil
	}, nil
}
