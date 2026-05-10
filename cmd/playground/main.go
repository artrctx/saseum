package main

import (
	"fmt"
	"saseum/internal/embed"
)

func main() {
	e, err := embed.New(embed.MiniLM)
	if err != nil {
		panic(err)
	}
	val, err := e.Generate("This is an example sentence\n\nEach sentence is converted")
	if err != nil {
		panic(err)
	}
	fmt.Println(val)
}
