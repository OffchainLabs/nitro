package api

import "github.com/gorilla/mux"

func (s *Server) Router() *mux.Router {
	return s.router
}

func (s *Server) Registered() bool {
	return s.registered
}
