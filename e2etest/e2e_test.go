package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	inet256client "go.inet256.org/inet256/client/go"
	"go.inet256.org/inet256/src/inet256"
	"go.inet256.org/inet256/src/inet256d"
	"go.inet256.org/inet256/src/inet256http"
	"go.inet256.org/inet256/src/inet256tests"
	"go.inet256.org/inet256/src/mesh256"
)

var ctx = context.Background()

func Test2Node(t *testing.T) {
	sides := make([]*side, 2)
	for i := range sides {
		sides[i] = newSide(t, i)
	}
	for i := range sides {
		for j := range sides {
			if i == j {
				continue
			}
			sides[i].peerWith(t, sides[j])
		}
	}
	for i := range sides {
		sides[i].startDaemon(t)
	}
	for i := range sides {
		c := sides[i].newClient(t).(*inet256http.Client)
		require.NoError(t, c.Ping(ctx))
	}
	n1 := sides[0].newNode(t, inet256tests.NewPrivateKey(t, 101))
	n2 := sides[1].newNode(t, inet256tests.NewPrivateKey(t, 102))
	inet256tests.TestSendRecvOne(t, n1, n2)
	inet256tests.TestSendRecvOne(t, n2, n1)
}

type side struct {
	i             int
	dir           string
	privateKey    inet256.PrivateKey
	apiEndpoint   string
	transportPort int

	d *inet256d.Daemon
}

func newSide(t testing.TB, i int) *side {
	dir := t.TempDir()
	privateKey := inet256tests.NewPrivateKey(t, i)
	transportPort := 32000 + i

	config := inet256d.DefaultConfig()
	config.PrivateKeyPath = "./private_key.inet256"
	config.APIEndpoint = fmt.Sprintf("unix://%s/inet256-%d.sock", dir, i)
	config.Transports = []inet256d.TransportSpec{
		newUDPTransportSpec("127.0.0.1:" + strconv.Itoa(transportPort)),
	}
	configPath := filepath.Join(dir, "config.yaml")
	require.NoError(t, inet256d.SaveConfig(config, configPath))

	keyPath := filepath.Join(dir, "private_key.inet256")
	data, err := inet256.DefaultPKI.MarshalPrivateKey(nil, privateKey)
	require.NoError(t, err)
	err = os.WriteFile(keyPath, data, 0o644)
	require.NoError(t, err)

	return &side{
		i:             i,
		dir:           dir,
		privateKey:    privateKey,
		apiEndpoint:   config.APIEndpoint,
		transportPort: transportPort,
	}
}

func (s *side) updateConfig(t testing.TB, fn func(inet256d.Config) inet256d.Config) {
	x, err := inet256d.LoadConfig(s.configPath())
	require.NoError(t, err)
	y := fn(*x)
	require.NoError(t, inet256d.SaveConfig(y, s.configPath()))
}

func (s *side) peerWith(t testing.TB, s2 *side) {
	s.addPeer(t, inet256d.PeerSpec{
		ID:    s2.localAddr(),
		Addrs: s2.transportAddrs(),
	})
}

func (s *side) addPeer(t testing.TB, x inet256d.PeerSpec) {
	s.updateConfig(t, func(config inet256d.Config) inet256d.Config {
		config.Peers = append(config.Peers, x)
		return config
	})
}

func (s *side) addDiscovery(t testing.TB, x inet256d.DiscoverySpec) {
	s.updateConfig(t, func(config inet256d.Config) inet256d.Config {
		config.Discovery = append(config.Discovery, x)
		return config
	})
}

func (s *side) configPath() string {
	return filepath.Join(s.dir, "config.yaml")
}

func (s *side) transportAddrs() []string {
	return []string{fmt.Sprintf("%s://%v@127.0.0.1:%d", mesh256.SecureProtocolName("udp"), s.localAddr(), s.transportPort)}
}

func (s *side) localAddr() inet256.Addr {
	return inet256.NewID(s.privateKey.Public().(inet256.PublicKey))
}

func (s *side) newClient(t testing.TB) inet256.Service {
	client, err := inet256client.NewClient(s.apiEndpoint)
	require.NoError(t, err)
	return client
}

// newNode returns a node which is cleaned up at the end of the test
func (s *side) newNode(t testing.TB, privateKey inet256.PrivateKey) inet256.Node {
	client := s.newClient(t)
	node, err := client.Open(ctx, privateKey)
	require.NoError(t, err)
	t.Cleanup(func() { node.Close() })
	return node
}

func (s *side) startDaemon(t testing.TB) {
	if s.d != nil {
		panic("daemon already started")
	}
	configPath := s.configPath()
	c, err := inet256d.LoadConfig(configPath)
	require.NoError(t, err)
	params, err := inet256d.MakeParams(configPath, *c)
	require.NoError(t, err)
	d := inet256d.New(*params)

	// run daemon, cancel then block until it exists during cleanup
	ctx, cf := context.WithCancel(ctx)
	done := make(chan struct{})
	t.Cleanup(func() {
		cf()
		t.Log("canceled daemon context.  waiting for daemon to exit...")
		<-done
	})
	go func() {
		defer close(done)
		if err := d.Run(ctx); err != nil {
			t.Log(err)
		}
	}()

	s.d = d
}

func newUDPTransportSpec(x string) inet256d.TransportSpec {
	y := inet256d.UDPTransportSpec(x)
	return inet256d.TransportSpec{
		UDP: &y,
	}
}
