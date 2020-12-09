package main

import (
	"log"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256cmd"
)

func main() {
	inet256cmd.Register(0, "onehop", inet256.OneHopFactory)
	if err := inet256cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
