// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package transactionfeed

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"

	"github.com/offchainlabs/nitro/util/stopwaiter"
)

var (
	clientsCurrentGauge     = metrics.NewRegisteredGauge("arb/transactionfeed/clients/current", nil)
	clientsDisconnectedSlow = metrics.NewRegisteredCounter("arb/transactionfeed/clients/disconnected/slow", nil)
	broadcastDroppedCounter = metrics.NewRegisteredCounter("arb/transactionfeed/broadcast/dropped", nil)
)

// broadcastDropLogInterval is the minimum gap between successive Warn logs
// for dropped broadcasts. Keeps log volume bounded when a slow run loop or
// pathological client wave causes a sustained drop streak; the
// broadcastDroppedCounter still reflects every drop.
const broadcastDropLogInterval = time.Minute

// registerChanBuf and unregisterChanBuf absorb bursts of clients connecting
// or disconnecting while the run loop is busy broadcasting.
const (
	registerChanBuf   = 16
	unregisterChanBuf = 64
)

type clientConn struct {
	conn       *websocket.Conn
	remoteAddr string
	out        chan []byte
}

type Server struct {
	stopwaiter.StopWaiter
	config              ServerConfig
	listener            net.Listener
	httpServer          *http.Server
	register         chan *clientConn
	unregister       chan *clientConn
	broadcast        chan []byte
	clientCount      atomic.Int32
	lastDropLogNanos atomic.Int64
}

func NewServer(config ServerConfig) *Server {
	return &Server{
		config:     config,
		register:   make(chan *clientConn, registerChanBuf),
		unregister: make(chan *clientConn, unregisterChanBuf),
		broadcast:  make(chan []byte, config.BroadcastBuf),
	}
}

func (s *Server) Start(ctx context.Context) error {
	addr := net.JoinHostPort(s.config.Addr, s.config.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	s.listener = ln
	s.StopWaiter.Start(ctx, s)

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleWS)
	s.httpServer = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: s.config.HandshakeTimeout,
	}

	s.LaunchThread(func(_ context.Context) {
		err := s.httpServer.Serve(ln)
		if errors.Is(err, http.ErrServerClosed) {
			return
		}
		log.Error("Transaction feed http serve exited unexpectedly", "err", err)
		s.StopOnly()
	})
	s.LaunchThread(s.run)
	log.Info("Transaction feed server listening", "addr", addr)
	return nil
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
	if err != nil {
		log.Warn("Transaction feed ws upgrade error", "err", err, "remote", r.RemoteAddr)
		return
	}

	cc := &clientConn{
		conn:       conn,
		remoteAddr: r.RemoteAddr,
		out:        make(chan []byte, s.config.ClientBuf),
	}

	ctx := s.GetContext()

	select {
	case s.register <- cc:
	case <-ctx.Done():
		_ = conn.Close(websocket.StatusGoingAway, "shutting down")
		return
	}

	log.Debug("Transaction feed client connected", "remote", cc.remoteAddr)

	readCtx := cc.conn.CloseRead(ctx)
	s.clientWriter(ctx, readCtx, cc)

	log.Debug("Transaction feed client disconnected", "remote", cc.remoteAddr)

	_ = conn.Close(websocket.StatusNormalClosure, "")
	s.sendUnregister(cc)
}

func (s *Server) clientWriter(ctx context.Context, readCtx context.Context, cc *clientConn) {
	ticker := time.NewTicker(s.config.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case data, ok := <-cc.out:
			if !ok {
				return
			}
			wctx, cancel := context.WithTimeout(ctx, s.config.WriteTimeout)
			err := cc.conn.Write(wctx, websocket.MessageBinary, data)
			cancel()
			if err != nil {
				log.Debug("Transaction feed client write failed", "err", err, "remote", cc.remoteAddr)
				return
			}

		case <-ticker.C:
			pctx, cancel := context.WithTimeout(ctx, s.config.WriteTimeout)
			err := cc.conn.Ping(pctx)
			cancel()
			if err != nil {
				log.Debug("Transaction feed client ping failed", "err", err, "remote", cc.remoteAddr)
				return
			}

		case <-readCtx.Done():
			return

		case <-ctx.Done():
			return
		}
	}
}

func (s *Server) run(ctx context.Context) {
	clients := make(map[*clientConn]struct{})
	defer func() {
		for cc := range clients {
			close(cc.out)
			_ = cc.conn.CloseNow()
		}
		s.clientCount.Store(0)
		clientsCurrentGauge.Update(0)
	}()

	for {
		select {
		case cc := <-s.register:
			clients[cc] = struct{}{}
			s.clientCount.Add(1)
			clientsCurrentGauge.Update(int64(s.clientCount.Load()))

		case cc := <-s.unregister:
			if _, ok := clients[cc]; ok {
				delete(clients, cc)
				close(cc.out)
				s.clientCount.Add(-1)
				clientsCurrentGauge.Update(int64(s.clientCount.Load()))
			}

		case data := <-s.broadcast:
			for cc := range clients {
				select {
				case cc.out <- data:
				default:
					log.Warn("Transaction feed client disconnected due to slow consumption", "remote", cc.remoteAddr)
					clientsDisconnectedSlow.Inc(1)
					delete(clients, cc)
					close(cc.out)
					_ = cc.conn.CloseNow()
					s.clientCount.Add(-1)
					clientsCurrentGauge.Update(int64(s.clientCount.Load()))
				}
			}

		case <-ctx.Done():
			return
		}
	}
}

func (s *Server) sendUnregister(cc *clientConn) {
	select {
	case s.unregister <- cc:
	case <-s.GetContext().Done():
	}
}

func (s *Server) BroadcastTransaction(msg *TransactionFeedMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Error("Failed to marshal transaction feed message", "err", err)
		return
	}

	select {
	case s.broadcast <- data:
	default:
		broadcastDroppedCounter.Inc(1)
		s.maybeLogBroadcastDrop()
	}
}

// maybeLogBroadcastDrop emits at most one Warn log per broadcastDropLogInterval
// to surface sustained drop pressure without spamming the log.
func (s *Server) maybeLogBroadcastDrop() {
	now := time.Now().UnixNano()
	last := s.lastDropLogNanos.Load()
	if now-last < int64(broadcastDropLogInterval) {
		return
	}
	if !s.lastDropLogNanos.CompareAndSwap(last, now) {
		return
	}
	log.Warn("Transaction feed broadcast channel full -- message dropped",
		"totalDropped", broadcastDroppedCounter.Snapshot().Count(),
		"clients", s.clientCount.Load(),
		"size", s.config.BroadcastBuf)
}

func (s *Server) ClientCount() int32 {
	return s.clientCount.Load()
}

func (s *Server) ListenerAddr() net.Addr {
	if s.listener == nil {
		return nil
	}
	return s.listener.Addr()
}

func (s *Server) StopAndWait() {
	if s.httpServer != nil {
		_ = s.httpServer.Close()
	}
	if s.listener != nil {
		_ = s.listener.Close()
	}
	s.StopWaiter.StopAndWait()
}
