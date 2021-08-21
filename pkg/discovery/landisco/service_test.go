package landisco

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/s/udpswarm"
	"github.com/inet256/inet256/internal/netutil"
	"github.com/inet256/inet256/pkg/discovery"
	"github.com/inet256/inet256/pkg/mesh256"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestService(t *testing.T) {
	ids := generatePeerIDs(2)
	id1, id2 := ids[0], ids[1]

	ps1 := mesh256.NewPeerStore()
	ps2 := mesh256.NewPeerStore()
	ps1.Add(id2)
	ps2.Add(id1)

	ctx, cf := context.WithCancel(context.Background())
	defer cf()
	ds1 := New().(*service)
	ds2 := New().(*service)
	go discovery.RunForever(ctx, ds1, discovery.Params{
		LocalID:       id1,
		GetLocalAddrs: getLocalAddrs,

		AddressBook: ps1,
		AddrParser:  parser,

		Logger: logrus.StandardLogger(),
	})
	go discovery.RunForever(ctx, ds2, discovery.Params{
		LocalID:       id2,
		GetLocalAddrs: getLocalAddrs,

		AddressBook: ps2,
		AddrParser:  parser,

		Logger: logrus.StandardLogger(),
	})
	ctx, cf = context.WithTimeout(ctx, 10*time.Second)
	defer cf()
	err := netutil.Retry(ctx, func() error {
		ds2.forceAnnounce()
		log.Println("did announce")
		addrs := ps1.ListAddrs(id2)
		if len(addrs) < 1 {
			return errors.Errorf("no addresses")
		}
		require.Equal(t, getLocalAddrs(), addrs)
		return nil
	})
	require.NoError(t, err)
}

func parser(x []byte) (p2p.Addr, error) {
	addr := udpswarm.Addr{}
	err := addr.UnmarshalText(x)
	return addr, err
}

func getLocalAddrs() []p2p.Addr {
	return []p2p.Addr{
		udpswarm.Addr{Port: 1234},
		udpswarm.Addr{Port: 5678},
	}
}
