package inet256d

import (
	"context"
	"net"
	"net/http"

	"time"

	"github.com/go-chi/chi"
	"github.com/inet256/inet256/pkg/inet256grpc"
	"github.com/inet256/inet256/pkg/mesh256"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

// runHTTPServer starts a listener at endpoint, and serves an HTTP API server backed by srv.
func (d *Daemon) runHTTPServer(ctx context.Context, endpoint string, srv *mesh256.Server, pgath prometheus.Gatherer) error {
	l, err := net.Listen("tcp", endpoint)
	if err != nil {
		return err
	}
	defer l.Close()

	grpcServer := d.makeGRPCServer(srv)

	mux := chi.NewMux()
	// grpc services
	for name := range grpcServer.GetServiceInfo() {
		mux.Handle("/"+name+"/*", grpcServer)
	}
	// health check
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("INET256\n"))
	})
	// prometheus metrics
	mux.Handle("/metrics", promhttp.HandlerFor(pgath, promhttp.HandlerOpts{}))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: this is the recommended way of mixing gRPC with other http routes.
		// But it seems better to have full control over which paths go to gRPC.
		// if r.ProtoMajor == 2 && strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
		// 	grpcServer.ServeHTTP(w, r)
		// } else {
		// 	mux.ServeHTTP(w, r)
		// }
		mux.ServeHTTP(w, r)
	})
	h2Srv := &http2.Server{}
	hSrv := http.Server{
		Handler:     h2c.NewHandler(handler, h2Srv),
		BaseContext: func(l net.Listener) context.Context { return ctx },
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

func (d *Daemon) makeGRPCServer(s *mesh256.Server) *grpc.Server {
	gs := grpc.NewServer(grpc.KeepaliveParams(keepalive.ServerParameters{
		Time:    1 * time.Second,
		Timeout: 5 * time.Second,
	}))
	x := inet256grpc.NewServer(s)
	inet256grpc.RegisterINET256Server(gs, x)
	inet256grpc.RegisterAdminServer(gs, x)
	return gs
}
