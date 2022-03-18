//
// Copyright 2021-2022, Offchain Labs, Inc. All rights reserved.
//

package broadcastclient

import (
	"context"
	"errors"
	"math/big"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/broadcaster"
	"github.com/offchainlabs/nitro/util"
)

type BroadcastClientConfig struct {
	Timeout time.Duration
	URLs    []string
}

var DefaultBroadcastClientConfig BroadcastClientConfig

type TransactionStreamerInterface interface {
	AddMessages(pos arbutil.MessageIndex, force bool, messages []arbstate.MessageWithMetadata) error
}

type BroadcastClient struct {
	util.StopWaiter

	websocketUrl    string
	lastInboxSeqNum *big.Int

	retryCount int64

	ConfirmedSequenceNumberListener chan arbutil.MessageIndex
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
		idleTimeout:     idleTimeout,
		txStreamer:      txStreamer,
	}
}

func (bc *BroadcastClient) Start(ctxIn context.Context) {
	bc.StopWaiter.Start(ctxIn)
	maxWaitDuration := 15 * time.Second
	waitDuration := 500 * time.Millisecond

	bc.CallIteratively(func(ctx context.Context) time.Duration {
		timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		conn, _, err := websocket.Dial(timeoutCtx, bc.websocketUrl, &websocket.DialOptions{})
		cancel()
		if err != nil {
			log.Warn("failed connect to sequencer broadcast, waiting and retrying", "url", bc.websocketUrl, "err", err)
		} else if err := bc.readMessages(ctx, conn); err != nil && !errors.Is(err, context.Canceled) {
			log.Warn("error getting broadcast message from broadcast feed", "err", err)
		}
		if waitDuration < maxWaitDuration {
			waitDuration += 500 * time.Millisecond
		}
		atomic.AddInt64(&bc.retryCount, 1)
		return waitDuration
	})
}

func (bc *BroadcastClient) readMessages(ctx context.Context, conn *websocket.Conn) error {
	for {
		res := broadcaster.BroadcastMessage{}
		readContext, cancel := context.WithTimeout(ctx, bc.idleTimeout)
		if err := wsjson.Read(readContext, conn, &res); err != nil {
			cancel()
			return err
		}
		cancel()

		if len(res.Messages) > 0 {
			log.Debug("received batch item", "count", len(res.Messages), "first seq", res.Messages[0].SequenceNumber)
		} else if res.ConfirmedSequenceNumberMessage != nil {
			log.Debug("confirmed sequence number", "seq", res.ConfirmedSequenceNumberMessage.SequenceNumber)
		} else {
			log.Debug("received broadcast with no messages populated")
		}

		if res.Version == 1 {
			if len(res.Messages) > 0 {
				var messages []arbstate.MessageWithMetadata
				for _, message := range res.Messages {
					messages = append(messages, message.Message)
				}
				if err := bc.txStreamer.AddMessages(res.Messages[0].SequenceNumber, false, messages); err != nil {
					log.Error("Error adding message from Sequencer Feed", "err", err)
				}
			}
			if res.ConfirmedSequenceNumberMessage != nil && bc.ConfirmedSequenceNumberListener != nil {
				bc.ConfirmedSequenceNumberListener <- res.ConfirmedSequenceNumberMessage.SequenceNumber
			}
		}
	}
}

func (bc *BroadcastClient) GetRetryCount() int64 {
	return atomic.LoadInt64(&bc.retryCount)
}
