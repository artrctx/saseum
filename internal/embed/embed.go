// https://github.com/tmc/langchaingo/tree/main
package embed

import "github.com/tmc/langchaingo/llms/openai"

type EmbeddingModel = string

const (
	TextEmbedding3Large EmbeddingModel = "text-embedding-3-large"
	TextEmbedding3Small EmbeddingModel = "text-embedding-3-small"
)

type Client struct {
	llm *openai.LLM
}

func New() (*Client, error) {
	llm, err := openai.New(openai.WithEmbeddingModel(TextEmbedding3Large))
	if err != nil {
		return nil, err
	}

	return &Client{llm}, nil
}
