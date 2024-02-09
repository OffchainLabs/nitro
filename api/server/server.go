// Package server defines the client-facing API methods for fetching data
// related to BOLD challenges. It handles HTTP methods with their requests and responses.
package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/OffchainLabs/bold/api/backend"
	"github.com/gorilla/mux"
)

var apiVersion = "/api/v1"

type Server struct {
	srv        *http.Server
	router     *mux.Router
	registered bool
	backend    backend.BusinessLogicProvider
}

func New(addr string, backend backend.BusinessLogicProvider) (*Server, error) {
	if addr == "" {
		addr = ":8080"
	}
	r := mux.NewRouter()

	s := &Server{
		backend: backend,
		srv: &http.Server{
			Handler:           r,
			Addr:              addr,
			WriteTimeout:      15 * time.Second,
			ReadTimeout:       30 * time.Second,
			ReadHeaderTimeout: 30 * time.Second,
		},
		router: r,
	}
	if err := s.registerMethods(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Server) Start(ctx context.Context) error {
	return s.srv.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}

func (s *Server) registerMethods() error {
	if s.registered {
		return errors.New("API server methods already registered")
	}

	r := s.router.PathPrefix(apiVersion).Subrouter()
	r.HandleFunc("/healthz", s.Healthz).Methods("GET")
	r.HandleFunc("/assertions", s.ListAssertions).Methods("GET")
	r.HandleFunc("/assertions/{identifier}", s.AssertionByIdentifier).Methods("GET")
	r.HandleFunc("/challenge/{assertion-hash}/edges", s.AllChallengeEdges).Methods("GET")
	r.HandleFunc("/challenge/{assertion-hash}/edges/id/{edge-id}", s.EdgeByIdentifier).Methods("GET")
	r.HandleFunc("/challenge/{assertion-hash}/edges/history/{history-commitment}", s.EdgeByHistoryCommitment).Methods("GET")
	r.HandleFunc("/challenge/{assertion-hash}/ministakes", s.MiniStakes).Methods("GET")
	r.HandleFunc("/tracked/royal-edges", s.RoyalTrackedChallengeEdges).Methods("GET")
	r.HandleFunc("/state-provider/requests/collect-machine-hashes", s.CollectMachineHashes).Methods("GET")
	s.registered = true
	return nil
}
