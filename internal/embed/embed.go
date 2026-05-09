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

// https://github.com/gomlx/onnx-gomlx
// https://github.com/gomlx/go-huggingface
type Client struct {
	cfg       ModelCfg
	tokCfg    *api.Config
	hub       *hub.Repo
	model     onnx.Model
	tokenizer tokenizers.Tokenizer
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
		return nil, err
	}

	return &Client{cfg, tok.Config(), repo, model, tok}, nil
}

func (c *Client) Dim() int {
	return c.cfg.dim
}

func (c *Client) Shape() []int {
	_, shapes := c.model.Outputs()
	return shapes[0].Dimensions
}

// overlap .1 or .2 of token
func (c *Client) GenerateEmbedding(text string) {
	// _ = chunkWithOverlap(c.tokenizer.Encode(text), int(c.tokCfg.ModelMaxLength), int(c.tokCfg.ModelMaxLength*0.1))

	set := make([]int, 0, 100)
	for i := range 100 {
		set = append(set, i)
	}

	fmt.Println(chunkWithOverlap(set, 10, 2))
}

// add guard
func chunkWithOverlap(set []int, chunkSize, overlap int) [][]int {
	maxLen, modChunkSize := len(set), chunkSize-overlap
	steps := (maxLen / modChunkSize) + 1

	chunked := make([][]int, 0, steps)
	for step := range steps {
		start := step * modChunkSize
		end := start + modChunkSize + overlap

		if end > maxLen {
			end = maxLen
		}

		chunked = append(chunked, set[start:end])
	}

	lastChunkedLen := len(chunked[len(chunked)-1])
	if lastChunkedLen < chunkSize {
		// i should pad with actual pad id value
		chunked[len(chunked)-1] = append(chunked[len(chunked)-1], make([]int, chunkSize-lastChunkedLen)...)
	}

	return chunked
}
