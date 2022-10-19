// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package wsbroadcastserver

import (
	"context"
	"encoding/json"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/offchainlabs/nitro/arbutil"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/mailru/easygo/netpoll"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

// ClientConnection represents client connection.
type ClientConnection struct {
	stopwaiter.StopWaiter

	ioMutex sync.Mutex
	conn    net.Conn

	desc            *netpoll.Desc
	Name            string
	clientManager   *ClientManager
	requestedSeqNum arbutil.MessageIndex

	lastHeardUnix int64
	out           chan []byte
}

func NewClientConnection(conn net.Conn, desc *netpoll.Desc, clientManager *ClientManager, requestedSeqNum arbutil.MessageIndex) *ClientConnection {
	return &ClientConnection{
		conn:            conn,
		desc:            desc,
		Name:            conn.RemoteAddr().String() + strconv.Itoa(rand.Intn(10)),
		clientManager:   clientManager,
		requestedSeqNum: requestedSeqNum,
		lastHeardUnix:   time.Now().Unix(),
		out:             make(chan []byte, clientManager.config().MaxSendQueue),
	}
}

func (cc *ClientConnection) Start(parentCtx context.Context) {
	cc.StopWaiter.Start(parentCtx, cc)
	cc.LaunchThread(func(ctx context.Context) {
		defer close(cc.out)
		for {
			select {
			case <-ctx.Done():
				return
			case data := <-cc.out:
				err := cc.writeRaw(data)
				if err != nil {
					logWarn(err, "error writing data to client")
					cc.clientManager.Remove(cc)
					return
				}
			}
		}
	})
}

func (cc *ClientConnection) StopAndWait() {
	// Ignore errors from conn.Close since we are just shutting down
	_ = cc.conn.Close()
	if !cc.Started() {
		// If client connection never started, need to close channel
		close(cc.out)
	} else {
		cc.StopWaiter.StopAndWait()
	}
}

func (cc *ClientConnection) RequestedSeqNum() arbutil.MessageIndex {
	return cc.requestedSeqNum
}

func (cc *ClientConnection) GetLastHeard() time.Time {
	return time.Unix(atomic.LoadInt64(&cc.lastHeardUnix), 0)
}

// Receive reads next message from client's underlying connection.
// It blocks until full message received.
func (cc *ClientConnection) Receive(ctx context.Context, timeout time.Duration) ([]byte, ws.OpCode, error) {
	msg, op, err := cc.readRequest(ctx, timeout)
	if err != nil {
		_ = cc.conn.Close()
		return nil, op, err
	}

	return msg, op, err
}

// readRequests reads json-rpc request from connection.
func (cc *ClientConnection) readRequest(ctx context.Context, timeout time.Duration) ([]byte, ws.OpCode, error) {
	cc.ioMutex.Lock()
	defer cc.ioMutex.Unlock()

	atomic.StoreInt64(&cc.lastHeardUnix, time.Now().Unix())

	return ReadData(ctx, cc.conn, nil, timeout, ws.StateServerSide)
}

func (cc *ClientConnection) Write(x interface{}) error {
	writer := wsutil.NewWriter(cc.conn, ws.StateServerSide, ws.OpText)
	encoder := json.NewEncoder(writer)

	cc.ioMutex.Lock()
	defer cc.ioMutex.Unlock()

	if err := encoder.Encode(x); err != nil {
		return err
	}

	return writer.Flush()
}

func (cc *ClientConnection) writeRaw(p []byte) error {
	cc.ioMutex.Lock()
	defer cc.ioMutex.Unlock()

	_, err := cc.conn.Write(p)

	return err
}

func (cc *ClientConnection) Ping() error {
	cc.ioMutex.Lock()
	defer cc.ioMutex.Unlock()
	_, err := cc.conn.Write(ws.CompiledPing)
	if err != nil {
		return err
	}

	return nil
}
