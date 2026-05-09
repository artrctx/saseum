// https://github.com/gomlx/go-huggingface
// https://github.com/gomlx/onnx-gomlx
package embed

import (
	"fmt"
	"os"

	"github.com/gomlx/go-huggingface/hub"
)

// huggingface model id
type ModelCfg struct {
	id   string
	path string
}

var (
	E5LargeV2 ModelCfg = ModelCfg{"intfloat/e5-large-v2", "onnx/model.onnx"}
	E5BaseV2  ModelCfg = ModelCfg{"intfloat/e5-base-v2", "onnx/model.onnx"}
)

// https://github.com/gomlx/onnx-gomlx
// https://github.com/gomlx/go-huggingface
type Client struct {
	modelPath string
}

// provide supported model id declared here
// this model needs to support onnx
func New(cfg ModelCfg) (*Client, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	repo := hub.New(cfg.id).WithCacheDir(cwd + "/models").WithAuth(os.Getenv("HF_TOKEN"))
	modelPath, err := repo.DownloadFile(cfg.path)
	if err != nil {
		return nil, err
	}
	fmt.Println("Model path", modelPath)

	// Parse ONNX model.
	return nil, nil
}
