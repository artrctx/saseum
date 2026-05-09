package main

import "saseum/internal/embed"

func main() {
	_, err := embed.New(embed.E5BaseV2)
	if err != nil {
		panic(err)
	}
}
