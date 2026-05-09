package main

import (
	"saseum/internal/embed"
)

func main() {
	e, err := embed.New(embed.E5BaseV2)
	if err != nil {
		panic(err)
	}
	e.GenerateEmbedding("THIS IS TEST")
}
