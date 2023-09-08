package server

import (
	"errors"
	"net/http"
	"path"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbutil"
)

// CatchupBuffer is a Protocol-specific client catch-up logic can be injected using this interface
type CatchupBuffer interface {
	OnHTTPRequest(http.ResponseWriter, arbutil.MessageIndex)
}

type HTTPBroadcastServer struct {
	server           *http.Server
	serverExitedChan chan interface{}
	serverError      error
}

func NewHTTPBroadcastServer(catchupBuffer CatchupBuffer) *HTTPBroadcastServer {
	handler := &BroadcastHandler{
		catchupBuffer: catchupBuffer,
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
	catchupBuffer CatchupBuffer
}

func (h *BroadcastHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestPath := path.Clean(r.URL.Path)
	log.Debug("received HTTP request", "requestPath", requestPath)

	switch {
	case requestPath == "/":
		h.bufferMessagesHandler(w, r)
	default:
		log.Warn("unknown request path sent to HTTP server", "requestPath", requestPath)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
}

func (h *BroadcastHandler) bufferMessagesHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Error("error parsing http request", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	val := r.FormValue("sequence_number")
	if val == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	requestedSeqNum, err := strconv.ParseUint(val, 10, 64)
	if err != nil {
		log.Error("error converting sequence number to uint64", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	h.catchupBuffer.OnHTTPRequest(w, arbutil.MessageIndex(requestedSeqNum))
}

//func serializeMessage(bm interface{}) (bytes.Buffer, error) {
//	var notCompressed bytes.Buffer
//	notCompressedWriter := wsutil.NewWriter(&notCompressed, ws.StateServerSide, ws.OpText)
//	encoder := json.NewEncoder(notCompressedWriter)
//
//	if err := encoder.Encode(bm); err != nil {
//		return bytes.Buffer{}, fmt.Errorf("unable to encode message: %w", err)
//	}
//
//	if err := notCompressedWriter.Flush(); err != nil {
//		return bytes.Buffer{}, fmt.Errorf("unable to flush message: %w", err)
//	}
//
//	return notCompressed, nil
//}
