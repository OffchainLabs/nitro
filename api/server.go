// Package api defines an API server for BOLD, allowig retrieval of information
// from both the assertion chain and the challenge manager contracts in order to
// understand ongoing challenges.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

var (
	ErrNoConfig                 = errors.New("no config provided")
	ErrNoDataAccessor           = errors.New("no data accessor provided")
	ErrAlreadyRegisteredMethods = errors.New("already registered methods")
)

type Config struct {
	Address      string
	DataAccessor DataAccessor
}

type Server struct {
	srv *http.Server

	data   DataAccessor
	router *mux.Router

	registered bool
}

func NewServer(cfg *Config) (*Server, error) {
	if cfg == nil {
		return nil, ErrNoConfig
	}

	if cfg.Address == "" {
		cfg.Address = ":8080"
	}

	if cfg.DataAccessor == nil {
		return nil, ErrNoDataAccessor
	}

	r := mux.NewRouter()

	s := &Server{
		srv: &http.Server{
			Handler:           r,
			Addr:              cfg.Address,
			WriteTimeout:      15 * time.Second,
			ReadTimeout:       15 * time.Second,
			ReadHeaderTimeout: 15 * time.Second,
		},
		data:   cfg.DataAccessor,
		router: r,
	}

	if err := s.registerMethods(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Server) Start() error {
	return s.srv.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}

func (s *Server) registerMethods() error {
	if s.registered {
		return ErrAlreadyRegisteredMethods
	}

	s.router.HandleFunc("/healthz", healthzHandler).Methods("GET")

	// Assertions
	s.router.HandleFunc("/assertions", listAssertionsHandler).Methods("GET")
	s.router.HandleFunc("/assertions/{id}", getAssertionHandler).Methods("GET")

	// Edges
	s.router.HandleFunc("/edges", s.listEdgesHandler).Methods("GET") // TODO: Query params like IsChallenge?
	s.router.HandleFunc("/edges/{id}", s.getEdgeHandler).Methods("GET")

	s.registered = true
	return nil
}

func writeJSONResponse(w http.ResponseWriter, code int, data any) error {
	body, err := json.Marshal(data)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, err = w.Write(body)
	return err
}

func writeError(w http.ResponseWriter, code int, err error) {
	w.WriteHeader(code)
	if _, err2 := w.Write([]byte(err.Error())); err != nil {
		log.Error("failed to write response body", "err", err2, "status", code)
	}
}
