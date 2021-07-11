package main

import (
	"log"

	"github.com/inet256/inet256/networks/floodnet"
	"github.com/inet256/inet256/pkg/inet256cmd"
	"github.com/inet256/inet256/pkg/inet256d"
	"github.com/inet256/inet256/pkg/inet256srv"
)

func main() {
	// inet256d.Register(inet256d.IndexFromString("kad+sr"), "kad+sr", kadsrnet.Factory)
	inet256d.Register(inet256d.IndexFromString("onehop"), "onehop", inet256srv.OneHopFactory)
	inet256d.Register(inet256d.IndexFromString("flood"), "flood", floodnet.Factory)
	if err := inet256cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
