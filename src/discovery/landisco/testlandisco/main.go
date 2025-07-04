package main

import (
	"context"
	"log"
	"os"
	"time"

	"go.brendoncarroll.net/tai64"
	"go.inet256.org/inet256/src/discovery/landisco"
	"go.inet256.org/inet256/src/inet256"
)

func main() {
	args := os.Args[1:]
	if len(args) < 1 {
		log.Fatalf("must provide interface name")
	}
	ctx := context.Background()
	s, err := landisco.NewBus(ctx, []string{args[0]}, time.Second)
	if err != nil {
		log.Fatal(err)
	}
	ticker := time.NewTicker(time.Second)
	now := tai64.Now()
	for {
		if err := s.Announce(ctx, now, inet256.ID{}, nil); err != nil {
			log.Println(err)
		}
		select {
		case <-ctx.Done():
			log.Fatal(ctx.Err())
		case <-ticker.C:
		}
	}
}
