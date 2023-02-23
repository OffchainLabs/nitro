// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package wsbroadcastserver

import (
	"compress/flate"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/pkg/errors"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsflate"
	"github.com/gobwas/ws/wsutil"
	"github.com/mailru/easygo/netpoll"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

// ClientConnection represents client connection.
type ClientConnection struct {
	stopwaiter.StopWaiter

	ioMutex  sync.Mutex
	conn     net.Conn
	creation time.Time
	clientIp net.IP

	desc            *netpoll.Desc
	Name            string
	clientManager   *ClientManager
	requestedSeqNum arbutil.MessageIndex
	nonceHash       common.Hash

	lastHeardUnix int64
	out           chan []byte

	compression bool
	flateReader *wsflate.Reader
}

var nonceHashPrefix = []byte("Arbitrum relay client connection nonce")

func computeNonceHash(nonce common.Hash) common.Hash {
	return crypto.Keccak256Hash(nonceHashPrefix, nonce.Bytes())
}

func NewClientConnection(
	conn net.Conn,
	desc *netpoll.Desc,
	clientManager *ClientManager,
	requestedSeqNum arbutil.MessageIndex,
	connectingIP net.IP,
	compression bool,
	nonce common.Hash,
) *ClientConnection {
	return &ClientConnection{
		conn:            conn,
		clientIp:        connectingIP,
		desc:            desc,
		creation:        time.Now(),
		Name:            fmt.Sprintf("%s@%s-%d", connectingIP, conn.RemoteAddr(), rand.Intn(10)),
		clientManager:   clientManager,
		requestedSeqNum: requestedSeqNum,
		lastHeardUnix:   time.Now().Unix(),
		out:             make(chan []byte, clientManager.config().MaxSendQueue),
		compression:     compression,
		flateReader:     NewFlateReader(),
		nonceHash:       computeNonceHash(nonce),
	}
}

func (cc *ClientConnection) Age() time.Duration {
	return time.Since(cc.creation)
}

func (cc *ClientConnection) Compression() bool {
	return cc.compression
}

func (cc *ClientConnection) Start(parentCtx context.Context) {
	cc.StopWaiter.Start(parentCtx, cc)
	cc.LaunchThread(func(ctx context.Context) {
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

func (cc *ClientConnection) StopOnly() {
	// Ignore errors from conn.Close since we are just shutting down
	_ = cc.conn.Close()
	if cc.Started() {
		cc.StopWaiter.StopOnly()
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

	var data []byte
	var opCode ws.OpCode
	var err error
	data, opCode, err = ReadData(ctx, cc.conn, nil, timeout, ws.StateServerSide, cc.compression, cc.flateReader)
	return data, opCode, err
}

func (cc *ClientConnection) Write(x interface{}) error {
	cc.ioMutex.Lock()
	defer cc.ioMutex.Unlock()
	state := ws.StateServerSide
	if cc.compression {
		state |= ws.StateExtended
	}
	wsWriter := wsutil.NewWriter(cc.conn, state, ws.OpText)
	var writer io.Writer
	var flateWriter *wsflate.Writer
	if cc.compression {
		var msg wsflate.MessageState
		msg.SetCompressed(true)
		wsWriter.SetExtensions(&msg)
		flateWriter = wsflate.NewWriter(wsWriter, func(w io.Writer) wsflate.Compressor {
			f, err := flate.NewWriterDict(w, DeflateCompressionLevel, GetStaticCompressorDictionary())
			if err != nil {
				log.Error("Failed to create flate writer", "err", err)
			}
			return f
		})
		writer = flateWriter
	} else {
		writer = wsWriter
	}
	encoder := json.NewEncoder(writer)
	if err := encoder.Encode(x); err != nil {
		return err
	}
	if flateWriter != nil {
		if err := flateWriter.Close(); err != nil {
			return errors.Wrap(err, "unable to flush message")
		}
	}
	return wsWriter.Flush()
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
