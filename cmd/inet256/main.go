package main

import (
	"log"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256cmd"
	"github.com/inet256/inet256/pkg/onehop"
)

func main() {
	inet256cmd.Register(inet256.NetworkSpec{
		Name:    "onehop",
		Factory: onehop.Factory,
	})
	if err := inet256cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
