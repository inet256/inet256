package inet256ipv6

import (
	"context"
	"crypto/ed25519"
	"io"
	"math/bits"
	"runtime"
	"sync"

	"github.com/brendoncarroll/go-p2p"

	"github.com/inet256/inet256/pkg/inet256"
)

func MineAddr(ctx context.Context, rand io.Reader, goal int) (inet256.Addr, p2p.PrivateKey, error) {
	N := runtime.GOMAXPROCS(0)
	checkEvery := 1000

	addrs := make([]inet256.Addr, N)
	privKeys := make([]ed25519.PrivateKey, N)

	ctx, cf := context.WithCancel(ctx)
	defer cf()
	wg := sync.WaitGroup{}
	wg.Add(N)

	for i := 0; i < N; i++ {
		i := i
		go func() {
			defer wg.Done()

			addrs[i] = allOnes()
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}
				for j := 0; j < checkEvery; j++ {
					pubKey, privKey, err := ed25519.GenerateKey(rand)
					if err != nil {
						panic(err)
					}
					addr := p2p.NewPeerID(pubKey)
					if leading0s(addr[:]) > leading0s(addrs[i][:]) {
						addrs[i] = addr
						privKeys[i] = privKey
						if leading0s(addr[:]) >= goal {
							cf()
							return
						}
					}
				}
			}
		}()
	}
	wg.Wait()

	max := -1
	for i := range addrs {
		if privKeys[i] == nil {
			continue
		}
		if leading0s(addrs[i][:]) > leading0s(addrs[max][:]) {
			max = i
		}
	}

	if max == -1 {
		return p2p.ZeroPeerID(), nil, ctx.Err()
	}
	var err error
	if leading0s(addrs[max][:]) >= goal {
		err = nil
	} else {
		err = ctx.Err()
	}

	return addrs[max], privKeys[max], err
}

func leading0s(x []byte) int {
	lz := 0
	for i := range x {
		lzi := bits.LeadingZeros8(x[i])
		lz += lzi
		if lzi < 8 {
			break
		}
	}
	return lz
}

func allOnes() inet256.Addr {
	a := inet256.Addr{}
	for i := range a[:] {
		a[i] = 0xff
	}
	return a
}
