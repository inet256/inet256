package inet256http

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/brendoncarroll/stdctx/logctx"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256ipc"
	"github.com/inet256/inet256/pkg/rcsrv"
	"github.com/inet256/inet256/pkg/serde"
)

type Server struct {
	x inet256.Service
}

func NewServer(x inet256.Service) *Server {
	return &Server{
		x: rcsrv.Wrap(x),
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodConnect:
		if err := s.handleOpen(w, r); err != nil {
			logctx.Errorln(r.Context(), err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	case r.Method == http.MethodDelete:
	default:
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (s *Server) handleOpen(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	idStr := parts[len(parts)-1]
	addr, err := inet256.ParseAddrBase64([]byte(idStr))
	if err != nil {
		return err
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	var req OpenReq
	if err := json.Unmarshal(body, &req); err != nil {
		return err
	}
	privKey, err := serde.ParsePrivateKey(req.PrivateKey)
	if err != nil {
		return err
	}
	if inet256.NewAddr(privKey.Public()) != addr {
		return errors.New("private key does not match address")
	}
	node, err := s.x.Open(ctx, privKey)
	if err != nil {
		return err
	}
	defer node.Close()
	w.WriteHeader(http.StatusOK)

	h, ok := w.(http.Hijacker)
	if !ok {
		return nil
	}
	conn, bwr, err := h.Hijack()
	if err != nil {
		return err
	}
	defer conn.Close()
	if err := bwr.Writer.Flush(); err != nil {
		return err
	}
	fr := inet256ipc.NewStreamFramer(bwr.Reader, conn)
	logctx.Infof(ctx, "serving node %v", node.LocalAddr())
	defer logctx.Infof(ctx, "done serving node %v", node.LocalAddr())
	return inet256ipc.ServeNode(ctx, node, fr)
}
