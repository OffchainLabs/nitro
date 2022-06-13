// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package wsbroadcastserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/pkg/errors"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws-examples/src/gopool"
	"github.com/gobwas/ws/wsutil"
	"github.com/mailru/easygo/netpoll"
)

/* Protocol-specific client catch-up logic can be injected using this interface. */
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
	settings      BroadcasterConfig
	catchupBuffer CatchupBuffer
}

type ClientConnectionAction struct {
	cc     *ClientConnection
	create bool
}

func NewClientManager(poller netpoll.Poller, settings BroadcasterConfig, catchupBuffer CatchupBuffer) *ClientManager {
	return &ClientManager{
		poller:        poller,
		pool:          gopool.NewPool(settings.Workers, settings.Queue, 1),
		clientPtrMap:  make(map[*ClientConnection]bool),
		broadcastChan: make(chan interface{}, 1),
		clientAction:  make(chan ClientConnectionAction, 128),
		settings:      settings,
		catchupBuffer: catchupBuffer,
	}
}

func (cm *ClientManager) registerClient(ctx context.Context, clientConnection *ClientConnection) error {
	if err := cm.catchupBuffer.OnRegisterClient(ctx, clientConnection); err != nil {
		return err
	}

	clientConnection.Start(ctx)
	cm.clientPtrMap[clientConnection] = true
	atomic.AddInt32(&cm.clientCount, 1)

	return nil
}

// Register registers new connection as a Client.
func (cm *ClientManager) Register(conn net.Conn, desc *netpoll.Desc) *ClientConnection {
	createClient := ClientConnectionAction{
		NewClientConnection(conn, desc, cm),
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
	clientConnection.StopAndWait()

	err := cm.poller.Stop(clientConnection.desc)
	if err != nil {
		log.Warn("Failed to stop poller", "err", err)
	}

	err = clientConnection.conn.Close()
	if err != nil {
		log.Warn("Failed to close client connection", "err", err)
	}

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

	var buf bytes.Buffer
	writer := wsutil.NewWriter(&buf, ws.StateServerSide, ws.OpText)
	encoder := json.NewEncoder(writer)
	if err := encoder.Encode(bm); err != nil {
		return nil, errors.Wrap(err, "unable to encode message")
	}
	if err := writer.Flush(); err != nil {
		return nil, errors.Wrap(err, "unable to flush message")
	}

	clientDeleteList := make([]*ClientConnection, 0, len(cm.clientPtrMap))
	for client := range cm.clientPtrMap {
		if len(client.out) == cm.settings.MaxSendQueue {
			// Queue for client too backed up, disconnect instead of blocking on channel send
			log.Info("disconnecting because send queue too large", "client", client.Name, "size", len(client.out))
			clientDeleteList = append(clientDeleteList, client)
		} else {
			client.out <- buf.Bytes()
		}
	}

	return clientDeleteList, nil
}

// verifyClients should be called every cm.settings.ClientPingInterval
func (cm *ClientManager) verifyClients() []*ClientConnection {
	clientConnectionCount := len(cm.clientPtrMap)

	// Create list of clients to clients to remove
	clientDeleteList := make([]*ClientConnection, 0, clientConnectionCount)

	// Send ping to all connected clients
	log.Debug("pinging clients", "count", len(cm.clientPtrMap))
	for client := range cm.clientPtrMap {
		diff := time.Since(client.GetLastHeard())
		if diff > cm.settings.ClientTimeout {
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
	cm.StopWaiter.Start(parentCtx)

	cm.LaunchThread(func(ctx context.Context) {
		defer cm.removeAll()

		pingInterval := time.NewTicker(cm.settings.Ping)
		defer pingInterval.Stop()
		var clientDeleteList []*ClientConnection
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
			case <-pingInterval.C:
				clientDeleteList = cm.verifyClients()
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
