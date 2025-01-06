package dns256

import (
	"context"
	"crypto/rand"
	"encoding/json"

	"go.brendoncarroll.net/exp/futures"
	"go.inet256.org/inet256/pkg/inet256"
)

// Client manages creating Requests and awaiting Responses.
type Client struct {
	node inet256.Node

	reqs *futures.Store[reqKey, *Response]
}

func NewClient(node inet256.Node) *Client {
	return &Client{
		node: node,
		reqs: futures.NewStore[reqKey, *Response](),
	}
}

func (c *Client) Do(ctx context.Context, dst inet256.Addr, req Request) (*Response, error) {
	ctx, cf := context.WithCancel(ctx)
	defer cf()
	if _, err := rand.Read(req.ID[:]); err != nil {
		return nil, err
	}
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	key := reqKey{
		Addr: dst,
		ID:   req.ID,
	}
	prom, created := c.reqs.GetOrCreate(key)
	if created {
		defer c.reqs.Delete(key, prom)
		go c.readLoop(ctx)
	}
	if err := c.node.Send(ctx, dst, data); err != nil {
		return nil, err
	}
	return futures.Await[*Response](ctx, prom)
}

func (c *Client) readLoop(ctx context.Context) error {
	for {
		if err := c.node.Receive(ctx, func(msg inet256.Message) {
			var res Response
			if err := json.Unmarshal(msg.Payload, &res); err != nil {
				return
			}
			if fut := c.reqs.Get(reqKey{Addr: msg.Src, ID: res.RequestID}); fut != nil {
				fut.Succeed(&res)
			}
		}); err != nil {
			return err
		}
	}
}

type reqKey struct {
	Addr inet256.Addr
	ID   [16]byte
}
