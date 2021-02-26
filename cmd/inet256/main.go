package main

import (
	"log"

	"github.com/inet256/inet256/pkg/inet256cmd"
	"github.com/inet256/inet256/pkg/inet256d"
	"github.com/inet256/inet256/pkg/kadsrnet"
)

func main() {
	inet256d.Register(inet256d.IndexFromString("kad+sr"), "kad+sr", kadsrnet.Factory)
	if err := inet256cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
