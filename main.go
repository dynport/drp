package main

import (
	"log"
	"os"

	"github.com/dynport/drp/drp"
)

var logger = log.New(os.Stderr, "", 0)

func main() {
	if err := run(); err != nil {
		logger.Fatal(err)
	}
}

func run() error {
	return drp.Run()
}
