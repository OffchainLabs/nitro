/*
 * Copyright 2021-2022, Offchain Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package wsbroadcastserver

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"nhooyr.io/websocket/wsjson"
	"sync"
	"time"

	"nhooyr.io/websocket"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/util"
)

// MaxSendQueue is the maximum number of items in a clients out channel before client gets disconnected.
// If set too low, a burst of items will cause all clients to be disconnected
const MaxSendQueue = 1000

type BroadcasterConfig struct {
	Addr          string
	IOTimeout     time.Duration
	Port          string
	Ping          time.Duration
	ClientTimeout time.Duration
	Queue         int
	Workers       int
}

var DefaultBroadcasterConfig BroadcasterConfig

type WSBroadcastServer struct {
	util.StopWaiter
	srv      *http.Server
	listener net.Listener
	settings BroadcasterConfig

	subscribersMu sync.Mutex
	subscribers   map[*ClientConnection]struct{}

	catchupBuffer CatchupBuffer
}

// ClientConnection represents a subscriber.
// Messages are sent on the msgs channel and if the client
// cannot keep up with the messages, closeSlow is called.
type ClientConnection struct {
	util.StopWaiter

	conn *websocket.Conn
	Name string

	pingTimeout time.Duration
	msgs        chan []byte
	closeSlow   func()
}

func (cc *ClientConnection) Start(ctx context.Context) {
	cancelableContext, cancel := context.WithCancel(ctx)
	defer cancel()
	cc.StopWaiter.Start(cancelableContext)
	cc.CallIteratively(func(ctx context.Context) time.Duration {
		pingContext, pingContextCancel := context.WithTimeout(ctx, cc.pingTimeout)
		defer pingContextCancel()
		err := cc.conn.Ping(pingContext)
		if err != nil {
			cc.closeSlow()
			cancel()
		}
		return cc.pingTimeout
	})
}

func (cc *ClientConnection) Write(x interface{}) error {
	return wsjson.Write(context.Background(), cc.conn, x)
}

// CatchupBuffer implemented protocol-specific client catch-up logic
type CatchupBuffer interface {
	OnRegisterClient(context.Context, *ClientConnection) error
	OnDoBroadcast(interface{}) error
}

func NewWSBroadcastServer(settings BroadcasterConfig, catchupBuffer CatchupBuffer) *WSBroadcastServer {
	return &WSBroadcastServer{
		settings:      settings,
		catchupBuffer: catchupBuffer,
		subscribers:   make(map[*ClientConnection]struct{}),
	}
}

func (s *WSBroadcastServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.clientHandler(w, r)
}

func (s *WSBroadcastServer) clientHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		log.Warn("failed initiating websocket connection", "err", err)
		return
	}
	defer func() {
		// Ignore errors when closing the websocket
		_ = conn.Close(websocket.StatusInternalError, "")
	}()

	client := &ClientConnection{
		conn: conn,
		Name: r.RemoteAddr,
		msgs: make(chan []byte, MaxSendQueue),
		closeSlow: func() {
			// Ignore errors when closing the websocket
			_ = conn.Close(websocket.StatusPolicyViolation, "connection too slow to keep up with messages")
		},
	}

	err = s.subscribe(r.Context(), client)
	if errors.Is(err, context.Canceled) {
		return
	}
	if websocket.CloseStatus(err) == websocket.StatusNormalClosure ||
		websocket.CloseStatus(err) == websocket.StatusGoingAway {
		return
	}
	if err != nil {
		log.Warn("websocket connection closed with error", "err", err)
		return
	}
}

// subscribe subscribes the given WebSocket to all broadcast messages.
// It creates a subscriber with a buffered msgs chan to give some room to slower
// connections and then registers the subscriber. It then listens for all messages
// and writes them to the WebSocket. If the context is cancelled or
// an error occurs, it returns and deletes the subscription.
//
// It uses CloseRead to keep reading from the connection to process control
// messages and cancel the context if the connection drops.
func (s *WSBroadcastServer) subscribe(ctx context.Context, c *ClientConnection) error {
	start := time.Now()
	err := func() error {
		registerCtx, cancel := context.WithTimeout(ctx, s.settings.ClientTimeout)
		defer cancel()
		return s.catchupBuffer.OnRegisterClient(registerCtx, c)
	}()
	if err != nil {
		return err
	}
	log.Info("client registered", "client", c.Name, "elapsed", time.Since(start))
	defer log.Info("client disconnected", "client", c.Name, "elapsed", time.Since(start))

	ctx = c.conn.CloseRead(ctx)
	s.addSubscriber(c)
	defer s.deleteSubscriber(c)

	for {
		select {
		case msg := <-c.msgs:
			err := func() error {
				ctx, cancel := context.WithTimeout(ctx, s.settings.ClientTimeout)
				defer cancel()
				return c.conn.Write(ctx, websocket.MessageText, msg)
			}()
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// addSubscriber registers a subscriber.
func (s *WSBroadcastServer) addSubscriber(c *ClientConnection) {
	s.subscribersMu.Lock()
	s.subscribers[c] = struct{}{}
	s.subscribersMu.Unlock()
}

// deleteSubscriber deletes the given subscriber.
func (s *WSBroadcastServer) deleteSubscriber(c *ClientConnection) {
	s.subscribersMu.Lock()
	delete(s.subscribers, c)
	s.subscribersMu.Unlock()
}

func (s *WSBroadcastServer) Start(ctx context.Context) error {
	s.StopWaiter.Start(ctx)
	ln, err := net.Listen("tcp", s.settings.Addr+":"+s.settings.Port)
	if err != nil {
		return err
	}
	s.srv = &http.Server{
		Handler:      s,
		ReadTimeout:  time.Second * 10,
		WriteTimeout: time.Second * 10,
	}
	s.listener = ln
	s.LaunchThread(func(ctx context.Context) {
		s.srv.BaseContext = func(net.Listener) context.Context {
			return ctx
		}
		if err := s.srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("shutting down broadcaster with error", "err", err)
		}
	})
	return nil
}

func (s *WSBroadcastServer) StopAndWait() {
	s.subscribersMu.Lock()
	for client := range s.subscribers {
		client.StopAndWait()
	}
	s.subscribersMu.Unlock()
	if err := s.srv.Shutdown(context.Background()); err != nil {
		log.Warn("error shutting down broadcaster", "err", err)
	}
	s.StopWaiter.StopAndWait()
}

// Broadcast sends batch item to all subscribers.
// It never blocks and so messages to slow subscribers
// are dropped.
func (s *WSBroadcastServer) Broadcast(bm interface{}) {
	if err := s.catchupBuffer.OnDoBroadcast(bm); err != nil {
		log.Error("error executing broadcast handler", "err", err)
		return
	}
	msg, err := json.Marshal(bm)
	if err != nil {
		log.Error("error to marshaling message for broadcast", "err", err)
		return
	}
	s.subscribersMu.Lock()
	defer s.subscribersMu.Unlock()

	for s := range s.subscribers {
		select {
		case s.msgs <- msg:
		default:
			go s.closeSlow()
		}
	}
}

func (s *WSBroadcastServer) ClientCount() int {
	return len(s.subscribers)
}

func (s *WSBroadcastServer) ListenerAddr() net.Addr {
	return s.listener.Addr()
}
