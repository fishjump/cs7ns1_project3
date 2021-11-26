package p2pserver

import (
	"net/http"
)

func (s *P2PServer) healthz(rw http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()

	rw.WriteHeader(http.StatusOK)
}
