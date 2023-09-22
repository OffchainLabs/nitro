package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/broadcaster/http/backlog"
)

type HTTPBroadcastServer struct {
	server           *http.Server
	serverExitedChan chan interface{}
	serverError      error
}

func NewHTTPBroadcastServer(httpBacklog backlog.Backlog) *HTTPBroadcastServer {
	handler := &BroadcastHandler{
		httpBacklog: httpBacklog,
	}

	server := &http.Server{
		Addr:              ":54321",
		Handler:           handler,
		ReadTimeout:       30 * time.Second,
		ReadHeaderTimeout: 30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	return &HTTPBroadcastServer{
		server:           server,
		serverExitedChan: make(chan interface{}),
		serverError:      nil,
	}
}

func (s *HTTPBroadcastServer) Start() {
	go func() {
		err := s.server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.serverError = err
		}
		close(s.serverExitedChan)
	}()
}

func (s *HTTPBroadcastServer) StopAndWait() error {
	err := s.server.Close()
	if err != nil {
		return err
	}
	<-s.serverExitedChan
	if s.serverError != nil {
		return err
	}
	return nil
}

type BroadcastHandler struct {
	httpBacklog backlog.Backlog
}

func (h *BroadcastHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestPath := path.Clean(r.URL.Path)
	log.Debug("received HTTP request", "requestPath", requestPath)

	switch {
	case r.Method == http.MethodGet && requestPath == "/":
		h.getMessagesHandler(w, r)
	default:
		log.Warn("unknown request path sent to HTTP server", "requestPath", requestPath)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
}

func (h *BroadcastHandler) getMessagesHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Error("error parsing http request", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	s := r.FormValue("start")
	if s == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	start, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		log.Error("error converting sequence number to uint64", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	e := r.FormValue("end")
	if e == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	end, err := strconv.ParseUint(e, 10, 64)
	if err != nil {
		log.Error("error converting sequence number to uint64", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	bm, err := h.httpBacklog.Get(arbutil.MessageIndex(start), arbutil.MessageIndex(end))
	if err != nil {
		msg := fmt.Sprintf("error getting cached messages: %s", err)
		log.Error(msg)
		w.WriteHeader(http.StatusInternalServerError)
	}

	m, err := json.Marshal(bm)
	if err != nil {
		msg := fmt.Sprintf("error serializing message: %s", err)
		log.Error(msg)
		w.WriteHeader(http.StatusInternalServerError)
	}

	w.Write(m)
}
