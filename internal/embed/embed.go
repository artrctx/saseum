// https://github.com/gomlx/go-huggingface
// https://github.com/gomlx/onnx-gomlx
package embed

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/gomlx/compute"
	"github.com/gomlx/go-huggingface/hub"
	"github.com/gomlx/go-huggingface/tokenizers"
	"github.com/gomlx/go-huggingface/tokenizers/api"
	"github.com/gomlx/gomlx/pkg/core/graph"
	"github.com/gomlx/gomlx/pkg/ml/context"
	"github.com/gomlx/onnx-gomlx/onnx"
	onnxGomlx "github.com/gomlx/onnx-gomlx/onnx/parser"
	"golang.org/x/sync/errgroup"
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
	outputDim    int
	maxTokenLen  float32
	padID        int
	beginID      int
	endID        int
	chunkOverlap float32
}

// https://github.com/gomlx/onnx-gomlx
// https://github.com/gomlx/go-huggingface
type Client struct {
	hub       *hub.Repo
	model     onnx.Model
	backend   *context.Exec
	tokenizer tokenizers.Tokenizer
	meta      metadata
}

// TODO: if files downloaded i should be able to skip downloading tokenizer.json and model
// ! Currently it still requires internet connection to verify all files are downloaded
// provide supported model id declared here
// this model needs to support onnx
// set backend to use with GOMLX_BACKEND env
func New(cfg ModelCfg) (*Client, error) {

	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	repo := hub.New(cfg.id).WithCacheDir(cwd + "/models").WithAuth(os.Getenv("HF_TOKEN"))
	//prep tokenizer
	tok, err := tokenizers.New(repo)
	if err != nil {
		return nil, err
	}
	// Get Metadata Tokens
	padID, err := tok.SpecialTokenID(api.TokPad)
	if err != nil {
		return nil, err
	}
	beginID, err := tok.SpecialTokenID(api.TokBeginningOfSentence)
	if err != nil {
		return nil, err
	}
	endID, err := tok.SpecialTokenID(api.TokEndOfSentence)
	if err != nil {
		return nil, err
	}

	// prepare compute backend
	backend, err := compute.New()
	if err != nil {
		return nil, err
	}

	//prep models
	modelPath, err := repo.DownloadFile(cfg.modelPath)
	if err != nil {
		return nil, err
	}

	model, err := onnxGomlx.ParseFile(modelPath)
	if err != nil {
		return nil, err
	}

	// update this
	exec, err := context.NewExec(backend, context.New(), func(ctx *context.Context, inputIds, attentionMask *graph.Node) []*graph.Node {
		inputs := map[string]*graph.Node{
			"input_ids":      inputIds,
			"attention_mask": attentionMask,
		}

		return model.CallGraph(ctx, inputIds.Graph(), inputs)
	})
	if err != nil {
		model.Close()
		return nil, err
	}

	return &Client{repo, model, exec, tok, metadata{
		outputDim:    cfg.dim,
		maxTokenLen:  float32(tok.Config().ModelMaxLength),
		padID:        padID,
		beginID:      beginID,
		endID:        endID,
		chunkOverlap: 0.1,
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
// generates embedding
func (c *Client) Generate(text string) (string, error) {
	_, err := c.encodeChunked(text)
	if err != nil {
		return "", err
	}

	return "", nil
}

func (c *Client) encodeChunked(text string) ([][]int, error) {
	sChunks := semanticChunks(text)

	g := errgroup.Group{}
	chunkChan := make(chan [][]int, len(sChunks))
	for _, s := range sChunks {
		// cs, err := chunkWithOverlap(c.tokenizer.Encode(s), int(c.meta.maxTokenLen), int(c.meta.maxTokenLen*0.1), c.meta)
		g.Go(func() error {
			chunks, err := chunkWithOverlap(c.tokenizer.Encode(s), int(c.meta.maxTokenLen), int(c.meta.maxTokenLen*c.meta.chunkOverlap), c.meta.beginID, c.meta.endID, c.meta.padID)
			if err != nil {
				return err
			}
			chunkChan <- chunks
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}
	close(chunkChan)

	var chunks [][]int
	for c := range chunkChan {
		chunks = append(chunks, c...)
	}
	return chunks, nil
}

// real token  -> 1
// padding     -> 0
func buildAttentionMask(ids []int, padID int) []int {
	masks := make([]int, len(ids))
	for idx, id := range ids {
		if id == padID {
			continue
		}
		masks[idx] = 1
	}
	return masks
}

// replaces \r\n -> \n and \n{3,} (over 3 lines of space) -> \n\n and split to semantic chunks by double new line
func semanticChunks(s string) []string {
	splitStr := strings.Split(regexp.MustCompile(`\n{3,}`).ReplaceAllString(strings.ReplaceAll(strings.TrimSpace(s), "\r\n", "\n"), "\n\n"), "\n\n")
	strs := make([]string, 0, len(splitStr))
	for _, s := range splitStr {
		strs = append(strs, strings.TrimSpace(s))
	}
	return strs
}

// add guard
func chunkWithOverlap(set []int, chunkSize, overlap int, beginID, endID, padID int) ([][]int, error) {
	if chunkSize <= overlap {
		return nil, fmt.Errorf("chunk size cannot be equal or smaller than overlap received chunk size = %d, overlap = %d", chunkSize, overlap)
	}

	// removing begin and end token id to be chunked and added back
	fTkn, lTkn := set[0], set[len(set)-1]
	if fTkn == beginID && lTkn == endID {
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
		chunk = append([]int{beginID}, set[start:end]...)
		chunk = append(chunk, endID)

		chunked = append(chunked, chunk)
	}

	lastChunkedLen := len(chunked[len(chunked)-1])
	if lastChunkedLen == chunkSize {
		return chunked, nil
	}

	// padding last chunk
	for range chunkSize - lastChunkedLen {
		chunked[len(chunked)-1] = append(chunked[len(chunked)-1], padID)
	}

	return chunked, nil
}
