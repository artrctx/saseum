// https://github.com/gomlx/go-huggingface
// https://github.com/gomlx/onnx-gomlx
package embed

import (
	"fmt"
	"os"

	"github.com/gomlx/go-huggingface/hub"
)

// huggingface model id
type ModelID = string

const (
	E5LargeV2 ModelID = "intfloat/e5-large-v2"
	E5BaseV2  ModelID = "intfloat/e5-base-v2"
)

type Client struct {
}

// https://www.datarobot.com/blog/choosing-the-right-vector-embedding-model-for-your-generative-ai-use-case/
// TODO: support user provided model
func New() (*Client, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	repo := hub.New(E5BaseV2).WithCacheDir(cwd + "/models").WithAuth(os.Getenv("HF_TOKEN"))
	modelPath, err := repo.DownloadFile("onnx/model.onnx")
	if err != nil {
		return nil, err
	}
	fmt.Println("Model path", modelPath)

	// Parse ONNX model.
	return nil, nil
}
