package inet256d

import (
	"context"
	"net"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/inet256/inet256/pkg/inet256http"
	"github.com/inet256/inet256/pkg/mesh256"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

// runHTTPServer starts a listener at endpoint, and serves an HTTP API server backed by srv.
func (d *Daemon) runHTTPServer(ctx context.Context, endpoint string, srv *mesh256.Server, pgath prometheus.Gatherer) error {
	l, err := net.Listen("tcp", endpoint)
	if err != nil {
		return err
	}
	defer l.Close()

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
		ReadTimeout: 0,
		IdleTimeout: 0,
	}
	go func() {
		logrus.Println("API listening on: ", l.Addr())
		if err := hSrv.Serve(l); err != nil && err != http.ErrServerClosed {
			logrus.Errorf("error serving http: %v", err)
		}
	}()
	<-ctx.Done()
	return hSrv.Shutdown(context.Background())
}
