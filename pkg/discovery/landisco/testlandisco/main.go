package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/inet256/inet256/pkg/discovery/landisco"
	"github.com/inet256/inet256/pkg/inet256"
)

func main() {
	args := os.Args[1:]
	if len(args) < 1 {
		log.Fatalf("must provide interface name")
	}
	ctx := context.Background()
	s, err := landisco.New([]string{args[0]}, time.Second)
	if err != nil {
		log.Fatal(err)
	}
	ticker := time.NewTicker(time.Second)
	for {
		if err := s.Announce(ctx, inet256.ID{}, nil); err != nil {
			log.Println(err)
		}
		select {
		case <-ctx.Done():
			log.Fatal(ctx.Err())
		case <-ticker.C:
		}
	}
}
