package main

import (
	"log"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256cmd"
	"github.com/inet256/inet256/pkg/kadsrnet"
)

func main() {
	inet256cmd.Register(inet256cmd.IndexFromString("onehop"), "onehop", inet256.OneHopFactory)
	inet256cmd.Register(inet256cmd.IndexFromString("kad+sr"), "kad+sr", kadsrnet.Factory)
	if err := inet256cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
