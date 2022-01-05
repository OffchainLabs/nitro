//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package broadcastclient

import (
	"context"
	"encoding/json"
	"math/big"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gobwas/ws"
	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/arbstate/arbstate"
	"github.com/offchainlabs/arbstate/broadcaster"
	"github.com/offchainlabs/arbstate/wsbroadcastserver"
)

type BroadcastClientConfig struct {
	Timeout time.Duration
	URL     string // TODO should this be an array for multiple clients?
}

var DefaultBroadcastClientConfig BroadcastClientConfig

type TransactionStreamerInterface interface {
	AddMessages(pos uint64, force bool, messages []arbstate.MessageWithMetadata) error
}

type BroadcastClient struct {
	websocketUrl    string
	lastInboxSeqNum *big.Int

	connMutex *sync.Mutex
	conn      net.Conn

	retryMutex *sync.Mutex
	retryCount int64

	retrying                        bool
	shuttingDown                    bool
	ConfirmedSequenceNumberListener chan uint64
	idleTimeout                     time.Duration
	txStreamer                      TransactionStreamerInterface
}

func NewBroadcastClient(websocketUrl string, lastInboxSeqNum *big.Int, idleTimeout time.Duration, txStreamer TransactionStreamerInterface) *BroadcastClient {
	var seqNum *big.Int
	if lastInboxSeqNum == nil {
		seqNum = big.NewInt(0)
	} else {
		seqNum = lastInboxSeqNum
	}

	return &BroadcastClient{
		websocketUrl:    websocketUrl,
		lastInboxSeqNum: seqNum,
		connMutex:       &sync.Mutex{},
		retryMutex:      &sync.Mutex{},
		idleTimeout:     idleTimeout,
		txStreamer:      txStreamer,
	}
}

func (bc *BroadcastClient) Start(ctx context.Context) {
	go (func() {
		for {
			err := bc.connect(ctx)
			if err == nil {
				bc.startBackgroundReader(ctx)
				break
			}
			log.Warn("failed connect to sequencer broadcast, waiting and retrying", "url", bc.websocketUrl, "err", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
			}
		}
	})()
}

func (bc *BroadcastClient) connect(ctx context.Context) error {
	if len(bc.websocketUrl) == 0 {
		// Nothing to do
		return nil
	}

	log.Info("connecting to arbitrum inbox message broadcaster", "url", bc.websocketUrl)
	timeoutDialer := ws.Dialer{
		Timeout: 10 * time.Second,
	}

	conn, _, _, err := timeoutDialer.Dial(ctx, bc.websocketUrl)
	if err != nil {
		return errors.Wrap(err, "broadcast client unable to connect")
	}

	bc.connMutex.Lock()
	bc.conn = conn
	bc.connMutex.Unlock()

	log.Info("Connected")

	return nil
}

func (bc *BroadcastClient) startBackgroundReader(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			msg, op, err := wsbroadcastserver.ReadData(ctx, bc.conn, bc.idleTimeout, ws.StateClientSide)
			if err != nil {
				if bc.shuttingDown {
					return
				}
				if strings.Contains(err.Error(), "i/o timeout") {
					log.Error("Server connection timed out without receiving data", "url", bc.websocketUrl, "err", err)
				} else {
					log.Error("error calling readData", "url", bc.websocketUrl, "opcode", int(op), "err", err)
				}
				_ = bc.conn.Close()
				bc.retryConnect(ctx)
				continue
			}

			if msg != nil {
				res := broadcaster.BroadcastMessage{}
				err = json.Unmarshal(msg, &res)
				if err != nil {
					log.Error("error unmarshalling message", "msg", msg, "err", err)
					continue
				}

				if len(res.Messages) > 0 {
					log.Debug("received batch item", "count", len(res.Messages), "first seq", res.Messages[0].SequenceNumber)
				} else if res.ConfirmedSequenceNumberMessage != nil {
					log.Debug("confirmed sequence number", "seq", res.ConfirmedSequenceNumberMessage.SequenceNumber)
				} else {
					log.Debug("received broadcast with no messages populated", "length", len(msg))
				}

				if res.Version == 1 {
					for _, message := range res.Messages {
						if err := bc.txStreamer.AddMessages(message.SequenceNumber, false, []arbstate.MessageWithMetadata{message.Message}); err != nil {
							log.Error("Error adding message from Sequencer Feed", "err", err)
						}
					}

					if res.ConfirmedSequenceNumberMessage != nil && bc.ConfirmedSequenceNumberListener != nil {
						bc.ConfirmedSequenceNumberListener <- res.ConfirmedSequenceNumberMessage.SequenceNumber
					}
				}
			}
		}
	}()
}

func (bc *BroadcastClient) GetRetryCount() int64 {
	return atomic.LoadInt64(&bc.retryCount)
}

func (bc *BroadcastClient) retryConnect(ctx context.Context) {
	bc.retryMutex.Lock()
	defer bc.retryMutex.Unlock()

	maxWaitDuration := 15 * time.Second
	waitDuration := 500 * time.Millisecond
	bc.retrying = true
	for !bc.shuttingDown {
		select {
		case <-ctx.Done():
			return
		case <-time.After(waitDuration):
		}

		atomic.AddInt64(&bc.retryCount, 1)
		err := bc.connect(ctx)
		if err == nil {
			bc.retrying = false
			return
		}

		if waitDuration < maxWaitDuration {
			waitDuration += 500 * time.Millisecond
		}
	}
}

func (bc *BroadcastClient) Close() {
	log.Debug("closing broadcaster client connection")
	bc.shuttingDown = true
	bc.connMutex.Lock()
	if bc.conn != nil {
		_ = bc.conn.Close()
	}
	bc.connMutex.Unlock()
}
