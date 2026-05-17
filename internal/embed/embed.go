// https://github.com/gomlx/go-huggingface
// https://github.com/gomlx/onnx-gomlx
package embed

import (
	"fmt"
	"math"
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
type ModelID struct {
	ID        string
	modelPath string
	dim       int
}

// all dynamic input no need to pad chunked tokens
var (
	// inputs : [input_ids attention_mask token_type_ids] | model.Inputs() |
	// ouputs : [last_hidden_state] | model.Outputs() | [-1 -1, 384]
	AllMiniLM ModelID = ModelID{"sentence-transformers/all-MiniLM-L6-v2", "onnx/model.onnx", 384}
	// inputs : [input_ids attention_mask token_type_ids]
	// ouputs : [last_hidden_state] | model.Outputs() | [-1 -1, 768]
	E5LargeV2 ModelID = ModelID{"intfloat/e5-large-v2", "onnx/model.onnx", 1024}
	// inputs : [input_ids attention_mask token_type_ids]
	// ouputs : [last_hidden_state] | model.Outputs() | [-1 -1, 1024]
	E5BaseV2 ModelID = ModelID{"intfloat/e5-base-v2", "onnx/model.onnx", 768}
)

type metadata struct {
	outputDim    int
	maxTokenLen  float32
	padID        int
	beginID      int
	endID        int
	chunkOverlap float32
}

type Result struct {
	Data  [][]float32
	Error error
}
type Task struct {
	input  string
	result chan Result
}

func inferenceWorker(tokenizer tokenizers.Tokenizer, meta metadata, backend compute.Backend, model onnx.Model, execCtx *context.Context, tasks <-chan Task) error {
	exec, err := context.NewExec(backend, execCtx, func(ctx *context.Context, inputIds, attentionMask, tokenTypeIds *graph.Node) []*graph.Node {
		return model.CallGraph(ctx, inputIds.Graph(), map[string]*graph.Node{
			"input_ids":      inputIds,
			"attention_mask": attentionMask,
			"token_type_ids": tokenTypeIds,
		})
	})
	if err != nil {
		return err
	}

	go func() {
		defer exec.Finalize()
		for task := range tasks {
			chunks, err := encodeChunked(task.input, tokenizer, meta)
			if err != nil {
				task.result <- Result{Error: err}
				continue
			}

			// current expecting only 1 output to be returned
			masks := buildAttentionMasks(chunks, meta.padID)
			tensor, err := exec.Exec1(chunks, masks, buildTokenTypeIds(len(chunks), len(chunks[0])))
			if err != nil {
				task.result <- Result{Error: err}
				continue
			}

			tknEmbeddings, ok := tensor.Value().([][][]float32)
			if !ok {
				task.result <- Result{Error: fmt.Errorf("failed to cast model output to [][][]float32")}
				continue
			}

			embeddings := make([][]float32, len(tknEmbeddings))
			for idx, tknE := range tknEmbeddings {
				embeddings[idx] = normalize(meanPool(tknE, masks[idx]))
			}

			task.result <- Result{Data: embeddings}
		}
	}()

	return nil
}

// https://github.com/gomlx/onnx-gomlx
// https://github.com/gomlx/go-huggingface
type Embedder struct {
	hub       *hub.Repo
	model     onnx.Model
	tokenizer tokenizers.Tokenizer
	meta      metadata
	queue     chan Task
}

// TODO: if files downloaded i should be able to skip downloading tokenizer.json and model
// ! Currently it still requires internet connection to verify all files are downloaded
// provide supported model id declared here
// this model needs to support onnx
// set backend to use with GOMLX_BACKEND env
func New(cfg ModelID, workerCount uint8) (*Embedder, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	repo := hub.New(cfg.ID).WithCacheDir(cwd + "/models").WithAuth(os.Getenv("HF_TOKEN"))
	// ----------- TOKENIZER -----------
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

	// ----------- MODEL & EXECUTOR -----------
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
	ctx := context.New()
	if err := model.VariablesToContext(ctx); err != nil {
		model.Close()
		return nil, err
	}
	metadata := metadata{
		outputDim:    cfg.dim,
		maxTokenLen:  float32(tok.Config().ModelMaxLength),
		padID:        padID,
		beginID:      beginID,
		endID:        endID,
		chunkOverlap: 0.1,
	}

	if workerCount == 0 {
		// default 5 workers
		workerCount = 5
	}
	taskQueues := make(chan Task, 100)
	eg := errgroup.Group{}
	for range workerCount {
		eg.Go(func() error {
			return inferenceWorker(tok, metadata, backend, model, ctx, taskQueues)
		})
	}

	if err := eg.Wait(); err != nil {
		model.Close()
		return nil, err
	}

	return &Embedder{repo, model, tok, metadata, taskQueues}, nil
}

func (e *Embedder) Close() error {
	close(e.queue)
	return e.model.Close()
}

func (e *Embedder) Dim() int {
	return e.meta.outputDim
}

func (e *Embedder) Queue(text string) <-chan Result {
	t := Task{input: text, result: make(chan Result)}
	e.queue <- t
	return t.result
}

func encodeChunked(text string, tokenizer tokenizers.Tokenizer, meta metadata) ([][]int, error) {
	sChunks := semanticChunks(text)

	g := errgroup.Group{}
	chunkChan := make(chan [][]int, len(sChunks))
	for _, s := range sChunks {
		g.Go(func() error {
			chunks, err := chunkWithOverlap(tokenizer.Encode(s), int(meta.maxTokenLen), int(meta.maxTokenLen*meta.chunkOverlap), meta.beginID, meta.endID, meta.padID)
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

// real token -> 1 || padding -> 0
func buildAttentionMasks(ids [][]int, padID int) [][]int {
	idsLen, maskLen := len(ids), len(ids[0])
	maskSet := make([][]int, 0, idsLen)
	for idx := range idsLen {
		masks := make([]int, maskLen)

		for idx, id := range ids[idx] {
			if id == padID {
				continue
			}

			masks[idx] = 1
		}

		maskSet = append(maskSet, masks)
	}
	return maskSet
}

// zeroed out slice
func buildTokenTypeIds(count, size int) [][]int {
	set := make([][]int, 0, count)
	for range count {
		set = append(set, make([]int, size))
	}
	return set
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
	steps := int(math.Ceil(float64(maxLen) / float64(modChunkSize)))

	// 100 12 2
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

func meanPool(tknEmbeddings [][]float32, mask []int) []float32 {
	hiddenSize := len(tknEmbeddings[0])
	pooled := make([]float32, hiddenSize)

	var count float32
	for tIdx, token := range tknEmbeddings {
		if mask[tIdx] == 0 {
			continue
		}

		for idx, v := range token {
			pooled[idx] += v
		}

		count++
	}

	for i := range hiddenSize {
		pooled[i] /= count
	}

	return pooled
}

// L2 Normalization | Unit Vector Normalization
func normalize(sets []float32) []float32 {
	var sum float64
	for _, x := range sets {
		fx := float64(x)
		sum += fx * fx
	}

	norm := float32(math.Sqrt(sum))

	for i := range sets {
		sets[i] /= norm
	}

	return sets
}
