package main

import "saseum/internal/embed"

func main() {
	_, err := embed.New()
	if err != nil {
		panic(err)
	}
}
