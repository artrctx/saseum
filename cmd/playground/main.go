package main

import (
	"saseum/internal/embed"
)

func main() {
	e, err := embed.New(embed.MiniLM)
	if err != nil {
		panic(err)
	}
	e.Generate("This is an example sentence\n\nEach sentence is converted")
}
