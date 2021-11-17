//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbnode

import (
	"context"
	"math/big"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/arbstate"
	"github.com/pkg/errors"
)

type Sequencer struct {
	txStreamer    *TransactionStreamer
	l1Client      L1Interface
	l1BlockNumber uint64
}

func NewSequencer(txStreamer *TransactionStreamer, l1Client L1Interface) (*Sequencer, error) {
	return &Sequencer{
		txStreamer:    txStreamer,
		l1Client:      l1Client,
		l1BlockNumber: 0,
	}, nil
}

func (s *Sequencer) PublishTransaction(ctx context.Context, tx *types.Transaction) error {
	txBytes, err := tx.MarshalBinary()
	if err != nil {
		return err
	}
	var l2Message []byte
	l2Message = append(l2Message, arbos.L2MessageKind_SignedTx)
	l2Message = append(l2Message, txBytes...)
	timestamp := common.BigToHash(new(big.Int).SetInt64(time.Now().Unix()))
	l1Block := atomic.LoadUint64(&s.l1BlockNumber)
	if s.l1Client != nil && l1Block == 0 {
		return errors.New("unknown L1 block")
	}
	message := &arbos.L1IncomingMessage{
		Header: &arbos.L1IncomingMessageHeader{
			Kind:        arbos.L1MessageType_L2Message,
			Sender:      arbstate.SequencerAddress,
			BlockNumber: common.BigToHash(new(big.Int).SetUint64(l1Block)),
			Timestamp:   timestamp,
			RequestId:   common.Hash{},
			GasPriceL1:  common.Hash{},
		},
		L2msg: l2Message,
	}

	return s.txStreamer.SequenceMessages([]*arbos.L1IncomingMessage{message})
}

func (s *Sequencer) Initialize(ctx context.Context) error {
	if s.l1Client == nil {
		return nil
	}

	block, err := s.l1Client.HeaderByNumber(ctx, nil)
	if err != nil {
		return err
	}
	atomic.StoreUint64(&s.l1BlockNumber, block.Number.Uint64())
	return nil
}

func (s *Sequencer) Start(ctx context.Context) error {
	if s.l1Client == nil {
		return nil
	}

	initialBlockNr := atomic.LoadUint64(&s.l1BlockNumber)
	if initialBlockNr == 0 {
		return errors.New("ArbInterface: not initialized")
	}

	headerChan := make(chan *types.Header)
	headerSubscription, err := s.l1Client.SubscribeNewHead(ctx, headerChan)
	if err != nil {
		return err
	}

	go (func() {
		for {
			select {
			case header := <-headerChan:
				atomic.StoreUint64(&s.l1BlockNumber, header.Number.Uint64())
			case err := <-headerSubscription.Err():
				log.Warn("error in subscription to L1 headers", "err", err)
				for {
					headerSubscription, err = s.l1Client.SubscribeNewHead(ctx, headerChan)
					if err != nil {
						log.Warn("error re-subscribing to L1 headers", "err", err)
						select {
						case <-ctx.Done():
							return
						case <-time.After(time.Second):
						}
					} else {
						break
					}
				}
			case <-ctx.Done():
				headerSubscription.Unsubscribe()
				return
			}
		}
	})()

	return nil
}
