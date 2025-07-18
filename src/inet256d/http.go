package inet256d

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"

	"github.com/go-chi/chi"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.brendoncarroll.net/stdctx/logctx"
	"golang.org/x/exp/slices"

	"go.inet256.org/inet256/src/inet256http"
	"go.inet256.org/inet256/src/mesh256"
)

// runHTTPServer starts a listener at endpoint, and serves an HTTP API server backed by srv.
func (d *Daemon) runHTTPServer(ctx context.Context, endpoint string, srv *mesh256.Server, pgath prometheus.Gatherer) error {
	u, err := url.Parse(endpoint)
	if err != nil {
		return err
	}
	if u.Scheme == "" {
		return fmt.Errorf("endpoint is missing scheme %q", endpoint)
	}
	laddr := path.Join(u.Host, u.Path)
	l, err := net.Listen(u.Scheme, laddr)
	if err != nil {
		return err
	}
	defer l.Close()
	if u.Scheme == "unix" {
		if err := os.Chmod(laddr, 0o666); err != nil {
			return err
		}
		defer os.Remove(laddr)
	}

	mux := chi.NewMux()

	// INET256 Service
	mux.Handle("/nodes/*", inet256http.NewServer(srv))
	// health check
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("INET256\n"))
	})
	// prometheus metrics
	mux.Handle("/metrics", promhttp.HandlerFor(pgath, promhttp.HandlerOpts{}))
	hSrv := http.Server{
		Handler:     mux,
		BaseContext: func(l net.Listener) context.Context { return ctx },
	}
	mux.HandleFunc("/admin/status", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		peers, _ := srv.PeerStatus(ctx)
		transportAddrs, _ := srv.TransportAddrs(ctx)
		slices.SortFunc(transportAddrs, func(a, b mesh256.TransportAddr) bool {
			return a.String() < b.String()
		})
		mainAddr, _ := srv.MainAddr(ctx)
		data, _ := json.Marshal(StatusRes{
			MainAddr:       mainAddr,
			TransportAddrs: mapSlice(transportAddrs, func(x TransportAddr) string { return fmt.Sprint(x) }),
			Peers:          peers,
		})
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	})
	go func() {
		logctx.Infoln(ctx, "API listening on: ", l.Addr())
		if err := hSrv.Serve(l); err != nil && err != http.ErrServerClosed {
			logctx.Infof(ctx, "error serving http: %v", err)
		}
	}()
	<-ctx.Done()
	return hSrv.Shutdown(context.Background())
}

func mapSlice[X, Y any](xs []X, fn func(X) Y) (ret []Y) {
	ret = make([]Y, len(xs))
	for i := range xs {
		ret[i] = fn(xs[i])
	}
	return ret
}
