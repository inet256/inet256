package main

import (
	"log"

	"go.inet256.org/inet256/src/inet256cmd"
)

func main() {
	if err := inet256cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
