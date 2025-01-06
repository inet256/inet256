package dns256

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"go.inet256.org/inet256/pkg/inet256"
	"go.inet256.org/inet256/pkg/inet256mem"
	"go.inet256.org/inet256/pkg/inet256test"
	"golang.org/x/exp/slices"
	"golang.org/x/sync/errgroup"
)

var ctx = context.Background()

func TestParseDots(t *testing.T) {
	for _, tc := range []struct {
		In  string
		Out Path
		Err error
	}{
		{In: "", Out: Path{}},
		{In: "com", Out: Path{"com"}},
		{In: "www.example.com", Out: Path{"com", "example", "www"}},
	} {
		y, err := ParseDots(tc.In)
		if err != nil {
			require.Nil(t, y)
			require.Equal(t, tc.Err, err)
		} else {
			require.Equal(t, tc.Out, y)
		}
	}
}

func TestClientServer(t *testing.T) {
	n1, n2 := newPair(t)
	res := doOne(t, n1, n2, Request{
		Query: Query{
			Path: MustParseDots("www.example.com"),
		},
	}, func(ctx context.Context, res *Response, req *Request) bool {
		res.Claims = []Claim{
			{Record: Record{"key": NewValue("value")}, TTL: 300},
		}
		return true
	})
	require.NotNil(t, res)
	t.Log(res.Claims)
	require.Len(t, res.Claims, 1)
	require.Equal(t, must(res.Claims[0].Record.AsString("key")), "value")
}

func TestResolveChain(t *testing.T) {
	nodes := makeNodes(t, 4)
	cnode := nodes[0]
	snodes := nodes[1:]

	ctx, cf := context.WithCancel(ctx)
	defer cf()
	eg := errgroup.Group{}
	t.Cleanup(func() {
		cf()
		require.NoError(t, eg.Wait())
	})
	eg.Go(func() error {
		bh := BasicHandler{
			Redirects: []RedirectINET256{
				{
					Path:   Path{"a"},
					Target: snodes[1].LocalAddr(),
				},
			},
		}
		return Serve(ctx, snodes[0], bh.Handle)
	})
	eg.Go(func() error {
		bh := BasicHandler{
			Redirects: []RedirectINET256{
				{
					Path:   Path{"b"},
					Target: snodes[2].LocalAddr(),
				},
			},
		}
		return Serve(ctx, snodes[1], bh.Handle)
	})
	eg.Go(func() error {
		return Serve(ctx, snodes[2], func(ctx context.Context, res *Response, req *Request) bool {
			if slices.Equal(req.Query.Path, Path{"c"}) {
				res.Claims = []Claim{
					{
						Record: Record{
							"key1": NewValue("hello world"),
						},
						TTL: 300,
					},
				}
				return true
			}
			return false
		})
	})

	client := NewClient(cnode)
	r := NewResolver([]Authority{NewINET256Root(client, snodes[0].LocalAddr())}, NewINET256Parser(client))
	ents, err := r.Resolve(ctx, Query{Path: Path{"a", "b", "c"}})
	require.NoError(t, err)
	t.Log(ents)
	require.Len(t, ents, 1)
	require.Equal(t, "hello world", must(ents[0].Record.AsString("key1")))
}

func newPair(t testing.TB) (n1, n2 inet256.Node) {
	ns := makeNodes(t, 2)
	return ns[0], ns[1]
}

func makeNodes(t testing.TB, n int) (ns []inet256.Node) {
	s := inet256mem.New()
	ns = make([]inet256.Node, n)
	var err error
	for i := range ns {
		pk := inet256test.NewPrivateKey(t, i)
		ns[i], err = s.Open(ctx, pk)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, s.Drop(ctx, pk)) })
	}
	return ns
}

func doOne(t testing.TB, cnode, snode inet256.Node, req Request, h Handler) Response {
	ctx, cf := context.WithCancel(ctx)
	defer cf()
	c := NewClient(cnode)
	eg := errgroup.Group{}
	eg.Go(func() error {
		return Serve(ctx, snode, h)
	})
	var res Response
	eg.Go(func() error {
		defer cf()
		req := Request{
			Query: Query{
				Path: MustParseDots("www.example.com"),
			},
		}
		res2, err := c.Do(ctx, snode.LocalAddr(), req)
		if err != nil {
			return err
		}
		res = *res2
		return nil
	})
	require.NoError(t, eg.Wait())
	return res
}

func must[T any](x T, err error) T {
	if err != nil {
		panic(err)
	}
	return x
}
