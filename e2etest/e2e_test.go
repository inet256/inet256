package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/p2ptest"
	"github.com/inet256/inet256/client/go_client/inet256client"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256d"
	"github.com/inet256/inet256/pkg/inet256test"
	"github.com/inet256/inet256/pkg/serde"
	"github.com/stretchr/testify/require"
)

func Test2Node(t *testing.T) {
	dir1, dir2 := t.TempDir(), t.TempDir()

	priv1 := p2ptest.NewTestKey(t, 1)
	priv2 := p2ptest.NewTestKey(t, 2)
	addr1 := inet256.NewAddr(priv1.Public())
	addr2 := inet256.NewAddr(priv2.Public())

	setupSide(t, dir1, priv1, 2561, 50001)
	setupSide(t, dir2, priv2, 2562, 50002)

	addPeer(t, dir1, addr2, []string{fmt.Sprintf("quic+udp://%v@127.0.0.1:50002", addr2)})
	addPeer(t, dir2, addr1, []string{fmt.Sprintf("quic+udp://%v@127.0.0.1:50001", addr1)})

	runDaemon(t, dir1)
	runDaemon(t, dir2)

	c1 := newClient(t, 2561)
	c2 := newClient(t, 2562)

	ctx := context.Background()
	n1, err := c1.Open(ctx, p2ptest.NewTestKey(t, 101))
	require.NoError(t, err)
	t.Cleanup(func() { n1.Close() })
	n2, err := c2.Open(ctx, p2ptest.NewTestKey(t, 102))
	require.NoError(t, err)
	t.Cleanup(func() { n2.Close() })

	inet256test.TestSendRecvOne(t, n1, n2)
	inet256test.TestSendRecvOne(t, n2, n1)
}

func setupSide(t testing.TB, dir string, privateKey p2p.PrivateKey, apiPort, listenPort int) {
	config := inet256d.DefaultConfig()
	config.PrivateKeyPath = "./private_key.pem"
	config.APIEndpoint = "127.0.0.1:" + strconv.Itoa(apiPort)
	config.Transports = []inet256d.TransportSpec{
		newUDPTransportSpec("127.0.0.1:" + strconv.Itoa(listenPort)),
	}
	configPath := filepath.Join(dir, "config.yaml")
	require.NoError(t, inet256d.SaveConfig(config, configPath))

	keyPath := filepath.Join(dir, "private_key.pem")
	data, err := serde.MarshalPrivateKeyPEM(privateKey)
	require.NoError(t, err)
	err = ioutil.WriteFile(keyPath, data, 0o644)
	require.NoError(t, err)
}

func addPeer(t testing.TB, dir string, id inet256.Addr, addrs []string) {
	configPath := filepath.Join(dir, "config.yaml")
	config, err := inet256d.LoadConfig(configPath)
	require.NoError(t, err)
	config.Peers = append(config.Peers, inet256d.PeerSpec{
		ID:    id,
		Addrs: addrs,
	})
	require.NoError(t, inet256d.SaveConfig(*config, configPath))
	return
}

func newUDPTransportSpec(x string) inet256d.TransportSpec {
	y := inet256d.UDPTransportSpec(x)
	return inet256d.TransportSpec{
		UDP: &y,
	}
}

func runDaemon(t testing.TB, dir string) {
	d := newDaemon(t, dir)
	ctx, cf := context.WithCancel(context.Background())
	t.Cleanup(cf)
	go d.Run(ctx)
}

func newDaemon(t testing.TB, dir string) *inet256d.Daemon {
	configPath := filepath.Join(dir, "config.yaml")
	c, err := inet256d.LoadConfig(configPath)
	require.NoError(t, err)
	params, err := inet256d.MakeParams(configPath, *c)
	require.NoError(t, err)
	return inet256d.New(*params)
}

func newClient(t testing.TB, apiPort int) inet256.Service {
	client, err := inet256client.NewClient("127.0.0.1:" + strconv.Itoa(apiPort))
	require.NoError(t, err)
	return client
}
