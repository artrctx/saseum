// https://github.com/gomlx/go-huggingface
// https://github.com/gomlx/onnx-gomlx
package embed

import (
	"fmt"
	"os"

	"github.com/gomlx/go-huggingface/hub"
	"github.com/gomlx/go-huggingface/tokenizers"
	"github.com/gomlx/go-huggingface/tokenizers/api"
	"github.com/gomlx/onnx-gomlx/onnx"
	onnxGomlx "github.com/gomlx/onnx-gomlx/onnx/parser"
)

// huggingface model id
type ModelCfg struct {
	id        string
	modelPath string
	dim       int
}

// all dynamic input no need to pad chunked tokens
var (
	MiniLM    ModelCfg = ModelCfg{"sentence-transformers/all-MiniLM-L6-v2", "onnx/model.onnx", 384}
	E5LargeV2 ModelCfg = ModelCfg{"intfloat/e5-large-v2", "onnx/model.onnx", 1024}
	E5BaseV2  ModelCfg = ModelCfg{"intfloat/e5-base-v2", "onnx/model.onnx", 768}
)

type metadata struct {
	outputDim   int
	maxTokenLen float64
	padID       int
	beginID     int
	endID       int
}

// https://github.com/gomlx/onnx-gomlx
// https://github.com/gomlx/go-huggingface
type Client struct {
	hub       *hub.Repo
	model     onnx.Model
	tokenizer tokenizers.Tokenizer
	meta      metadata
}

// TODO: if files downloaded i should be able to skip downloading tokenizer.json and model
// ! Currently it still requires internet connection to verify all files are downloaded
// provide supported model id declared here
// this model needs to support onnx
func New(cfg ModelCfg) (*Client, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	repo := hub.New(cfg.id).WithCacheDir(cwd + "/models").WithAuth(os.Getenv("HF_TOKEN"))
	modelPath, err := repo.DownloadFile(cfg.modelPath)
	if err != nil {
		return nil, err
	}

	model, err := onnxGomlx.ParseFile(modelPath)
	if err != nil {
		return nil, err
	}

	tok, err := tokenizers.New(repo)
	if err != nil {
		model.Close()
		return nil, err
	}
	// Get Metadata Tokens
	padID, err := tok.SpecialTokenID(api.TokPad)
	if err != nil {
		model.Close()
		return nil, err
	}
	beginID, err := tok.SpecialTokenID(api.TokBeginningOfSentence)
	if err != nil {
		model.Close()
		return nil, err
	}
	endID, err := tok.SpecialTokenID(api.TokEndOfSentence)
	if err != nil {
		model.Close()
		return nil, err
	}

	return &Client{repo, model, tok, metadata{
		outputDim:   cfg.dim,
		maxTokenLen: tok.Config().ModelMaxLength,
		padID:       padID,
		beginID:     beginID,
		endID:       endID,
	}}, nil
}

func (c *Client) Close() error {
	return c.model.Close()
}

func (c *Client) Dim() int {
	return c.meta.outputDim
}

func (c *Client) Shape() []int {
	_, shapes := c.model.Outputs()
	return shapes[0].Dimensions
}

// overlap .1 or .2 of token
func (c *Client) GenerateEmbedding(text string) {
	chunks, err := chunkWithOverlap(c.tokenizer.Encode(text), int(c.meta.maxTokenLen), int(c.meta.maxTokenLen*0.1), c.meta)
	fmt.Println(chunks, err)
}

// add guard
func chunkWithOverlap(set []int, chunkSize, overlap int, meta metadata) ([][]int, error) {
	if chunkSize <= overlap {
		return nil, fmt.Errorf("chunk size cannot be equal or smaller than overlap received chunk size = %d, overlap = %d", chunkSize, overlap)
	}

	// removing begin and end token id to be chunked and added back
	fTkn, lTkn := set[0], set[len(set)-1]
	if fTkn == meta.beginID && lTkn == meta.endID {
		set = set[1 : len(set)-1]
	}

	maxLen, modChunkSize := len(set), chunkSize-overlap-2
	steps := (maxLen / modChunkSize) + 1

	chunked := make([][]int, 0, steps)
	for step := range steps {
		start := step * modChunkSize
		end := start + modChunkSize + overlap

		if end > maxLen {
			end = maxLen
		}

		chunk := set[start:end]
		chunk = append([]int{meta.beginID}, set[start:end]...)
		chunk = append(chunk, meta.endID)

		chunked = append(chunked, chunk)
	}

	lastChunkedLen := len(chunked[len(chunked)-1])
	if lastChunkedLen == chunkSize {
		return chunked, nil
	}

	// padding last chunk
	for range chunkSize - lastChunkedLen {
		chunked[len(chunked)-1] = append(chunked[len(chunked)-1], meta.padID)
	}

	return chunked, nil
}
