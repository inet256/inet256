package main

import (
	"context"
	"log"
	"time"

	"github.com/inet256/inet256/pkg/discovery/landisco"
	"github.com/inet256/inet256/pkg/inet256"
)

func main() {
	ctx := context.Background()
	s, err := landisco.New([]string{"ens192"})
	if err != nil {
		log.Fatal(err)
	}
	ticker := time.NewTicker(time.Second)
	for {
		if err := s.Announce(ctx, inet256.ID{}, nil); err != nil {
			log.Println(err)
		}
		select {
		case <-ticker.C:
		}
	}
}
