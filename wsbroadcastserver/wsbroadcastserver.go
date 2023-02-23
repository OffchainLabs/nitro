// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package wsbroadcastserver

import (
	"context"
	"crypto/rand"
	"fmt"
	"net"
	"net/http"
	"net/textproto"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gobwas/httphead"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws-examples/src/gopool"
	"github.com/gobwas/ws/wsflate"
	"github.com/mailru/easygo/netpoll"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbutil"
)

var (
	HTTPHeaderCloudflareConnectingIP  = textproto.CanonicalMIMEHeaderKey("CF-Connecting-IP")
	HTTPHeaderFeedServerVersion       = textproto.CanonicalMIMEHeaderKey("Arbitrum-Feed-Server-Version")
	HTTPHeaderFeedClientVersion       = textproto.CanonicalMIMEHeaderKey("Arbitrum-Feed-Client-Version")
	HTTPHeaderRequestedSequenceNumber = textproto.CanonicalMIMEHeaderKey("Arbitrum-Requested-Sequence-Number")
	HTTPHeaderChainId                 = textproto.CanonicalMIMEHeaderKey("Arbitrum-Chain-Id")
	HTTPHeaderNonce                   = textproto.CanonicalMIMEHeaderKey("Arbitrum-Relay-Client-Nonce")
)

const (
	FeedServerVersion = 2
	FeedClientVersion = 2
	LivenessProbeURI  = "livenessprobe"
)

type BroadcasterConfig struct {
	Enable             bool                    `koanf:"enable"`
	Signed             bool                    `koanf:"signed"`
	Addr               string                  `koanf:"addr"`
	ReadTimeout        time.Duration           `koanf:"read-timeout" reload:"hot"`      // reloaded value will affect all clients (next time the timeout is checked)
	WriteTimeout       time.Duration           `koanf:"write-timeout" reload:"hot"`     // reloading will affect only new connections
	HandshakeTimeout   time.Duration           `koanf:"handshake-timeout" reload:"hot"` // reloading will affect only new connections
	Port               string                  `koanf:"port"`
	Ping               time.Duration           `koanf:"ping" reload:"hot"`           // reloaded value will change future ping intervals
	ClientTimeout      time.Duration           `koanf:"client-timeout" reload:"hot"` // reloaded value will affect all clients (next time the timeout is checked)
	Queue              int                     `koanf:"queue"`
	Workers            int                     `koanf:"workers"`
	MaxSendQueue       int                     `koanf:"max-send-queue" reload:"hot"`  // reloaded value will affect only new connections
	RequireVersion     bool                    `koanf:"require-version" reload:"hot"` // reloaded value will affect only future upgrades to websocket
	DisableSigning     bool                    `koanf:"disable-signing"`
	LogConnect         bool                    `koanf:"log-connect"`
	LogDisconnect      bool                    `koanf:"log-disconnect"`
	EnableCompression  bool                    `koanf:"enable-compression" reload:"hot"`  // if reloaded to false will cause disconnection of clients with enabled compression on next broadcast
	RequireCompression bool                    `koanf:"require-compression" reload:"hot"` // if reloaded to true will cause disconnection of clients with disabled compression on next broadcast
	LimitCatchup       bool                    `koanf:"limit-catchup" reload:"hot"`
	ConnectionLimits   ConnectionLimiterConfig `koanf:"connection-limits" reload:"hot"`
}

func (bc *BroadcasterConfig) Validate() error {
	if !bc.EnableCompression && bc.RequireCompression {
		return errors.New("require-compression cannot be true while enable-compression is false")
	}
	return nil
}

type BroadcasterConfigFetcher func() *BroadcasterConfig

func BroadcasterConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultBroadcasterConfig.Enable, "enable broadcaster")
	f.Bool(prefix+".signed", DefaultBroadcasterConfig.Signed, "sign broadcast messages")
	f.String(prefix+".addr", DefaultBroadcasterConfig.Addr, "address to bind the relay feed output to")
	f.Duration(prefix+".read-timeout", DefaultBroadcasterConfig.ReadTimeout, "duration to wait before timing out reading data (i.e. pings) from clients")
	f.Duration(prefix+".write-timeout", DefaultBroadcasterConfig.WriteTimeout, "duration to wait before timing out writing data to clients")
	f.Duration(prefix+".handshake-timeout", DefaultBroadcasterConfig.HandshakeTimeout, "duration to wait before timing out HTTP to WS upgrade")
	f.String(prefix+".port", DefaultBroadcasterConfig.Port, "port to bind the relay feed output to")
	f.Duration(prefix+".ping", DefaultBroadcasterConfig.Ping, "duration for ping interval")
	f.Duration(prefix+".client-timeout", DefaultBroadcasterConfig.ClientTimeout, "duration to wait before timing out connections to client")
	f.Int(prefix+".queue", DefaultBroadcasterConfig.Queue, "queue size")
	f.Int(prefix+".workers", DefaultBroadcasterConfig.Workers, "number of threads to reserve for HTTP to WS upgrade")
	f.Int(prefix+".max-send-queue", DefaultBroadcasterConfig.MaxSendQueue, "maximum number of messages allowed to accumulate before client is disconnected")
	f.Bool(prefix+".require-version", DefaultBroadcasterConfig.RequireVersion, "don't connect if client version not present")
	f.Bool(prefix+".disable-signing", DefaultBroadcasterConfig.DisableSigning, "don't sign feed messages")
	f.Bool(prefix+".log-connect", DefaultBroadcasterConfig.LogConnect, "log every client connect")
	f.Bool(prefix+".log-disconnect", DefaultBroadcasterConfig.LogDisconnect, "log every client disconnect")
	f.Bool(prefix+".enable-compression", DefaultBroadcasterConfig.EnableCompression, "enable per message deflate compression support")
	f.Bool(prefix+".require-compression", DefaultBroadcasterConfig.RequireCompression, "require clients to use compression")
	f.Bool(prefix+".limit-catchup", DefaultBroadcasterConfig.LimitCatchup, "only supply catchup buffer if requested sequence number is reasonable")
	ConnectionLimiterConfigAddOptions(prefix+".connection-limits", f)
}

var DefaultBroadcasterConfig = BroadcasterConfig{
	Enable:             false,
	Signed:             false,
	Addr:               "",
	ReadTimeout:        time.Second,
	WriteTimeout:       2 * time.Second,
	HandshakeTimeout:   time.Second,
	Port:               "9642",
	Ping:               5 * time.Second,
	ClientTimeout:      15 * time.Second,
	Queue:              100,
	Workers:            100,
	MaxSendQueue:       4096,
	RequireVersion:     false,
	DisableSigning:     true,
	LogConnect:         false,
	LogDisconnect:      false,
	EnableCompression:  true,
	RequireCompression: false,
	LimitCatchup:       false,
	ConnectionLimits:   DefaultConnectionLimiterConfig,
}

var DefaultTestBroadcasterConfig = BroadcasterConfig{
	Enable:             false,
	Signed:             false,
	Addr:               "0.0.0.0",
	ReadTimeout:        2 * time.Second,
	WriteTimeout:       2 * time.Second,
	HandshakeTimeout:   2 * time.Second,
	Port:               "0",
	Ping:               5 * time.Second,
	ClientTimeout:      15 * time.Second,
	Queue:              1,
	Workers:            100,
	MaxSendQueue:       4096,
	RequireVersion:     false,
	DisableSigning:     false,
	LogConnect:         false,
	LogDisconnect:      false,
	EnableCompression:  true,
	RequireCompression: false,
	LimitCatchup:       false,
	ConnectionLimits:   DefaultConnectionLimiterConfig,
}

type WSBroadcastServer struct {
	startMutex sync.Mutex
	poller     netpoll.Poller

	acceptDescMutex sync.Mutex
	acceptDesc      *netpoll.Desc

	pseudoRandomNonceMutex sync.Mutex
	lastPseudoRandomNonce  common.Hash

	listener      net.Listener
	config        BroadcasterConfigFetcher
	started       bool
	clientManager *ClientManager
	catchupBuffer CatchupBuffer
	chainId       uint64
	fatalErrChan  chan error
}

func NewWSBroadcastServer(config BroadcasterConfigFetcher, catchupBuffer CatchupBuffer, chainId uint64, fatalErrChan chan error) (*WSBroadcastServer, error) {
	var nonce common.Hash
	_, err := rand.Read(nonce[:])
	if err != nil {
		return nil, err
	}
	return &WSBroadcastServer{
		config:                config,
		started:               false,
		catchupBuffer:         catchupBuffer,
		chainId:               chainId,
		fatalErrChan:          fatalErrChan,
		lastPseudoRandomNonce: nonce,
	}, nil
}

func (s *WSBroadcastServer) Initialize() error {
	if s.poller != nil {
		return errors.New("broadcast server already initialized")
	}

	var err error
	s.poller, err = netpoll.New(nil)
	if err != nil {
		log.Error("unable to initialize netpoll for monitoring client connection events", "err", err)
		return err
	}

	// Make pool of X size, Y sized work queue and one pre-spawned
	// goroutine.
	s.clientManager = NewClientManager(s.poller, s.config, s.catchupBuffer)

	return nil
}

func (s *WSBroadcastServer) Start(ctx context.Context) error {
	// Prepare handshake header writer from http.Header mapping.
	header := ws.HandshakeHeaderHTTP(http.Header{
		HTTPHeaderFeedServerVersion: []string{strconv.Itoa(FeedServerVersion)},
		HTTPHeaderChainId:           []string{strconv.FormatUint(s.chainId, 10)},
	})

	return s.StartWithHeader(ctx, header)
}

var nonceRegex = regexp.MustCompile("^(0x)?[a-fA-F0-9]{64}$")

func (s *WSBroadcastServer) getPseudoRandomNonce() common.Hash {
	s.pseudoRandomNonceMutex.Lock()
	defer s.pseudoRandomNonceMutex.Unlock()
	// Increment lastPseudoRandomNonce
	for i := 31; i >= 0; i-- {
		if s.lastPseudoRandomNonce[i] == 255 {
			s.lastPseudoRandomNonce[i] = 0
		} else {
			s.lastPseudoRandomNonce[i]++
			break
		}
	}
	return s.lastPseudoRandomNonce
}

func (s *WSBroadcastServer) StartWithHeader(ctx context.Context, header ws.HandshakeHeader) error {
	s.startMutex.Lock()
	defer s.startMutex.Unlock()
	if s.started {
		return errors.New("broadcast server already started")
	}

	s.clientManager.Start(ctx)

	// handle incoming connection requests.
	// It upgrades TCP connection to WebSocket, registers netpoll listener on
	// it and stores it as a Client connection in ClientManager instance.
	//
	// Called below in accept() loop.
	handle := func(conn net.Conn) {
		config := s.config()
		// Set read and write deadlines for the handshake/upgrade
		err := conn.SetReadDeadline(time.Now().Add(config.HandshakeTimeout))
		if err != nil {
			log.Warn("error setting handshake read deadline", "err", err)
			_ = conn.Close()
			return
		}
		err = conn.SetWriteDeadline(time.Now().Add(config.HandshakeTimeout))
		if err != nil {
			log.Warn("error setting handshake write deadline", "err", err)
			_ = conn.Close()
			return
		}

		var compress *wsflate.Extension
		var negotiate func(httphead.Option) (httphead.Option, error)
		if config.EnableCompression {
			compress = &wsflate.Extension{
				Parameters: wsflate.DefaultParameters, // TODO
			}
			negotiate = compress.Negotiate
		}
		var feedClientVersionSeen bool
		var connectingIP net.IP
		var requestedSeqNum arbutil.MessageIndex
		var nonce common.Hash
		upgrader := ws.Upgrader{
			OnRequest: func(uri []byte) error {
				if strings.Contains(string(uri), LivenessProbeURI) {
					return ws.RejectConnectionError(
						ws.RejectionStatus(http.StatusOK),
					)
				}
				return nil
			},
			OnHeader: func(key []byte, value []byte) error {
				headerName := textproto.CanonicalMIMEHeaderKey(string(key))
				if headerName == HTTPHeaderFeedClientVersion {
					feedClientVersion, err := strconv.ParseUint(string(value), 0, 64)
					if err != nil {
						return ws.RejectConnectionError(
							ws.RejectionStatus(http.StatusBadRequest),
							ws.RejectionReason(fmt.Sprintf("Malformed HTTP header %s", HTTPHeaderFeedClientVersion)),
						)
					}
					if feedClientVersion < FeedClientVersion {
						return ws.RejectConnectionError(
							ws.RejectionStatus(http.StatusBadRequest),
							ws.RejectionReason(fmt.Sprintf("Feed Client version too old: %d, expected %d", feedClientVersion, FeedClientVersion)),
						)
					}
					feedClientVersionSeen = true
				} else if headerName == HTTPHeaderRequestedSequenceNumber {
					num, err := strconv.ParseUint(string(value), 0, 64)
					if err != nil {
						return ws.RejectConnectionError(
							ws.RejectionStatus(http.StatusBadRequest),
							ws.RejectionReason(fmt.Sprintf("Malformed HTTP header %s", HTTPHeaderRequestedSequenceNumber)),
						)
					}
					requestedSeqNum = arbutil.MessageIndex(num)
				} else if headerName == HTTPHeaderCloudflareConnectingIP {
					connectingIP = net.ParseIP(string(value))
					log.Trace("Client IP parsed from header", "ip", connectingIP, "header", headerName, "value", string(value))
				} else if headerName == HTTPHeaderNonce {
					if nonceRegex.Match(value) {
						nonce = common.HexToHash(string(value))
					} else {
						log.Trace("failed to parse nonce from header", "ip", connectingIP, "header", headerName, "value", string(value))
					}
				}

				return nil
			},
			OnBeforeUpgrade: func() (ws.HandshakeHeader, error) {
				if config.RequireVersion && !feedClientVersionSeen {
					return nil, ws.RejectConnectionError(
						ws.RejectionStatus(http.StatusBadRequest),
						ws.RejectionReason(fmt.Sprintf("Missing HTTP header %s", HTTPHeaderFeedClientVersion)),
					)
				}
				if connectingIP == nil {
					if addr, ok := conn.RemoteAddr().(*net.TCPAddr); ok {
						connectingIP = addr.IP
						log.Trace("Client IP taken from socket", "ip", connectingIP, "remoteAddr", conn.RemoteAddr())
					} else {
						log.Warn("No client IP could be determined from socket", "remoteAddr", conn.RemoteAddr())
					}
				}

				if config.ConnectionLimits.Enable && !s.clientManager.connectionLimiter.IsAllowed(connectingIP) {
					return nil, ws.RejectConnectionError(
						ws.RejectionStatus(http.StatusTooManyRequests),
						ws.RejectionReason("Too many open feed connections."),
					)
				}

				return header, nil
			},
			Negotiate: negotiate,
		}

		// Zero-copy upgrade to WebSocket connection.
		_, err = upgrader.Upgrade(conn)

		if err != nil {
			if err.Error() != "" {
				// Only log if liveness probe was not called
				log.Debug("websocket upgrade error", "connectingIP", connectingIP, "err", err)
				clientsTotalFailedUpgradeCounter.Inc(1)
			}
			_ = conn.Close()
			return
		}

		if nonce == (common.Hash{}) {
			nonce = s.getPseudoRandomNonce()
		}

		compressionAccepted := false
		if compress != nil {
			_, compressionAccepted = compress.Accepted()
		}
		if config.RequireCompression && !compressionAccepted {
			log.Warn("client did not accept required compression, disconnecting", "connectingIP", connectingIP)
			_ = conn.Close()
			return
		}
		// Unset our handshake/upgrade deadlines
		err = conn.SetReadDeadline(time.Time{})
		if err != nil {
			log.Warn("error unsetting read deadline", "connectingIP", connectingIP, "err", err)
			_ = conn.Close()
			return
		}
		err = conn.SetWriteDeadline(time.Time{})
		if err != nil {
			log.Warn("error unsetting write deadline", "connectingIP", connectingIP, "err", err)
			_ = conn.Close()
			return
		}

		// Create netpoll event descriptor to handle only read events.
		desc, err := netpoll.HandleRead(conn)
		if err != nil {
			log.Warn("error in HandleRead", "connectingIP", connectingIP, "err", err)
			_ = conn.Close()
			return
		}

		// Register incoming client in clientManager.
		safeConn := writeDeadliner{conn, config.WriteTimeout}

		client := s.clientManager.Register(safeConn, desc, requestedSeqNum, connectingIP, compressionAccepted, nonce)

		// Subscribe to events about conn.
		err = s.poller.Start(desc, func(ev netpoll.Event) {
			if ev&(netpoll.EventReadHup|netpoll.EventHup) != 0 {
				// ReadHup or Hup received, means the client has close the connection
				// remove it from the clientManager registry.
				log.Debug("Hup received", "age", client.Age(), "client", client.Name)
				s.clientManager.Remove(client)
				return
			}

			if ev > 1 {
				log.Debug("event greater than 1 received", "client", client.Name, "event", int(ev))
			}

			// receive client messages, close on error
			s.clientManager.pool.Schedule(func() {
				// Ignore any messages sent from client, close on any error
				if _, _, err := client.Receive(ctx, s.config().ReadTimeout); err != nil {
					s.clientManager.Remove(client)
					return
				}
			})
		})

		if err != nil {
			log.Warn("error starting client connection poller", "err", err)
		}
	}

	// Create tcp server for relay connections
	config := s.config()
	ln, err := net.Listen("tcp", config.Addr+":"+config.Port)
	if err != nil {
		log.Error("error calling net.Listen", "err", err)
		return err
	}

	s.listener = ln

	log.Info("arbitrum websocket broadcast server is listening", "address", ln.Addr().String())

	// Create netpoll descriptor for the listener.
	// We use OneShot here to synchronously manage the rate that new connections are accepted
	acceptDesc, err := netpoll.HandleListener(ln, netpoll.EventRead|netpoll.EventOneShot)
	if err != nil {
		log.Error("error calling HandleListener", "err", err)
		return err
	}
	s.acceptDesc = acceptDesc

	// acceptErrChan blocks until connection accepted or error occurred
	// OneShot is used, so reusing a single channel is fine
	acceptErrChan := make(chan error, 1)

	// Subscribe to events about listener.
	err = s.poller.Start(acceptDesc, func(e netpoll.Event) {
		select {
		case <-ctx.Done():
			return
		default:
		}
		// We do not want to accept incoming connection when goroutine pool is
		// busy. So if there are no free goroutines during 1ms we want to
		// cooldown the server and do not receive connection for some short
		// time.
		err := s.clientManager.pool.ScheduleTimeout(time.Millisecond, func() {
			conn, err := ln.Accept()
			if err != nil {
				acceptErrChan <- err
				return
			}

			acceptErrChan <- nil
			handle(conn)
		})
		if err == nil {
			err = <-acceptErrChan
		}
		if err != nil {
			if errors.Is(err, gopool.ErrScheduleTimeout) {
				log.Warn("broadcast poller timed out waiting for available worker", "err", err)
				clientsTotalFailedWorkerCounter.Inc(1)
			} else if errors.Is(err, netpoll.ErrNotRegistered) {
				log.Error("broadcast poller unable to register file descriptor", "err", err)
			} else {
				var netError net.Error
				isNetError := errors.As(err, &netError)
				if (!isNetError || !netError.Timeout()) && !strings.Contains(err.Error(), "timed out") {
					log.Error("broadcast poller error", "err", err)
				}
			}

			// cooldown
			delay := 5 * time.Millisecond
			log.Info("accept error", "delay", delay.String(), "err", err)
			time.Sleep(delay)
		}

		s.acceptDescMutex.Lock()
		if s.acceptDesc == nil {
			// Already shutting down
			s.acceptDescMutex.Unlock()
			return
		}
		err = s.poller.Resume(s.acceptDesc)
		s.acceptDescMutex.Unlock()
		if err != nil {
			log.Warn("error in poller.Resume", "err", err)
			s.fatalErrChan <- errors.Wrap(err, "error in poller.Resume")
			return
		}
	})
	if err != nil {
		log.Warn("error in starting broadcaster poller", "err", err)
		return err
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

	s.acceptDescMutex.Lock()
	err = s.acceptDesc.Close()
	s.acceptDesc = nil
	s.acceptDescMutex.Unlock()
	if err != nil {
		log.Warn("error in acceptDesc.Close", "err", err)
	}

	s.clientManager.StopAndWait()
	s.started = false
}

func (s *WSBroadcastServer) Started() bool {
	return s.started
}

// Broadcast sends batch item to all clients.
func (s *WSBroadcastServer) Broadcast(bm interface{}) {
	s.clientManager.Broadcast(bm)
}

func (s *WSBroadcastServer) ClientCount() int32 {
	return s.clientManager.ClientCount()
}

// writeDeadliner is a wrapper around net.Conn that sets write deadlines before
// every Write() call.
type writeDeadliner struct {
	net.Conn
	t time.Duration
}

func (d writeDeadliner) Write(p []byte) (int, error) {
	if err := d.Conn.SetWriteDeadline(time.Now().Add(d.t)); err != nil {
		return 0, err
	}
	return d.Conn.Write(p)
}
