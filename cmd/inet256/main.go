package main

import (
	"log"

	"github.com/inet256/inet256/pkg/inet256cmd"
)

func main() {
	if err := inet256cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
