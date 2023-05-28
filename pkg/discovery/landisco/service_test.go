package landisco

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/netip"
	"testing"
	"time"

	"github.com/brendoncarroll/go-p2p/s/multiswarm"
	"github.com/brendoncarroll/go-p2p/s/udpswarm"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	"github.com/inet256/inet256/internal/retry"
	"github.com/inet256/inet256/internal/slices2"
	"github.com/inet256/inet256/pkg/discovery"
	"github.com/inet256/inet256/pkg/inet256test"
	"github.com/inet256/inet256/pkg/mesh256"
	"github.com/inet256/inet256/pkg/peers"
)

func TestMulticast(t *testing.T) {
	addr := &udp6MulticastAddr
	ifaces, err := net.Interfaces()
	require.NoError(t, err)
	for _, iface := range ifaces {
		for i := 0; i < 3; i++ {
			conn, err := net.ListenMulticastUDP("udp6", &iface, addr)
			require.NoError(t, err)
			defer conn.Close()
			t.Log("opened multicast conn", conn.LocalAddr(), conn.RemoteAddr())
		}
		maddrs, err := iface.MulticastAddrs()
		require.NoError(t, err)
		t.Log(iface.Name, maddrs)
		require.Contains(t, maddrs, &net.IPAddr{IP: addr.IP})
	}
}

func TestService(t *testing.T) {
	ids := generatePeerIDs(2)
	id1, id2 := ids[0], ids[1]

	ps1 := mesh256.NewPeerStore()
	ps2 := mesh256.NewPeerStore()
	ps1.Add(id2)
	ps2.Add(id1)

	ctx := inet256test.Context(t)
	ctx, cf := context.WithCancel(ctx)
	defer cf()
	ifs, err := net.Interfaces()
	require.NoError(t, err)
	ifNames := slices2.Map(ifs[:1], func(x net.Interface) string { return x.Name })

	ds1, err := New(ifNames, time.Second)
	require.NoError(t, err)
	t.Log(ds1)
	ds2, err := New(ifNames, time.Second)
	require.NoError(t, err)
	eg, ctx2 := errgroup.WithContext(ctx)
	t.Log(ds2)

	eg.Go(func() error {
		return ds1.Run(ctx2, discovery.Params{
			LocalID:       id1,
			GetLocalAddrs: getLocalAddrs,

			Peers:      ps1,
			AddrParser: parser,
		})
	})
	eg.Go(func() error {
		return ds2.Run(ctx2, discovery.Params{
			LocalID:       id2,
			GetLocalAddrs: getLocalAddrs,

			Peers:      ps2,
			AddrParser: parser,
		})
	})
	ctx, cf = context.WithTimeout(ctx, 10*time.Second)
	defer cf()
	err = retry.Retry(ctx, func() error {
		if err := ds2.Announce(ctx, id2, getLocalAddrs()); err != nil {
			t.Log(err)
			return err
		}
		t.Log("did announce")
		addrs := peers.ListAddrs[discovery.TransportAddr](ps1, id2)
		if len(addrs) < 1 {
			return fmt.Errorf("no addresses")
		}
		require.Equal(t, getLocalAddrs(), addrs)
		return nil
	})
	require.NoError(t, err)
}

func parser(x []byte) (multiswarm.Addr, error) {
	parts := bytes.SplitN(x, []byte("://"), 2)
	if len(parts) < 2 {
		return multiswarm.Addr{}, fmt.Errorf("could not parse scheme %v", parts)
	}

	switch string(parts[0]) {
	case "udp":
		var addr udpswarm.Addr
		if err := addr.UnmarshalText(parts[1]); err != nil {
			return multiswarm.Addr{}, err
		}
		return multiswarm.Addr{
			Scheme: "udp",
			Addr:   addr,
		}, nil
	default:
		return multiswarm.Addr{}, fmt.Errorf("unrecognized scheme %q", parts[0])
	}
}

func getLocalAddrs() []multiswarm.Addr {
	return []mesh256.TransportAddr{
		makeAddr("1.2.3.4", 1234),
		makeAddr("5.6.7.8", 5678),
	}
}

func makeAddr(ipStr string, port uint16) multiswarm.Addr {
	return multiswarm.Addr{
		Scheme: "udp",
		Addr:   udpswarm.Addr{IP: netip.MustParseAddr(ipStr), Port: port},
	}
}
