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
	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/arbos/l1pricing"
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
			Poster:      l1pricing.SequencerAddress,
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

	headerChan, cancel := HeaderSubscribeWithRetry(ctx, s.l1Client)
	defer cancel()

	go (func() {
		for {
			select {
			case header, ok := <-headerChan:
				if !ok {
					return
				}
				atomic.StoreUint64(&s.l1BlockNumber, header.Number.Uint64())
			case <-ctx.Done():
				return
			}
		}
	})()

	return nil
}
