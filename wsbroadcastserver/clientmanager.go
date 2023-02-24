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
	"github.com/zyedidia/generic/btree"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

var (
	clientsCurrentGauge               = metrics.NewRegisteredGauge("arb/feed/clients/current", nil)
	clientsConnectCount               = metrics.NewRegisteredCounter("arb/feed/clients/connect", nil)
	clientsDisconnectCount            = metrics.NewRegisteredCounter("arb/feed/clients/disconnect", nil)
	clientsTotalSuccessCounter        = metrics.NewRegisteredCounter("arb/feed/clients/success", nil)
	clientsTotalFailedRegisterCounter = metrics.NewRegisteredCounter("arb/feed/clients/failed/register", nil)
	clientsTotalFailedUpgradeCounter  = metrics.NewRegisteredCounter("arb/feed/clients/failed/upgrade", nil)
	clientsTotalFailedWorkerCounter   = metrics.NewRegisteredCounter("arb/feed/clients/failed/worker", nil)
	clientsDurationHistogram          = metrics.NewRegisteredHistogram("arb/feed/clients/duration", nil, metrics.NewBoundedHistogramSample())
)

// CatchupBuffer is a Protocol-specific client catch-up logic can be injected using this interface
type CatchupBuffer interface {
	OnRegisterClient(*ClientConnection) (error, int, time.Duration)
	OnDoBroadcast(interface{}) error
	GetMessageCount() int
}

// ClientManager manages client connections
type ClientManager struct {
	stopwaiter.StopWaiter

	clientPtrMap  *btree.Tree[common.Hash, *ClientConnection]
	clientCount   int32
	pool          *gopool.Pool
	poller        netpoll.Poller
	broadcastChan chan interface{}
	clientAction  chan ClientConnectionAction
	config        BroadcasterConfigFetcher
	catchupBuffer CatchupBuffer
	flateWriter   *flate.Writer

	connectionLimiter *ConnectionLimiter
}

type ClientConnectionAction struct {
	cc     *ClientConnection
	create bool
}

func lesserHash(aHash, bHash common.Hash) bool {
	for i, a := range aHash {
		b := bHash[i]
		if a < b {
			return true
		} else if a > b {
			return false
		}
	}
	return false
}

func NewClientManager(poller netpoll.Poller, configFetcher BroadcasterConfigFetcher, catchupBuffer CatchupBuffer) *ClientManager {
	config := configFetcher()
	return &ClientManager{
		poller:            poller,
		pool:              gopool.NewPool(config.Workers, config.Queue, 1),
		clientPtrMap:      btree.New[common.Hash, *ClientConnection](lesserHash),
		broadcastChan:     make(chan interface{}, 1),
		clientAction:      make(chan ClientConnectionAction, 128),
		config:            configFetcher,
		catchupBuffer:     catchupBuffer,
		connectionLimiter: NewConnectionLimiter(func() *ConnectionLimiterConfig { return &configFetcher().ConnectionLimits }),
	}
}

func (cm *ClientManager) registerClient(ctx context.Context, clientConnection *ClientConnection) error {
	defer func() {
		if r := recover(); r != nil {
			log.Error("Recovered in registerClient", "recover", r)
		}
	}()

	_, exists := cm.clientPtrMap.Get(clientConnection.nonceHash)
	if exists {
		return fmt.Errorf("duplicate nonce hash %v", clientConnection.nonceHash)
	}
	if cm.config().ConnectionLimits.Enable && !cm.connectionLimiter.Register(clientConnection.clientIp) {
		return fmt.Errorf("connection limited %s", clientConnection.clientIp)
	}

	clientsCurrentGauge.Inc(1)
	clientsConnectCount.Inc(1)

	atomic.AddInt32(&cm.clientCount, 1)
	err, sent, elapsed := cm.catchupBuffer.OnRegisterClient(clientConnection)
	if err != nil {
		clientsTotalFailedRegisterCounter.Inc(1)
		if cm.config().ConnectionLimits.Enable {
			cm.connectionLimiter.Release(clientConnection.clientIp)
		}
		return err
	}
	if cm.config().LogConnect {
		log.Info("client registered", "client", clientConnection.Name, "requestedSeqNum", clientConnection.RequestedSeqNum(), "sentCount", sent, "elapsed", elapsed)
	}

	clientConnection.Start(ctx)
	cm.clientPtrMap.Put(clientConnection.nonceHash, clientConnection)
	clientsTotalSuccessCounter.Inc(1)

	return nil
}

// Register registers new connection as a Client.
func (cm *ClientManager) Register(
	conn net.Conn,
	desc *netpoll.Desc,
	requestedSeqNum arbutil.MessageIndex,
	connectingIP net.IP,
	compression bool,
	nonce common.Hash,
) *ClientConnection {
	createClient := ClientConnectionAction{
		NewClientConnection(conn, desc, cm, requestedSeqNum, connectingIP, compression, nonce),
		true,
	}
	cm.clientAction <- createClient

	return createClient.cc
}

// removeAll removes all clients after main ClientManager thread exits
func (cm *ClientManager) removeAll() {
	// Only called after main ClientManager thread exits, so remove client directly
	cm.clientPtrMap.Each(func(_ common.Hash, client *ClientConnection) {
		cm.removeClientImpl(client)
	})
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
	atomic.AddInt32(&cm.clientCount, -1)
}

func (cm *ClientManager) removeClient(clientConnection *ClientConnection) {
	_, exists := cm.clientPtrMap.Get(clientConnection.nonceHash)
	if !exists {
		return
	}

	cm.removeClientImpl(clientConnection)
	if cm.config().ConnectionLimits.Enable {
		cm.connectionLimiter.Release(clientConnection.clientIp)
	}

	cm.clientPtrMap.Remove(clientConnection.nonceHash)
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
	if cm.Stopped() {
		// This should only occur if a reorg occurs after the broadcast server is stopped,
		// with the sequencer enabled but not the sequencer coordinator.
		// In this case we should proceed without broadcasting the message.
		return
	}
	cm.broadcastChan <- bm
}

const maxNonceHashAge = time.Hour * 25

func (cm *ClientManager) doBroadcast(bm interface{}) ([]*ClientConnection, error) {
	if err := cm.catchupBuffer.OnDoBroadcast(bm); err != nil {
		return nil, err
	}
	config := cm.config()
	//                                        /-> wsutil.Writer -> not compressed msg buffer
	// bm -> json.Encoder -> io.MultiWriter -|
	//                                        \-> cm.flateWriter -> wsutil.Writer -> compressed msg buffer
	writers := []io.Writer{}
	var notCompressed bytes.Buffer
	var notCompressedWriter *wsutil.Writer
	var compressed bytes.Buffer
	var compressedWriter *wsutil.Writer
	if !config.RequireCompression {
		notCompressedWriter = wsutil.NewWriter(&notCompressed, ws.StateServerSide, ws.OpText)
		writers = append(writers, notCompressedWriter)
	}
	if config.EnableCompression {
		if cm.flateWriter == nil {
			var err error
			cm.flateWriter, err = flate.NewWriterDict(nil, DeflateCompressionLevel, GetStaticCompressorDictionary())
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
	if notCompressedWriter != nil {
		if err := notCompressedWriter.Flush(); err != nil {
			return nil, errors.Wrap(err, "unable to flush message")
		}
	}
	if compressedWriter != nil {
		if err := cm.flateWriter.Close(); err != nil {
			return nil, errors.Wrap(err, "unable to close flate writer")
		}
		if err := compressedWriter.Flush(); err != nil {
			return nil, errors.Wrap(err, "unable to flush message")
		}
	}

	sendQueueTooLargeCount := 0
	clientDeleteList := make([]*ClientConnection, 0, cm.clientPtrMap.Size())
	recomputeNonceList := make([]*ClientConnection, 0, cm.clientPtrMap.Size())
	i := 0
	cm.clientPtrMap.Each(func(_ common.Hash, client *ClientConnection) {
		var data []byte
		if client.Compression() {
			if config.EnableCompression {
				data = compressed.Bytes()
			} else {
				log.Warn("disconnecting because client has enabled compression, but compression support is disabled", "client", client.Name)
				clientDeleteList = append(clientDeleteList, client)
				return
			}
		} else {
			if !config.RequireCompression {
				data = notCompressed.Bytes()
			} else {
				log.Warn("disconnecting because client has disabled compression, but compression support is required", "client", client.Name)
				clientDeleteList = append(clientDeleteList, client)
				return
			}
		}
		select {
		case client.out <- data:
		default:
			// Queue for client too backed up, disconnect instead of blocking on channel send
			sendQueueTooLargeCount++
			clientDeleteList = append(clientDeleteList, client)
			return
		}
		if i != 0 && i <= config.BroadcastDelay.BatchSize*config.BroadcastDelay.BatchCount && i%config.BroadcastDelay.BatchSize == 0 {
			time.Sleep(config.BroadcastDelay.Delay)
		}
		i++
		if time.Since(client.nonceHashLastComputed) >= maxNonceHashAge {
			recomputeNonceList = append(recomputeNonceList, client)
		}
	})

	for _, client := range recomputeNonceList {
		now := time.Now()
		newNonceHash := computeNonceHash(client.nonce, now)
		if client.nonceHash == newNonceHash {
			client.nonceHashLastComputed = now
			continue
		}
		_, exists := cm.clientPtrMap.Get(newNonceHash)
		if exists {
			// This nonce was already used on the next day
			clientDeleteList = append(clientDeleteList, client)
			continue
		}
		cm.clientPtrMap.Remove(client.nonceHash)
		client.nonceHash = newNonceHash
		client.nonceHashLastComputed = now
		cm.clientPtrMap.Put(newNonceHash, client)
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

// verifyClients should be called every cm.config.ClientPingInterval
func (cm *ClientManager) verifyClients() []*ClientConnection {
	clientConnectionCount := cm.clientPtrMap.Size()

	// Create list of clients to remove
	clientDeleteList := make([]*ClientConnection, 0, clientConnectionCount)

	// Send ping to all connected clients
	log.Debug("pinging clients", "count", cm.clientPtrMap.Size())
	cm.clientPtrMap.Each(func(_ common.Hash, client *ClientConnection) {
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
	})

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
