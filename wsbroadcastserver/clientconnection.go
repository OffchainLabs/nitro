// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package wsbroadcastserver

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/broadcaster/backlog"
	m "github.com/offchainlabs/nitro/broadcaster/message"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsflate"
	"github.com/mailru/easygo/netpoll"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

var errContextDone = errors.New("context done")

type message struct {
	data           []byte
	sequenceNumber *arbutil.MessageIndex
}

type ClientConnectionAction struct {
	cc     *ClientConnection
	create bool
}

// ClientConnection represents client connection.
type ClientConnection struct {
	stopwaiter.StopWaiter

	ioMutex  sync.Mutex
	conn     net.Conn
	creation time.Time
	clientIp net.IP

	desc            *netpoll.Desc
	Name            string
	clientAction    chan ClientConnectionAction
	requestedSeqNum arbutil.MessageIndex
	LastSentSeqNum  atomic.Uint64

	lastHeardUnix atomic.Int64
	out           chan message
	backlog       backlog.Backlog
	registered    chan bool
	backlogSent   bool

	compression bool
	flateReader *wsflate.Reader

	delay time.Duration
}

func NewClientConnection(
	conn net.Conn,
	desc *netpoll.Desc,
	clientAction chan ClientConnectionAction,
	requestedSeqNum arbutil.MessageIndex,
	connectingIP net.IP,
	compression bool,
	maxSendQueue int,
	delay time.Duration,
	bklg backlog.Backlog,
) *ClientConnection {
	clientConnection := &ClientConnection{
		conn:            conn,
		clientIp:        connectingIP,
		desc:            desc,
		creation:        time.Now(),
		Name:            fmt.Sprintf("%s@%s-%d", connectingIP, conn.RemoteAddr(), rand.Intn(10)),
		clientAction:    clientAction,
		requestedSeqNum: requestedSeqNum,
		out:             make(chan message, maxSendQueue),
		compression:     compression,
		flateReader:     NewFlateReader(),
		delay:           delay,
		backlog:         bklg,
		registered:      make(chan bool, 1),
		backlogSent:     false,
	}
	clientConnection.lastHeardUnix.Store(time.Now().Unix())
	return clientConnection
}

func (cc *ClientConnection) Age() time.Duration {
	return time.Since(cc.creation)
}

func (cc *ClientConnection) Compression() bool {
	return cc.compression
}

// Register sends the ClientConnection to be registered with the ClientManager.
func (cc *ClientConnection) Register() {
	cc.clientAction <- ClientConnectionAction{
		cc:     cc,
		create: true,
	}
}

// Remove sends the ClientConnection to be removed from the ClientManager.
func (cc *ClientConnection) Remove() {
	cc.clientAction <- ClientConnectionAction{
		cc:     cc,
		create: false,
	}
}

func (cc *ClientConnection) writeBacklog(ctx context.Context, segment backlog.BacklogSegment) error {
	var prevSegment backlog.BacklogSegment
	isFirstSegment := true
	for !backlog.IsBacklogSegmentNil(segment) {
		// must get the next segment before the messages to be sent are
		// retrieved ensures another segment is not added in between calls.
		prevSegment = segment
		segment = segment.Next()

		select {
		case <-ctx.Done():
			return errContextDone
		default:
		}

		msgs := prevSegment.Messages()
		if isFirstSegment && prevSegment.Contains(uint64(cc.requestedSeqNum)) {
			requestedIdx := int(cc.requestedSeqNum) - int(prevSegment.Start())
			// This might be false if messages were added after we fetched the segment's messages
			if len(msgs) >= requestedIdx {
				msgs = msgs[requestedIdx:]
			}
		}
		if len(msgs) == 0 {
			break
		}
		isFirstSegment = false
		bm := &m.BroadcastMessage{
			Version:  m.V1,
			Messages: msgs,
		}
		err := cc.writeBroadcastMessage(bm)
		if err != nil {
			return err
		}

		// do not use prevSegment.End() method, must figure out the last
		// sequence number from the messages that were actually sent in case
		// more messages are added.
		end := uint64(msgs[len(msgs)-1].SequenceNumber)
		cc.LastSentSeqNum.Store(end)
		log.Debug("segment sent to client", "client", cc.Name, "sentCount", len(bm.Messages), "lastSentSeqNum", end)
	}
	return nil
}

func (cc *ClientConnection) writeBroadcastMessage(bm *m.BroadcastMessage) error {
	notCompressed, compressed, err := serializeMessage(bm, !cc.compression, cc.compression)
	if err != nil {
		return err
	}

	var data []byte
	if cc.compression {
		data = compressed.Bytes()
	} else {
		data = notCompressed.Bytes()
	}
	err = cc.writeRaw(data)
	if err != nil {
		return err
	}
	return nil
}

func (cc *ClientConnection) Start(parentCtx context.Context) {
	cc.StopWaiter.Start(parentCtx, cc)
	cc.LaunchThread(func(ctx context.Context) {
		// A delay may be configured, ensures the Broadcaster delays before any
		// messages are sent to the client. The ClientConnection has not been
		// registered so the out channel filling is not a concern.
		if cc.delay != 0 {
			t := time.NewTimer(cc.delay)
			select {
			case <-ctx.Done():
				t.Stop()
				return
			case <-t.C:
			}
		}

		// Send the current backlog before registering the ClientConnection in
		// case the backlog is very large
		segment := cc.backlog.Head()
		if !backlog.IsBacklogSegmentNil(segment) && segment.Start() < uint64(cc.requestedSeqNum) {
			s, err := cc.backlog.Lookup(uint64(cc.requestedSeqNum))
			if err != nil {
				logWarn(err, "error finding requested sequence number in backlog: sending the entire backlog instead")
			} else {
				segment = s
			}
		}
		err := cc.writeBacklog(ctx, segment)
		if errors.Is(err, errContextDone) {
			return
		} else if err != nil {
			logWarn(err, "error writing messages from backlog")
			cc.Remove()
			return
		}

		cc.Register()
		timer := time.NewTimer(5 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-cc.registered:
			timer.Stop()
			log.Debug("ClientConnection registered with ClientManager", "client", cc.Name)
		case <-timer.C:
			log.Error("timed out waiting for ClientConnection to register with ClientManager", "client", cc.Name)
		}

		// broadcast any new messages sent to the out channel
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-cc.out:
				if msg.sequenceNumber != nil && uint64(*msg.sequenceNumber) <= cc.LastSentSeqNum.Load() {
					log.Debug("client has already sent message with this sequence number, skipping the message", "client", cc.Name, "sequence number", *msg.sequenceNumber)
					continue
				}

				expSeqNum := cc.LastSentSeqNum.Load() + 1
				if !cc.backlogSent && msg.sequenceNumber != nil && uint64(*msg.sequenceNumber) > expSeqNum {
					catchupSeqNum := uint64(*msg.sequenceNumber) - 1
					bm, err := cc.backlog.Get(expSeqNum, catchupSeqNum)
					if err != nil {
						logWarn(err, fmt.Sprintf("error reading messages %d to %d from backlog", expSeqNum, catchupSeqNum))
						return
					}

					err = cc.writeBroadcastMessage(bm)
					if err != nil {
						logWarn(err, fmt.Sprintf("error writing messages %d to %d from backlog", expSeqNum, catchupSeqNum))
						cc.Remove()
						return
					}
				}
				cc.backlogSent = true

				err := cc.writeRaw(msg.data)
				if err != nil {
					logWarn(err, "error writing data to client")
					cc.Remove()
					return
				}
			}
		}
	})
}

// Registered is used by the ClientManager to indicate that ClientConnection
// has been registered with the ClientManager
func (cc *ClientConnection) Registered() {
	cc.registered <- true
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
	return time.Unix(cc.lastHeardUnix.Load(), 0)
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

// readRequest reads json-rpc request from connection.
func (cc *ClientConnection) readRequest(ctx context.Context, timeout time.Duration) ([]byte, ws.OpCode, error) {
	cc.ioMutex.Lock()
	defer cc.ioMutex.Unlock()

	cc.lastHeardUnix.Store(time.Now().Unix())

	var data []byte
	var opCode ws.OpCode
	var err error
	data, opCode, err = ReadData(ctx, cc.conn, nil, timeout, ws.StateServerSide, cc.compression, cc.flateReader)
	return data, opCode, err
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
