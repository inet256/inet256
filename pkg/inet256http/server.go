package inet256http

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/go-chi/chi"
	"github.com/inet256/inet256/pkg/inet256"
)

type Server struct {
	service inet256.Service
	router  chi.Router

	mu        sync.Mutex
	nodes     map[inet256.Addr]inet256.Node
	refCounts map[inet256.Addr]int
}

func NewServer(srv inet256.Service) *Server {
	router := chi.NewMux()
	s := &Server{
		service: srv,
		router:  router,
	}
	router.Route("/v0", func(r chi.Router) {
		r.Connect("/connect", s.connect)
		r.Get("/nodes/{id}", s.lookup)
		r.Get("/mtu/{id}", s.mtu)
		r.Get("/findAddr", s.findAddr)
	})
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) lookup(w http.ResponseWriter, r *http.Request) {

}

func (s *Server) findAddr(w http.ResponseWriter, r *http.Request) {

}

func (s *Server) mtu(w http.ResponseWriter, r *http.Request) {

}

func (s *Server) connect(w http.ResponseWriter, r *http.Request) {

}

func (s *Server) getNode(ctx context.Context, privateKey inet256.PrivateKey) (inet256.Node, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := inet256.NewAddr(privateKey.Public())
	n, exists := s.nodes[id]
	if !exists {
		var err error
		n, err = s.service.CreateNode(ctx, privateKey)
		if err != nil {
			return nil, err
		}
		s.nodes[id] = n
	}
	s.refCounts[id] += 1
	return n, nil
}

func (s *Server) dropNode(x inet256.Node) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := x.LocalAddr()
	s.refCounts[id] -= 1
	if s.refCounts[id] == 0 {
		delete(s.nodes, id)
		delete(s.refCounts, id)
	}
}

func readJSON(r io.ReadCloser, x interface{}) error {
	defer r.Close()
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &x)
}
