package anthropic

import (
	_ "embed"

	"github.com/daulet/tokenizers"
	log "github.com/sirupsen/logrus"
)

//go:embed claude-v1-tokenization.json
var tokenizerConfig []byte

func TokenCounter(s string) (int, error) {
	log.SetLevel(log.DebugLevel)
	tk, err := tokenizers.FromBytes(tokenizerConfig)
	if err != nil {
		return 0, err
	}
	defer tk.Close()

	tokens := tk.Encode(s, true)

	return len(tokens), nil

}
