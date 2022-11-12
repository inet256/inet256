package inet256ipc

import (
	"context"
	"log"
	"net"
	"testing"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256mem"
	"github.com/inet256/inet256/pkg/inet256test"
)

var ctx = context.Background()

func TestSendRecvOne(t *testing.T) {
	s := inet256mem.New()
	n1 := inet256test.OpenNode(t, s, 0)
	n2 := inet256test.OpenNode(t, s, 1)
	n1, n2 = newTestNode(t, n1), newTestNode(t, n2)
	inet256test.TestSendRecvOne(t, n1, n2)
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
