package main

import (
	"fmt"
	"saseum/internal/embed"
)

func main() {
	e, err := embed.New(embed.AllMiniLM, 5)
	if err != nil {
		panic(err)
	}
	r := <-e.Queue("This is an example sentence\n\nEach sentence is converted")

	if r.Error != nil {
		panic(r.Error)
	}

	fmt.Println(len(r.Data), len(r.Data[0]))
}
