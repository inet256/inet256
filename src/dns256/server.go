package dns256

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"go.brendoncarroll.net/stdctx/logctx"

	"go.inet256.org/inet256/src/inet256"
)

// Handler modifies res in response to req and returns true if a message should be sent in response.
// The res.RequestID will be set automatically
type Handler func(ctx context.Context, res *Response, req *Request) bool

// Serve uses h to handle and respond to requests received by node until a
// non-transient error occurs.
// Such an error could come from the Node, or the Context.
// Serve returns nil, only if node returns context.Cancelled.
func Serve(ctx context.Context, node inet256.Node, h Handler) error {
	wg := sync.WaitGroup{}
	defer wg.Wait()
	for {
		if err := node.Receive(ctx, func(msg inet256.Message) {
			var req Request
			if err := json.Unmarshal(msg.Payload, &req); err != nil {
				logctx.Warnf(ctx, "error parsing json: %v", err)
				return
			}
			reqID := req.ID
			replyTo := msg.Src

			wg.Add(1)
			go func() {
				defer wg.Done()
				ctx, cf := context.WithTimeout(ctx, 5*time.Second)
				defer cf()
				var res Response
				if h(ctx, &res, &req) {
					res.RequestID = reqID
					data, err := json.Marshal(res)
					if err != nil {
						panic(err)
					}
					if err := node.Send(ctx, replyTo, data); err != nil {
						logctx.Errorf(ctx, "sending: %v", err)
					}
				}
			}()
		}); err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		}
	}
}

// BasicHandler is a simple server backed with some redirects and some records.
type BasicHandler struct {
	Redirects []RedirectINET256
	Records   []Record
	TTL       uint32
}

func (bh BasicHandler) Handle(ctx context.Context, res *Response, req *Request) bool {
	ttl := bh.TTL
	if ttl <= 0 {
		ttl = 300
	}
	for _, red := range bh.Redirects {
		if HasPrefix(req.Query.Path, red.Path) {
			res.IsRedirect = true
			res.Claims = append(res.Claims, Claim{
				Record: red.ToRecord(),
				TTL:    ttl,
			})
			return true
		}
	}
	for _, rec := range bh.Records {
		if req.Query.Matches(rec) {
			res.Claims = append(res.Claims, Claim{
				Record: rec,
				TTL:    ttl,
			})
		}
	}
	return true
}

type Layer struct {
	Prefix  Path
	Handler Handler
}

type LayeredHandler []Layer

func (h LayeredHandler) Handle(res *Response, req *Request) bool {
	for _, l := range h {
		if HasPrefix(l.Prefix, req.Query.Path) {
			return h.Handle(res, req)
		}
	}
	return false
}
