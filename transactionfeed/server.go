// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package transactionfeed

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"sync/atomic"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"

	"github.com/offchainlabs/nitro/util/stopwaiter"
)

var (
	clientsCurrentGauge     = metrics.NewRegisteredGauge("arb/transactionfeed/clients/current", nil)
	clientsDisconnectedSlow = metrics.NewRegisteredCounter("arb/transactionfeed/clients/disconnected/slow", nil)
	broadcastDroppedCounter = metrics.NewRegisteredCounter("arb/transactionfeed/broadcast/dropped", nil)
)

type clientConn struct {
	conn net.Conn
	out  chan []byte
}

type Server struct {
	stopwaiter.StopWaiter
	config      ServerConfig
	listener    net.Listener
	register    chan net.Conn
	unregister  chan net.Conn
	broadcast   chan []byte
	clientCount atomic.Int32
}

func NewServer(config ServerConfig) *Server {
	return &Server{
		config: config,
		// register is small-buffered so handshake goroutines do not stall
		// during broadcast fan-out in run().
		register:   make(chan net.Conn, 16),
		unregister: make(chan net.Conn, 64),
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

	s.LaunchThread(s.acceptLoop)
	s.LaunchThread(s.run)
	log.Info("Transaction feed server listening", "addr", addr)
	return nil
}

func (s *Server) acceptLoop(ctx context.Context) {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			select {
			case <-ctx.Done():
				return
			default:
			}
			log.Warn("Transaction feed accept error", "err", err)
			continue
		}
		s.LaunchThread(func(ctx context.Context) {
			s.handleHandshake(ctx, conn)
		})
	}
}

func (s *Server) handleHandshake(ctx context.Context, conn net.Conn) {
	if err := conn.SetReadDeadline(time.Now().Add(s.config.HandshakeTimeout)); err != nil {
		conn.Close()
		return
	}
	if _, err := ws.Upgrade(conn); err != nil {
		log.Warn("Transaction feed ws upgrade error", "err", err)
		conn.Close()
		return
	}
	// Clear the handshake deadline; clientReader manages its own.
	if err := conn.SetReadDeadline(time.Time{}); err != nil {
		conn.Close()
		return
	}
	select {
	case s.register <- conn:
	case <-ctx.Done():
		conn.Close()
	}
}

func (s *Server) run(ctx context.Context) {
	clients := make(map[net.Conn]*clientConn)
	defer func() {
		for c, cc := range clients {
			close(cc.out)
			c.Close()
		}
		// All clients are gone on shutdown; reset the counter/gauge so
		// post-stop reads don't observe stale non-zero values.
		s.clientCount.Store(0)
		clientsCurrentGauge.Update(0)
	}()

	for {
		select {
		case conn := <-s.register:
			cc := &clientConn{
				conn: conn,
				out:  make(chan []byte, s.config.ClientBuf),
			}
			clients[conn] = cc
			s.clientCount.Add(1)
			clientsCurrentGauge.Update(int64(s.clientCount.Load()))
			s.LaunchThread(func(ctx context.Context) {
				s.clientWriter(ctx, cc)
			})
			s.LaunchThread(func(ctx context.Context) {
				s.clientReader(ctx, cc)
			})

		case conn := <-s.unregister:
			if cc, ok := clients[conn]; ok {
				close(cc.out)
				conn.Close()
				delete(clients, conn)
				s.clientCount.Add(-1)
				clientsCurrentGauge.Update(int64(s.clientCount.Load()))
			}

		case data := <-s.broadcast:
			for conn, cc := range clients {
				select {
				case cc.out <- data:
				default:
					clientsDisconnectedSlow.Inc(1)
					close(cc.out)
					conn.Close()
					delete(clients, conn)
					s.clientCount.Add(-1)
					clientsCurrentGauge.Update(int64(s.clientCount.Load()))
				}
			}

		case <-ctx.Done():
			return
		}
	}
}

func (s *Server) clientWriter(ctx context.Context, cc *clientConn) {
	ticker := time.NewTicker(s.config.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case data, ok := <-cc.out:
			if !ok {
				return
			}
			if err := cc.conn.SetWriteDeadline(time.Now().Add(s.config.WriteTimeout)); err != nil {
				s.sendUnregister(cc.conn)
				return
			}
			if err := wsutil.WriteServerText(cc.conn, data); err != nil {
				s.sendUnregister(cc.conn)
				return
			}

		case <-ticker.C:
			if err := cc.conn.SetWriteDeadline(time.Now().Add(s.config.WriteTimeout)); err != nil {
				s.sendUnregister(cc.conn)
				return
			}
			if err := wsutil.WriteServerMessage(cc.conn, ws.OpPing, nil); err != nil {
				s.sendUnregister(cc.conn)
				return
			}

		case <-ctx.Done():
			return
		}
	}
}

func (s *Server) clientReader(ctx context.Context, cc *clientConn) {
	controlHandler := wsutil.ControlFrameHandler(cc.conn, ws.StateServerSide)
	reader := &wsutil.Reader{
		Source:         cc.conn,
		State:          ws.StateServerSide,
		OnIntermediate: controlHandler,
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if err := cc.conn.SetReadDeadline(time.Now().Add(s.config.ClientTimeout)); err != nil {
			s.sendUnregister(cc.conn)
			return
		}
		hdr, err := reader.NextFrame()
		if err != nil {
			s.sendUnregister(cc.conn)
			return
		}
		if hdr.OpCode.IsControl() {
			if err := controlHandler(hdr, reader); err != nil {
				s.sendUnregister(cc.conn)
				return
			}
			continue
		}
		// Discard any data frames from clients
		if err := reader.Discard(); err != nil {
			s.sendUnregister(cc.conn)
			return
		}
	}
}

func (s *Server) sendUnregister(conn net.Conn) {
	conn.Close()
	select {
	case s.unregister <- conn:
	default:
	}
}

func (s *Server) BroadcastTransaction(msg *TransactionFeedMessage) {
	if msg == nil {
		return
	}
	data, err := json.Marshal(msg)
	if err != nil {
		log.Error("Failed to marshal transaction feed message", "err", err)
		return
	}

	select {
	case s.broadcast <- data:
	default:
		broadcastDroppedCounter.Inc(1)
	}
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
	if s.listener != nil {
		s.listener.Close()
	}
	s.StopWaiter.StopAndWait()
}
