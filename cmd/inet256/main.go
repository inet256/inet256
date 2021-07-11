package main

import (
	"log"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256cmd"
	"github.com/inet256/inet256/pkg/inet256d"
)

func main() {
	// inet256d.Register(inet256d.IndexFromString("kad+sr"), "kad+sr", kadsrnet.Factory)
	inet256d.Register(inet256d.IndexFromString("onehop"), "onehop", inet256.OneHopFactory)
	if err := inet256cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
