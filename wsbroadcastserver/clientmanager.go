// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package wsbroadcastserver

import (
	"bytes"
	"compress/flate"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws-examples/src/gopool"
	"github.com/gobwas/ws/wsflate"
	"github.com/gobwas/ws/wsutil"
	"github.com/mailru/easygo/netpoll"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/broadcaster/backlog"
	m "github.com/offchainlabs/nitro/broadcaster/message"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

var (
	clientsCurrentGauge              = metrics.NewRegisteredGauge("arb/feed/clients/current", nil)
	clientsConnectCount              = metrics.NewRegisteredCounter("arb/feed/clients/connect", nil)
	clientsDisconnectCount           = metrics.NewRegisteredCounter("arb/feed/clients/disconnect", nil)
	clientsTotalSuccessCounter       = metrics.NewRegisteredCounter("arb/feed/clients/success", nil)
	clientsTotalFailedUpgradeCounter = metrics.NewRegisteredCounter("arb/feed/clients/failed/upgrade", nil)
	clientsTotalFailedWorkerCounter  = metrics.NewRegisteredCounter("arb/feed/clients/failed/worker", nil)
	clientsDurationHistogram         = metrics.NewRegisteredHistogram("arb/feed/clients/duration", nil, metrics.NewBoundedHistogramSample())
)

// ClientManager manages client connections
type ClientManager struct {
	stopwaiter.StopWaiter

	clientPtrMap  map[*ClientConnection]bool
	clientCount   atomic.Int32
	pool          *gopool.Pool
	poller        netpoll.Poller
	broadcastChan chan *m.BroadcastMessage
	clientAction  chan ClientConnectionAction
	config        BroadcasterConfigFetcher
	backlog       backlog.Backlog

	connectionLimiter *ConnectionLimiter
}

func NewClientManager(poller netpoll.Poller, configFetcher BroadcasterConfigFetcher, bklg backlog.Backlog) *ClientManager {
	config := configFetcher()
	return &ClientManager{
		poller:            poller,
		pool:              gopool.NewPool(config.Workers, config.Queue, 1),
		clientPtrMap:      make(map[*ClientConnection]bool),
		broadcastChan:     make(chan *m.BroadcastMessage, 1),
		clientAction:      make(chan ClientConnectionAction, 128),
		config:            configFetcher,
		backlog:           bklg,
		connectionLimiter: NewConnectionLimiter(func() *ConnectionLimiterConfig { return &configFetcher().ConnectionLimits }),
	}
}

func (cm *ClientManager) registerClient(ctx context.Context, clientConnection *ClientConnection) error {
	defer func() {
		if r := recover(); r != nil {
			log.Error("Recovered in registerClient", "recover", r)
		}
	}()

	// TODO:(clamb) the clientsTotalFailedRegisterCounter was deleted after backlog logic moved to ClientConnection. Should this metric be reintroduced or will it be ok to just delete completely given the behaviour has changed, ask Lee

	if cm.config().ConnectionLimits.Enable && !cm.connectionLimiter.Register(clientConnection.clientIp) {
		return fmt.Errorf("Connection limited %s", clientConnection.clientIp)
	}

	clientsCurrentGauge.Inc(1)
	clientsConnectCount.Inc(1)

	cm.clientCount.Add(1)
	cm.clientPtrMap[clientConnection] = true
	clientsTotalSuccessCounter.Inc(1)

	return nil
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

	if cm.config().LogDisconnect {
		log.Info("client removed", "client", clientConnection.Name, "age", clientConnection.Age())
	}

	clientsDurationHistogram.Update(clientConnection.Age().Microseconds())
	clientsCurrentGauge.Dec(1)
	clientsDisconnectCount.Inc(1)
	cm.clientCount.Add(-1)
}

func (cm *ClientManager) removeClient(clientConnection *ClientConnection) {
	if !cm.clientPtrMap[clientConnection] {
		return
	}

	cm.removeClientImpl(clientConnection)
	if cm.config().ConnectionLimits.Enable {
		cm.connectionLimiter.Release(clientConnection.clientIp)
	}

	delete(cm.clientPtrMap, clientConnection)
}

func (cm *ClientManager) ClientCount() int32 {
	return cm.clientCount.Load()
}

// Broadcast sends batch item to all clients.
func (cm *ClientManager) Broadcast(bm *m.BroadcastMessage) {
	if cm.Stopped() {
		// This should only occur if a reorg occurs after the broadcast server is stopped,
		// with the sequencer enabled but not the sequencer coordinator.
		// In this case we should proceed without broadcasting the message.
		return
	}
	cm.broadcastChan <- bm
}

func (cm *ClientManager) doBroadcast(bm *m.BroadcastMessage) ([]*ClientConnection, error) {
	if err := cm.backlog.Append(bm); err != nil {
		return nil, err
	}
	config := cm.config()
	//                                        /-> wsutil.Writer -> not compressed msg buffer
	// bm -> json.Encoder -> io.MultiWriter -|
	//                                        \-> flateWriter -> wsutil.Writer -> compressed msg buffer

	notCompressed, compressed, err := serializeMessage(bm, !config.RequireCompression, config.EnableCompression)
	if err != nil {
		return nil, err
	}

	sendQueueTooLargeCount := 0
	clientDeleteList := make([]*ClientConnection, 0, len(cm.clientPtrMap))
	for client := range cm.clientPtrMap {
		var data []byte
		if client.Compression() {
			if config.EnableCompression {
				data = compressed.Bytes()
			} else {
				log.Warn("disconnecting because client has enabled compression, but compression support is disabled", "client", client.Name)
				clientDeleteList = append(clientDeleteList, client)
				continue
			}
		} else {
			if !config.RequireCompression {
				data = notCompressed.Bytes()
			} else {
				log.Warn("disconnecting because client has disabled compression, but compression support is required", "client", client.Name)
				clientDeleteList = append(clientDeleteList, client)
				continue
			}
		}

		var seqNum *arbutil.MessageIndex
		n := len(bm.Messages)
		if n == 0 {
			seqNum = nil
		} else if n == 1 {
			seqNum = &bm.Messages[0].SequenceNumber
		} else {
			return nil, fmt.Errorf("doBroadcast was sent %d BroadcastFeedMessages, it can only parse 1 BroadcastFeedMessage at a time", n)
		}

		m := message{
			sequenceNumber: seqNum,
			data:           data,
		}
		select {
		case client.out <- m:
		default:
			// Queue for client too backed up, disconnect instead of blocking on channel send
			sendQueueTooLargeCount++
			clientDeleteList = append(clientDeleteList, client)
		}
	}

	if sendQueueTooLargeCount > 0 {
		if sendQueueTooLargeCount < 10 {
			log.Warn("disconnecting clients because send queue too large", "count", sendQueueTooLargeCount)
		} else {
			log.Error("disconnecting clients because send queue too large", "count", sendQueueTooLargeCount)
		}
	}

	return clientDeleteList, nil
}

func serializeMessage(bm *m.BroadcastMessage, enableNonCompressedOutput, enableCompressedOutput bool) (bytes.Buffer, bytes.Buffer, error) {
	flateWriter, err := flate.NewWriterDict(nil, DeflateCompressionLevel, GetStaticCompressorDictionary())
	if err != nil {
		return bytes.Buffer{}, bytes.Buffer{}, fmt.Errorf("unable to create flate writer: %w", err)
	}

	var notCompressed bytes.Buffer
	var compressed bytes.Buffer
	writers := []io.Writer{}
	var notCompressedWriter *wsutil.Writer
	var compressedWriter *wsutil.Writer
	if enableNonCompressedOutput {
		notCompressedWriter = wsutil.NewWriter(&notCompressed, ws.StateServerSide, ws.OpText)
		writers = append(writers, notCompressedWriter)
	}
	if enableCompressedOutput {
		compressedWriter = wsutil.NewWriter(&compressed, ws.StateServerSide|ws.StateExtended, ws.OpText)
		var msg wsflate.MessageState
		msg.SetCompressed(true)
		compressedWriter.SetExtensions(&msg)
		flateWriter.Reset(compressedWriter)
		writers = append(writers, flateWriter)
	}

	multiWriter := io.MultiWriter(writers...)
	encoder := json.NewEncoder(multiWriter)
	if err := encoder.Encode(bm); err != nil {
		return bytes.Buffer{}, bytes.Buffer{}, fmt.Errorf("unable to encode message: %w", err)
	}
	if notCompressedWriter != nil {
		if err := notCompressedWriter.Flush(); err != nil {
			return bytes.Buffer{}, bytes.Buffer{}, fmt.Errorf("unable to flush message: %w", err)
		}
	}
	if compressedWriter != nil {
		if err := flateWriter.Close(); err != nil {
			return bytes.Buffer{}, bytes.Buffer{}, fmt.Errorf("unable to close flate writer: %w", err)
		}
		if err := compressedWriter.Flush(); err != nil {
			return bytes.Buffer{}, bytes.Buffer{}, fmt.Errorf("unable to flush message: %w", err)
		}
	}
	return notCompressed, compressed, nil
}

// verifyClients should be called every cm.config.ClientPingInterval
func (cm *ClientManager) verifyClients() []*ClientConnection {
	clientConnectionCount := len(cm.clientPtrMap)

	// Create list of clients to remove
	clientDeleteList := make([]*ClientConnection, 0, clientConnectionCount)

	// Send ping to all connected clients
	log.Debug("pinging clients", "count", len(cm.clientPtrMap))
	for client := range cm.clientPtrMap {
		diff := time.Since(client.GetLastHeard())
		if diff > cm.config().ClientTimeout {
			log.Debug("disconnecting because connection timed out", "client", client.Name)
			clientDeleteList = append(clientDeleteList, client)
		} else {
			err := client.Ping()
			if err != nil {
				log.Debug("disconnecting because error pinging client", "client", client.Name)
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
					clientAction.cc.Registered()
				} else {
					cm.removeClient(clientAction.cc)
				}
			case bm := <-cm.broadcastChan:
				var err error
				for i, msg := range bm.Messages {
					m := &m.BroadcastMessage{
						Version:  bm.Version,
						Messages: []*m.BroadcastFeedMessage{msg},
					}
					// This ensures that only one message is sent with the confirmed sequence number
					if i == 0 {
						m.ConfirmedSequenceNumberMessage = bm.ConfirmedSequenceNumberMessage
					}
					clientDeleteList, err = cm.doBroadcast(m)
					logError(err, "failed to do broadcast")
				}

				// A message with ConfirmedSequenceNumberMessage could be sent without any messages
				// this section ensures that message is still sent.
				if len(bm.Messages) == 0 {
					clientDeleteList, err = cm.doBroadcast(bm)
					logError(err, "failed to do broadcast")
				}
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
