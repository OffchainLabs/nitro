// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package wsbroadcastserver

import (
	"bytes"
	"compress/flate"
	"context"
	"encoding/json"
	"io"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws-examples/src/gopool"
	"github.com/gobwas/ws/wsflate"
	"github.com/gobwas/ws/wsutil"
	"github.com/mailru/easygo/netpoll"
	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

var (
	clientsConnectedGauge             = metrics.NewRegisteredGauge("arb/feed/clients/connected", nil)
	clientsTotalSuccessCounter        = metrics.NewRegisteredCounter("arb/feed/clients/success", nil)
	clientsTotalFailedRegisterCounter = metrics.NewRegisteredCounter("arb/feed/clients/failed/register", nil)
	clientsTotalFailedUpgradeCounter  = metrics.NewRegisteredCounter("arb/feed/clients/failed/upgrade", nil)
	clientsTotalFailedWorkerCounter   = metrics.NewRegisteredCounter("arb/feed/clients/failed/worker", nil)
	clientsDurationHistogram          = metrics.NewRegisteredHistogram("arb/feed/clients/duration", nil, metrics.NewBoundedHistogramSample())
)

// CatchupBuffer is a Protocol-specific client catch-up logic can be injected using this interface
type CatchupBuffer interface {
	OnRegisterClient(context.Context, *ClientConnection) error
	OnDoBroadcast(interface{}) error
	GetMessageCount() int
}

// ClientManager manages client connections
type ClientManager struct {
	stopwaiter.StopWaiter

	clientPtrMap  map[*ClientConnection]bool
	clientCount   int32
	pool          *gopool.Pool
	poller        netpoll.Poller
	broadcastChan chan interface{}
	clientAction  chan ClientConnectionAction
	config        BroadcasterConfigFetcher
	catchupBuffer CatchupBuffer
	flateWriter   *flate.Writer
}

type ClientConnectionAction struct {
	cc     *ClientConnection
	create bool
}

func NewClientManager(poller netpoll.Poller, configFetcher BroadcasterConfigFetcher, catchupBuffer CatchupBuffer) *ClientManager {
	config := configFetcher()
	return &ClientManager{
		poller:        poller,
		pool:          gopool.NewPool(config.Workers, config.Queue, 1),
		clientPtrMap:  make(map[*ClientConnection]bool),
		broadcastChan: make(chan interface{}, 1),
		clientAction:  make(chan ClientConnectionAction, 128),
		config:        configFetcher,
		catchupBuffer: catchupBuffer,
	}
}

func (cm *ClientManager) registerClient(ctx context.Context, clientConnection *ClientConnection) error {
	defer func() {
		if r := recover(); r != nil {
			log.Error("Recovered in registerClient", "recover", r)
		}
	}()

	clientsConnectedGauge.Inc(1)
	atomic.AddInt32(&cm.clientCount, 1)
	if err := cm.catchupBuffer.OnRegisterClient(ctx, clientConnection); err != nil {
		clientsTotalFailedRegisterCounter.Inc(1)
		return err
	}

	clientConnection.Start(ctx)
	cm.clientPtrMap[clientConnection] = true
	clientsTotalSuccessCounter.Inc(1)

	return nil
}

// Register registers new connection as a Client.
func (cm *ClientManager) Register(
	conn net.Conn,
	desc *netpoll.Desc,
	requestedSeqNum arbutil.MessageIndex,
	connectingIP string,
	compression bool,
) *ClientConnection {
	createClient := ClientConnectionAction{
		NewClientConnection(conn, desc, cm, requestedSeqNum, connectingIP, compression),
		true,
	}
	cm.clientAction <- createClient

	return createClient.cc
}

// removeAll removes all clients after main ClientManager thread exits
func (cm *ClientManager) removeAll() {
	// Only called after main ClientManager thread exits, so remove client directly
	for client := range cm.clientPtrMap {
		cm.removeClientImpl(client)
	}
}

func (cm *ClientManager) removeClientImpl(clientConnection *ClientConnection) {
	clientConnection.StopOnly()

	err := cm.poller.Stop(clientConnection.desc)
	if err != nil {
		log.Warn("Failed to stop poller", "err", err)
	}

	err = clientConnection.conn.Close()
	if err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
		log.Warn("Failed to close client connection", "err", err)
	}

	log.Info("client removed", "client", clientConnection.Name, "age", clientConnection.Age())
	clientsDurationHistogram.Update(clientConnection.Age().Nanoseconds())
	clientsConnectedGauge.Dec(1)
	atomic.AddInt32(&cm.clientCount, -1)
}

func (cm *ClientManager) removeClient(clientConnection *ClientConnection) {
	if !cm.clientPtrMap[clientConnection] {
		return
	}

	cm.removeClientImpl(clientConnection)

	delete(cm.clientPtrMap, clientConnection)
}

func (cm *ClientManager) Remove(clientConnection *ClientConnection) {
	cm.clientAction <- ClientConnectionAction{
		clientConnection,
		false,
	}
}

func (cm *ClientManager) ClientCount() int32 {
	return atomic.LoadInt32(&cm.clientCount)
}

// Broadcast sends batch item to all clients.
func (cm *ClientManager) Broadcast(bm interface{}) {
	cm.broadcastChan <- bm
}

func (cm *ClientManager) doBroadcast(bm interface{}) ([]*ClientConnection, error) {
	if err := cm.catchupBuffer.OnDoBroadcast(bm); err != nil {
		return nil, err
	}
	// TODO make enableCompression a config
	enableCompression := true

	//                                        /-> wsutil.Writer -> notCompressed buffer
	// bm -> json.Encoder -> io.MultiWriter -|
	// 										  \-> cm.flateWriter -> wsutil.Writer -> compressed buffer
	var notCompressed bytes.Buffer
	notCompressedWriter := wsutil.NewWriter(&notCompressed, ws.StateServerSide, ws.OpText)
	writers := []io.Writer{notCompressedWriter}

	var compressed bytes.Buffer
	var compressedWriter *wsutil.Writer
	if enableCompression {
		if cm.flateWriter == nil {
			var err error
			cm.flateWriter, err = flate.NewWriterDict(nil, GetCompressionLevel(), GetStaticCompressorDictionary())
			if err != nil {
				return nil, errors.Wrap(err, "unable to create flate writer")
			}
		}
		compressedWriter = wsutil.NewWriter(&compressed, ws.StateServerSide|ws.StateExtended, ws.OpText)
		var msg wsflate.MessageState
		msg.SetCompressed(true)
		compressedWriter.SetExtensions(&msg)
		cm.flateWriter.Reset(compressedWriter)
		writers = append(writers, cm.flateWriter)
	}

	multiWriter := io.MultiWriter(writers...)
	encoder := json.NewEncoder(multiWriter)
	if err := encoder.Encode(bm); err != nil {
		return nil, errors.Wrap(err, "unable to encode message")
	}
	if err := notCompressedWriter.Flush(); err != nil {
		return nil, errors.Wrap(err, "unable to flush message")
	}
	log.Info("not compressed", "data", notCompressed.String())
	if enableCompression && cm.flateWriter != nil {
		if err := cm.flateWriter.Flush(); err != nil {
			return nil, errors.Wrap(err, "unable to flush message")
		}
	}
	if err := compressedWriter.Flush(); err != nil {
		return nil, errors.Wrap(err, "unable to flush message")
	}
	log.Info("compressed", "data", compressed.String())

	clientDeleteList := make([]*ClientConnection, 0, len(cm.clientPtrMap))
	for client := range cm.clientPtrMap {
		var data []byte
		if client.Compression() {
			if enableCompression {
				data = compressed.Bytes()
			} else {
				// TODO revise
				log.Error("disconnecting because client has enabled compression, but compression support is disabled", "client", client.Name, "size")
				clientDeleteList = append(clientDeleteList, client)
				continue
			}
		} else {
			data = notCompressed.Bytes()
		}
		select {
		case client.out <- data:
		default:
			// Queue for client too backed up, disconnect instead of blocking on channel send
			log.Info("disconnecting because send queue too large", "client", client.Name, "size", len(client.out))
			clientDeleteList = append(clientDeleteList, client)
		}
	}

	return clientDeleteList, nil
}

// verifyClients should be called every cm.config.ClientPingInterval
func (cm *ClientManager) verifyClients() []*ClientConnection {
	clientConnectionCount := len(cm.clientPtrMap)

	// Create list of clients to clients to remove
	clientDeleteList := make([]*ClientConnection, 0, clientConnectionCount)

	// Send ping to all connected clients
	log.Debug("pinging clients", "count", len(cm.clientPtrMap))
	for client := range cm.clientPtrMap {
		diff := time.Since(client.GetLastHeard())
		if diff > cm.config().ClientTimeout {
			log.Info("disconnecting because connection timed out", "client", client.Name)
			clientDeleteList = append(clientDeleteList, client)
		} else {
			err := client.Ping()
			if err != nil {
				log.Warn("disconnecting because error pinging client", "client", client.Name)
				clientDeleteList = append(clientDeleteList, client)
			}
		}
	}

	return clientDeleteList
}

func (cm *ClientManager) Start(parentCtx context.Context) {
	cm.StopWaiter.Start(parentCtx, cm)

	cm.LaunchThread(func(ctx context.Context) {
		defer cm.removeAll()

		// Ping needs to occur regularly regardless of other traffic
		pingTimer := time.NewTimer(cm.config().Ping)
		var clientDeleteList []*ClientConnection
		defer pingTimer.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case clientAction := <-cm.clientAction:
				if clientAction.create {
					err := cm.registerClient(ctx, clientAction.cc)
					if err != nil {
						// Log message already output in registerClient
						cm.removeClientImpl(clientAction.cc)
					}
				} else {
					cm.removeClient(clientAction.cc)
				}
			case bm := <-cm.broadcastChan:
				var err error
				clientDeleteList, err = cm.doBroadcast(bm)
				logError(err, "failed to do broadcast")
			case <-pingTimer.C:
				clientDeleteList = cm.verifyClients()
				pingTimer.Reset(cm.config().Ping)
			}

			if len(clientDeleteList) > 0 {
				for _, client := range clientDeleteList {
					cm.removeClient(client)
				}
				clientDeleteList = nil
			}
		}
	})
}
