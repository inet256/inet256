package inet256ipv6

import (
	"context"
	"io"
	"runtime"
	"sync/atomic"

	"go.brendoncarroll.net/p2p/p/kademlia"
	"go.brendoncarroll.net/stdctx/logctx"
	"golang.org/x/sync/errgroup"

	"go.inet256.org/inet256/src/inet256"
)

const (
	// WorthItBits is the number of leading 0s the address must have
	// for the encoding scheme to be better than just taking the prefix
	WorthItBits = 9

	// PreImageResistance128Bits is the number of leading 0s the address must have
	// for the uncompressed prefix to have 128 bits of preimage resistance.
	PreImageResistance128Bits = 16
)

// MineAddr repeatedly generates private-keys using entropy from r, derives INET256 addresses from them,
// and checks how well they compress into the IPv6 mapping.
// goal is the number of leading 0s to achieve in address before stopping.
func MineAddr(ctx context.Context, r io.Reader, goal int) (inet256.Addr, inet256.PrivateKey, error) {
	N := runtime.GOMAXPROCS(0)
	const checkEvery = 1000

	addrs := make([]inet256.Addr, N)
	privKeys := make([]inet256.PrivateKey, N)
	var count uint64

	ctx, cf := context.WithCancel(ctx)
	defer cf()
	eg, ctx := errgroup.WithContext(ctx)
	for i := 0; i < N; i++ {
		i := i
		addrs[i] = allOnes()
		eg.Go(func() error {
			for {
				for j := 0; j < checkEvery; j++ {
					pubKey, privKey, err := inet256.GenerateKey()
					if err != nil {
						panic(err)
					}
					addr := inet256.NewID(pubKey)
					if leading0s(addr[:]) > leading0s(addrs[i][:]) {
						addrs[i] = addr
						privKeys[i] = privKey
						if leading0s(addr[:]) >= goal {
							atomic.AddUint64(&count, uint64(j))
							cf()
							return nil
						}
					}
				}
				atomic.AddUint64(&count, uint64(checkEvery))
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}
			}
		})
	}
	err := eg.Wait()
	logctx.Infof(ctx, "searched %d addresses", count)
	// find the addr with the most leading 0s, which also meets the goal.
	maxLz := -1
	maxI := len(addrs)
	for i := range addrs {
		lz := leading0s(addrs[i][:])
		if lz < goal {
			continue
		}
		if lz > maxLz {
			maxLz = lz
			maxI = i
		}
	}
	// check if we found one
	if maxI < len(addrs) {
		return addrs[maxI], privKeys[maxI], nil
	}
	return inet256.Addr{}, nil, err
}

func leading0s(x []byte) int {
	return kademlia.LeadingZeros(x)
}

func allOnes() inet256.Addr {
	a := inet256.Addr{}
	for i := range a[:] {
		a[i] = 0xff
	}
	return a
}
