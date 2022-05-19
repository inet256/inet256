package main

import (
	"context"
	"crypto/ed25519"
	"log"
	"strconv"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256mem"
	kcp "github.com/xtaci/kcp-go"
	"golang.org/x/sync/errgroup"
)

func main() {
	if err := run(); err != nil {
		log.Println("ERROR:", err)
	}
}

func run() error {
	// srv := mesh256.NewServer(mesh256.Params{
	// 	NewNetwork: beaconnet.Factory,
	// 	PrivateKey: generateKey(),
	// 	Peers:      mesh256.NewPeerStore(),
	// })
	srv := inet256mem.New()
	// listener
	ctx := context.Background()
	n1, err := srv.Open(ctx, generateKey())
	if err != nil {
		return err
	}
	defer n1.Close()
	n2, err := srv.Open(ctx, generateKey())
	if err != nil {
		return err
	}
	defer n2.Close()
	bc, _ := kcp.NewNoneBlockCrypt(nil)
	l, err := kcp.ServeConn(bc, 1, 0, inet256.NewPacketConn(n1))
	if err != nil {
		return err
	}
	defer l.Close()
	eg := errgroup.Group{}
	eg.Go(func() error {
		for i := 0; i < 1; i++ {
			log.Println("listening on", l.Addr())
			c, err := l.Accept()
			if err != nil {
				return err
			}
			log.Println("accepted connection from", c.RemoteAddr())
			eg.Go(func() error {
				defer c.Close()
				buf := make([]byte, 1024)
				n, err := c.Read(buf)
				if err != nil {
					return err
				}
				log.Printf("listener received %d bytes: %q", n, buf[:n])
				if _, err := c.Write([]byte(strconv.Itoa(n))); err != nil {
					return err
				}
				log.Println("listener wrote reply")
				return nil
			})
		}
		return nil
	})
	eg.Go(func() error {
		log.Println("dialing", l.Addr())
		c, err := kcp.NewConn2(l.Addr(), bc, 1, 0, inet256.NewPacketConn(n2))
		if err != nil {
			log.Println("error dialing", err)
			return err
		}
		defer c.Close()
		if _, err := c.Write([]byte("hello world")); err != nil {
			return err
		}
		buf := make([]byte, 1024)
		n, err := c.Read(buf)
		if err != nil {
			log.Println(err)
		}
		log.Printf("dialer received from server: %q", buf[:n])
		return nil
	})
	return eg.Wait()
}

func generateKey() inet256.PrivateKey {
	_, privKey, _ := ed25519.GenerateKey(nil)
	return privKey
}
