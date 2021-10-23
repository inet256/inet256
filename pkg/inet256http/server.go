package inet256http

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

type Server struct {
	service inet256.Service
	router  chi.Router
	log     *logrus.Logger
}

func NewServer(srv inet256.Service) *Server {
	router := chi.NewMux()
	s := &Server{
		service: srv,
		router:  router,
		log:     logrus.StandardLogger(),
	}
	router.Route("/v0", func(r chi.Router) {
		r.Post("/connect", s.connect)
		r.Post("/findAddr", s.findAddr)
		r.Get("/mtu/{id}", s.mtu)
		r.Get("/public-key/{id}", s.lookupKey)
	})
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) findAddr(w http.ResponseWriter, r *http.Request) {
	var x FindAddrReq
	readJSON(r.Body, &x)
}

func (s *Server) mtu(w http.ResponseWriter, r *http.Request) {

}

func (s *Server) lookupKey(w http.ResponseWriter, r *http.Request) {

}

func (s *Server) connect(w http.ResponseWriter, r *http.Request) {
	flusher := w.(http.Flusher)
	ctx := r.Context()
	privateKey, err := getPrivateKey(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	node, err := s.service.CreateNode(ctx, privateKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	defer func() {
		if err := node.Close(); err != nil {
			s.log.Errorf("error closing node: %v", err)
		}
	}()
	w.WriteHeader(http.StatusOK)
	eg := errgroup.Group{}
	eg.Go(func() error {
		var msg inet256.Message
		for {
			if err := inet256.Receive(ctx, node, &msg); err != nil {
				return err
			}
			if err := writeMessage(w, msg); err != nil {
				return err
			}
			flusher.Flush()
		}
	})
	eg.Go(func() error {
		buf := make([]byte, inet256.MaxMTU)
		for {
			var src, dst inet256.Addr
			n, err := readMessage(r.Body, &src, &dst, buf)
			if err != nil {
				return err
			}
			if err := node.Tell(ctx, dst, buf[:n]); err != nil {
				return err
			}
		}
	})
	if err := eg.Wait(); err != nil {
		logrus.Error(err)
	}
}

func writeError(w http.ResponseWriter, status int, err error) {
	w.WriteHeader(status)
	w.Write([]byte(err.Error()))
}
