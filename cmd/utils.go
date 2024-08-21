package cmd

import (
	"log"
	"os"
)

func handleError(err error) {
	if err != nil {
		log.Fatalf("%v", err.Error())
		os.Exit(1)
	}
}
