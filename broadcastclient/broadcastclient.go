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
	"time"

	"github.com/gobwas/ws"
	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/arbitrum/packages/arb-util/wsbroadcastserver"
	"github.com/offchainlabs/arbstate/broadcaster"
)

type BroadcastClient struct {
	websocketUrl    string
	lastInboxSeqNum *big.Int

	connMutex *sync.Mutex
	conn      net.Conn

	retryMutex *sync.Mutex
	retryCount int

	retrying                        bool
	shuttingDown                    bool
	ConfirmedSequenceNumberListener chan uint64
	idleTimeout                     time.Duration
}

func NewBroadcastClient(websocketUrl string, lastInboxSeqNum *big.Int, idleTimeout time.Duration) *BroadcastClient {
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
	}
}

func (bc *BroadcastClient) Connect(ctx context.Context) (chan broadcaster.BroadcastFeedMessage, error) {
	messageReceiver := make(chan broadcaster.BroadcastFeedMessage)

	return messageReceiver, bc.ConnectWithChannel(ctx, messageReceiver)
}

func (bc *BroadcastClient) ConnectWithChannel(ctx context.Context, messageReceiver chan broadcaster.BroadcastFeedMessage) error {
	_, err := bc.connect(ctx, messageReceiver)
	if err != nil {
		return err
	}

	bc.startBackgroundReader(ctx, messageReceiver)

	return nil
}

func (bc *BroadcastClient) ConnectInBackground(ctx context.Context, messageReceiver chan broadcaster.BroadcastFeedMessage) {
	go (func() {
		for {
			err := bc.ConnectWithChannel(ctx, messageReceiver)
			if err == nil {
				break
			}
			log.Warn("failed connect to sequencer broadcast, waiting and retrying, url = ", bc.websocketUrl, err.Error())
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
			}
		}
	})()
}

func (bc *BroadcastClient) connect(ctx context.Context, messageReceiver chan broadcaster.BroadcastFeedMessage) (chan broadcaster.BroadcastFeedMessage, error) {

	if len(bc.websocketUrl) == 0 {
		// Nothing to do
		return nil, nil
	}

	log.Info("connecting to arbitrum inbox message broadcaster, url = ", bc.websocketUrl)
	timeoutDialer := ws.Dialer{
		Timeout: 10 * time.Second,
	}

	conn, _, _, err := timeoutDialer.Dial(ctx, bc.websocketUrl)
	if err != nil {
		log.Warn("broadcast client unable to connect", err.Error())
		return nil, errors.Wrap(err, "broadcast client unable to connect")
	}

	bc.connMutex.Lock()
	bc.conn = conn
	bc.connMutex.Unlock()

	log.Info("Connected")

	return messageReceiver, nil
}

func (bc *BroadcastClient) startBackgroundReader(ctx context.Context, messageReceiver chan broadcaster.BroadcastFeedMessage) {
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
					log.Error("Server connection timed out without receiving data, feed = ", bc.websocketUrl)
				} else {
					log.Error("error calling readData, feed = ", bc.websocketUrl, ", opcode = ", int(op), err.Error())
				}
				_ = bc.conn.Close()
				bc.RetryConnect(ctx, messageReceiver)
				continue
			}

			if msg != nil {
				res := broadcaster.BroadcastMessage{}
				err = json.Unmarshal(msg, &res)
				if err != nil {
					log.Error("error unmarshalling message ", msg, err.Error())
					continue
				}

				if len(res.Messages) > 0 {
					log.Debug("received batch item, count", len(res.Messages), ", first seq, ", res.Messages[0].SequenceNumber)
				} else if res.ConfirmedSequenceNumberMessage != nil {
					log.Debug("confirmed sequence number ", res.ConfirmedSequenceNumberMessage.SequenceNumber)
				} else {
					log.Debug("received broadcast with no messages populated, length = ", len(msg))
				}

				if res.Version == 1 {
					for _, message := range res.Messages {
						messageReceiver <- *message
					}

					if res.ConfirmedSequenceNumberMessage != nil && bc.ConfirmedSequenceNumberListener != nil {
						bc.ConfirmedSequenceNumberListener <- res.ConfirmedSequenceNumberMessage.SequenceNumber
					}
				}
			}
		}
	}()
}

func (bc *BroadcastClient) GetRetryCount() int {
	bc.retryMutex.Lock()
	defer bc.retryMutex.Unlock()

	return bc.retryCount
}

func (bc *BroadcastClient) RetryConnect(ctx context.Context, messageReceiver chan broadcaster.BroadcastFeedMessage) {
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

		bc.retryCount++
		_, err := bc.connect(ctx, messageReceiver)
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
