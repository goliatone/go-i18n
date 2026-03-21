package main

import (
	"log"

	"github.com/goliatone/go-i18n/examples/basic"
)

func main() {
	if err := basic.Run(); err != nil {
		log.Fatalf("example: %v", err)
	}
}
