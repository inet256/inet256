package inet256ipc

import (
	"context"
	"log"
	"net"
	"testing"

	"go.inet256.org/inet256/src/inet256"
	"go.inet256.org/inet256/src/inet256mem"
	"go.inet256.org/inet256/src/inet256tests"
)

var ctx = context.Background()

func TestSendRecvOne(t *testing.T) {
	s := inet256mem.New()
	n1 := inet256tests.OpenNode(t, s, 0)
	n2 := inet256tests.OpenNode(t, s, 1)
	n1, n2 = newTestNode(t, n1), newTestNode(t, n2)
	inet256tests.TestSendRecvOne(t, n1, n2)
}

func newTestNode(t testing.TB, x inet256.Node) inet256.Node {
	c1, c2 := net.Pipe()
	go func() {
		log.Println("serving node")
		ServeNode(ctx, x, NewStreamFramer(c1, c1))
		log.Println("done serving node")
	}()
	nc := NewNodeClient(NewStreamFramer(c2, c2), x.PublicKey())
	t.Cleanup(func() {
		log.Println("starting cleanup")
		c1.Close()
		c2.Close()
		log.Println("closing node client...")
		nc.Close()
		log.Println("closed node client")
	})
	return nc
}
