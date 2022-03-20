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
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws-examples/src/gopool"
	"github.com/mailru/easygo/netpoll"
	flag "github.com/spf13/pflag"
)

type BroadcasterConfig struct {
	Enable        bool          `koanf:"enable"`
	Addr          string        `koanf:"addr"`
	IOTimeout     time.Duration `koanf:"io-timeout"`
	Port          string        `koanf:"port"`
	Ping          time.Duration `koanf:"ping"`
	ClientTimeout time.Duration `koanf:"client-timeout"`
	Queue         int           `koanf:"queue"`
	Workers       int           `koanf:"workers"`
}

func BroadcasterConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultBroadcasterConfig.Enable, "enable broadcaster")
	f.String(prefix+".addr", DefaultBroadcasterConfig.Addr, "address to bind the relay feed output to")
	f.Duration(prefix+".io-timeout", DefaultBroadcasterConfig.IOTimeout, "duration to wait before timing out HTTP to WS upgrade")
	f.String(prefix+".port", DefaultBroadcasterConfig.Port, "port to bind the relay feed output to")
	f.Duration(prefix+".ping", DefaultBroadcasterConfig.Ping, "duration for ping interval")
	f.Duration(prefix+".client-timeout", DefaultBroadcasterConfig.ClientTimeout, "duration to wait before timing out connections to client")
	f.Int(prefix+".queue", DefaultBroadcasterConfig.Queue, "queue size")
	f.Int(prefix+".workers", DefaultBroadcasterConfig.Workers, "number of threads to reserve for HTTP to WS upgrade")
}

var DefaultBroadcasterConfig = BroadcasterConfig{
	Enable:        false,
	Addr:          "",
	IOTimeout:     5 * time.Second,
	Port:          "9642",
	Ping:          5 * time.Second,
	ClientTimeout: 15 * time.Second,
	Queue:         100,
	Workers:       100,
}

type WSBroadcastServer struct {
	startMutex    *sync.Mutex
	poller        netpoll.Poller
	acceptDesc    *netpoll.Desc
	listener      net.Listener
	settings      BroadcasterConfig
	started       bool
	clientManager *ClientManager
	catchupBuffer CatchupBuffer
}

func NewWSBroadcastServer(settings BroadcasterConfig, catchupBuffer CatchupBuffer) *WSBroadcastServer {
	return &WSBroadcastServer{
		startMutex:    &sync.Mutex{},
		settings:      settings,
		started:       false,
		catchupBuffer: catchupBuffer,
	}
}

func (s *WSBroadcastServer) Start(ctx context.Context) error {
	s.startMutex.Lock()
	defer s.startMutex.Unlock()
	if s.started {
		return nil
	}

	var err error
	s.poller, err = netpoll.New(nil)
	if err != nil {
		log.Error("unable to initialize netpoll for monitoring client connection events", "err", err)
		return err
	}

	// Make pool of X size, Y sized work queue and one pre-spawned
	// goroutine.
	var clientManager = NewClientManager(s.poller, s.settings, s.catchupBuffer)
	clientManager.Start(ctx)

	s.clientManager = clientManager // maintain the pointer in this instance... used for testing

	// handle incoming connection requests.
	// It upgrades TCP connection to WebSocket, registers netpoll listener on
	// it and stores it as a Client connection in ClientManager instance.
	//
	// Called below in accept() loop.
	handle := func(conn net.Conn) {

		safeConn := deadliner{conn, s.settings.IOTimeout}

		// Zero-copy upgrade to WebSocket connection.
		hs, err := ws.Upgrade(safeConn)
		if err != nil {
			log.Warn("websocket upgrade error", "connection_name", nameConn(safeConn), "err", err)
			_ = safeConn.Close()
			return
		}

		log.Info(fmt.Sprintf("established websocket connection: %+v", hs), "connection-name", nameConn(safeConn))

		// Create netpoll event descriptor to handle only read events.
		desc, err := netpoll.HandleRead(conn)
		if err != nil {
			log.Warn("error in HandleRead", "connection-name", nameConn(safeConn), "err", err)
			_ = conn.Close()
			return
		}

		// Register incoming client in clientManager.
		client := clientManager.Register(safeConn, desc)

		// Subscribe to events about conn.
		err = s.poller.Start(desc, func(ev netpoll.Event) {
			if ev&(netpoll.EventReadHup|netpoll.EventHup) != 0 {
				// ReadHup or Hup received, means the client has close the connection
				// remove it from the clientManager registry.
				log.Info("Hup received", "connection_name", nameConn(safeConn))
				clientManager.Remove(client)
				return
			}

			if ev > 1 {
				log.Info("event greater than 1 received", "connection_name", nameConn(safeConn), "event", int(ev))
			}

			// receive client messages, close on error
			clientManager.pool.Schedule(func() {
				// Ignore any messages sent from client
				if _, _, err := client.Receive(ctx, s.settings.ClientTimeout); err != nil {
					log.Warn("receive error", "connection_name", nameConn(safeConn), "err", err)
					clientManager.Remove(client)
					return
				}
			})
		})

		if err != nil {
			log.Warn("error starting client connection poller", "err", err)
		}
	}

	// Create tcp server for relay connections
	ln, err := net.Listen("tcp", s.settings.Addr+":"+s.settings.Port)
	if err != nil {
		log.Error("error calling net.Listen", "err", err)
		return err
	}

	s.listener = ln

	log.Info("arbitrum websocket broadcast server is listening", "address", ln.Addr().String())

	// Create netpoll descriptor for the listener.
	// We use OneShot here to manually resume events stream when we want to.
	acceptDesc, err := netpoll.HandleListener(ln, netpoll.EventRead|netpoll.EventOneShot)
	if err != nil {
		log.Error("error calling HandleListener", "err", err)
		return err
	}
	s.acceptDesc = acceptDesc

	// accept is a channel to signal about next incoming connection Accept()
	// results.
	accept := make(chan error, 1)

	// Subscribe to events about listener.
	err = s.poller.Start(acceptDesc, func(e netpoll.Event) {
		// We do not want to accept incoming connection when goroutine pool is
		// busy. So if there are no free goroutines during 1ms we want to
		// cooldown the server and do not receive connection for some short
		// time.
		err := clientManager.pool.ScheduleTimeout(time.Millisecond, func() {
			conn, err := ln.Accept()
			if err != nil {
				accept <- err
				return
			}

			accept <- nil
			handle(conn)
		})
		if err == nil {
			err = <-accept
		}
		if err != nil {
			if errors.Is(err, gopool.ErrScheduleTimeout) {
				var netError net.Error
				success := errors.As(err, &netError)
				if !success || !netError.Temporary() {
					log.Error("error in poller.Start", "err", err)
					return
				}
				if strings.Contains(err.Error(), "file descriptor was not registered") {
					log.Info("poller exiting", "err", err)
					return
				}
			}

			// cooldown
			delay := 5 * time.Millisecond
			log.Info("accept error", "delay", delay.String(), "err", err)
			time.Sleep(delay)
		}

		err = s.poller.Resume(acceptDesc)
		if err != nil {
			log.Warn("error in poller.Resume", "err", err)
		}
	})
	if err != nil {
		log.Warn("error in poller.Start", "err", err)
	}

	s.started = true

	return nil
}

func (s *WSBroadcastServer) ListenerAddr() net.Addr {
	return s.listener.Addr()
}

func (s *WSBroadcastServer) StopAndWait() {
	err := s.listener.Close()
	if err != nil {
		log.Warn("error in listener.Close", "err", err)
	}

	err = s.poller.Stop(s.acceptDesc)
	if err != nil {
		log.Warn("error in poller.Stop", "err", err)
	}

	err = s.acceptDesc.Close()
	if err != nil {
		log.Warn("error in acceptDesc.Close", "err", err)
	}

	s.clientManager.StopAndWait()
	s.started = false
}

// Broadcast sends batch item to all clients.
func (s *WSBroadcastServer) Broadcast(bm interface{}) {
	s.clientManager.Broadcast(bm)
}

func (s *WSBroadcastServer) ClientCount() int32 {
	return s.clientManager.ClientCount()
}

// deadliner is a wrapper around net.Conn that sets read/write deadlines before
// every Read() or Write() call.
type deadliner struct {
	net.Conn
	t time.Duration
}

func (d deadliner) Write(p []byte) (int, error) {
	if err := d.Conn.SetWriteDeadline(time.Now().Add(d.t)); err != nil {
		return 0, err
	}
	return d.Conn.Write(p)
}

func (d deadliner) Read(p []byte) (int, error) {
	if err := d.Conn.SetReadDeadline(time.Now().Add(d.t)); err != nil {
		return 0, err
	}
	return d.Conn.Read(p)
}

func nameConn(conn net.Conn) string {
	return conn.LocalAddr().String() + " > " + conn.RemoteAddr().String()
}
