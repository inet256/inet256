package dns256

import (
	"context"
	"fmt"
	"net/netip"

	"go.brendoncarroll.net/stdctx/logctx"
	"go.inet256.org/inet256/src/inet256"
)

type LookupFunc func(ctx context.Context, req Request) (*Response, error)

// NewINET256Root returns an Authority which will consult peer for records using client.
func NewINET256Root(client *Client, peer inet256.Addr) Authority {
	return Authority{
		Lookup: func(ctx context.Context, req Request) (*Response, error) {
			return client.Do(ctx, peer, req)
		},
	}
}

// NewINET256Parser returns a function which parses records into peers and then queries them.
func NewINET256Parser(client *Client) func(Record) (LookupFunc, error) {
	return func(r Record) (LookupFunc, error) {
		addr, err := r.AsINET256("target")
		if err != nil {
			return nil, err
		}
		return func(ctx context.Context, req Request) (*Response, error) {
			return client.Do(ctx, addr, req)
		}, nil
	}
}

// Authority
type Authority struct {
	Path   Path
	Lookup LookupFunc
	TTL    uint32
}

// Resolver performs path resolution.
type Resolver struct {
	roots      []Authority
	makeLookup func(Record) (LookupFunc, error)
}

func NewResolver(roots []Authority, makeLookup func(Record) (LookupFunc, error), opts ...ResolverOpt) *Resolver {
	return &Resolver{
		roots:      roots,
		makeLookup: makeLookup,
	}
}

func (r *Resolver) Resolve(ctx context.Context, q Query, opts ...ResolveOpt) ([]Claim, error) {
	config := resolveConfig{
		maxHops: 32,
	}
	for _, opt := range opts {
		opt(&config)
	}

	sources := append([]Authority{}, r.roots...)
	for i := 0; len(sources) > 0; i++ {
		if i >= config.maxHops {
			return nil, fmt.Errorf("max hops exceeded")
		}
		var target Authority
		target, sources = sources[len(sources)-1], sources[:len(sources)-1]
		q2 := Query{
			Path:   TrimPrefix(q.Path, target.Path),
			Filter: q.Filter,
		}
		res, err := target.Lookup(ctx, Request{
			Query: q2,
		})
		if err != nil {
			return nil, err
		}
		if !res.IsRedirect {
			return res.Claims, nil
		}
		for _, c := range res.Claims {
			prefix, err := c.Record.AsPath("path")
			if err != nil {
				return nil, err
			}
			// TODO: add checks here.
			if !HasPrefix(q2.Path, prefix) {
				return nil, fmt.Errorf("invalid redirect %v from %v", prefix, target)
			}
			lf, err := r.makeLookup(c.Record)
			if err != nil {
				logctx.Warnf(ctx, "error making source: %v", err)
				continue
			}
			src2 := Authority{
				Path:   Concat(target.Path, prefix),
				Lookup: lf,
				TTL:    c.TTL,
			}
			sources = append(sources, src2)
		}
	}
	return nil, nil
}

func (r *Resolver) ResolveINET256(ctx context.Context, x Path, opts ...ResolveOpt) ([]inet256.Addr, error) {
	q := Query{Path: x, Filter: MustHaveKeys("INET256")}
	claims, err := r.Resolve(ctx, q)
	if err != nil {
		return nil, err
	}
	var ret []inet256.Addr
	for _, c := range claims {
		addr, err := c.Record.AsINET256("INET256")
		if err != nil {
			continue
		}
		ret = append(ret, addr)
	}
	return ret, nil
}

func (r *Resolver) ResolveIP(ctx context.Context, x Path, opts ...ResolveOpt) ([]netip.Addr, error) {
	q := Query{Path: x, Filter: MustHaveKeys("IP")}
	cs, err := r.Resolve(ctx, q, opts...)
	if err != nil {
		return nil, err
	}
	var ret []netip.Addr
	for _, c := range cs {
		addr, err := c.Record.AsIP("IP")
		if err != nil {
			continue
		}
		ret = append(ret, addr)
	}
	return ret, nil
}

func (r *Resolver) ResolveIPPort(ctx context.Context, x Path, opts ...ResolveOpt) ([]netip.AddrPort, error) {
	q := Query{Path: x, Filter: MustHaveKeys("IP", "port")}
	ents, err := r.Resolve(ctx, q, opts...)
	if err != nil {
		return nil, err
	}
	var ret []netip.AddrPort
	for _, ent := range ents {
		ip, err := ent.Record.AsIP("IP")
		if err != nil {
			continue
		}
		port, err := ent.Record.AsUint16("port")
		if err != nil {
			continue
		}
		ret = append(ret, netip.AddrPortFrom(ip, port))
	}
	return ret, nil
}
