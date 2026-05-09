// https://github.com/gomlx/go-huggingface
// https://github.com/gomlx/onnx-gomlx
package embed

import (
	"os"

	"github.com/gomlx/go-huggingface/hub"
	"github.com/gomlx/go-huggingface/tokenizers"
	"github.com/gomlx/onnx-gomlx/onnx"
	onnxGomlx "github.com/gomlx/onnx-gomlx/onnx/parser"
)

// huggingface model id
type ModelCfg struct {
	id        string
	modelPath string
}

var (
	E5LargeV2 ModelCfg = ModelCfg{"intfloat/e5-large-v2", "onnx/model.onnx"}
	E5BaseV2  ModelCfg = ModelCfg{"intfloat/e5-base-v2", "onnx/model.onnx"}
)

// https://github.com/gomlx/onnx-gomlx
// https://github.com/gomlx/go-huggingface
type Client struct {
	encodingSize uint
	hub          *hub.Repo
	model        onnx.Model
	tokenizer    tokenizers.Tokenizer
}

// should have fixed embedidng isze
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

	return &Client{768, repo, model, tok}, nil
}

func (c *Client) WithEncodingSize(size uint) *Client {
	c.encodingSize = size
	return c
}

func (c *Client) CreateEmbedding(text string) {
	// sentences := []string{
	// 	"This is an example sentence",
	// 	"Each sentence is converted"}

	// for _, s := range sentences {
	// 	enc := tok.Encode(s)
	// 	fmt.Println(enc)
	// }
}
