package main

import (
	"log"

	"github.com/inet256/inet256/networks/beaconnet"
	"github.com/inet256/inet256/networks/floodnet"
	"github.com/inet256/inet256/pkg/inet256cmd"
	"github.com/inet256/inet256/pkg/inet256d"
	"github.com/inet256/inet256/pkg/inet256srv"
)

func main() {
	inet256d.Register("onehop", inet256d.IndexFromString("onehop"), inet256srv.OneHopFactory)
	inet256d.Register("floodnet", inet256d.IndexFromString("floodnet"), floodnet.Factory)
	inet256d.Register("beaconnet", inet256d.IndexFromString("beacon0"), beaconnet.Factory)

	if err := inet256cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
