//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbnode

import (
	"context"
	"math/big"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/arbstate"
)

type ArbInterface struct {
	txStreamer    *TransactionStreamer
	l1Client      L1Interface
	l1BlockNumber uint64
}

func NewArbInterface(txStreamer *TransactionStreamer, l1Client L1Interface) (*ArbInterface, error) {
	return &ArbInterface{
		txStreamer:    txStreamer,
		l1Client:      l1Client,
		l1BlockNumber: 0,
	}, nil
}

func (a *ArbInterface) PublishTransaction(tx *types.Transaction) error {
	txBytes, err := tx.MarshalBinary()
	if err != nil {
		return err
	}
	var l2Message []byte
	l2Message = append(l2Message, arbos.L2MessageKind_SignedTx)
	l2Message = append(l2Message, txBytes...)
	timestamp := common.BigToHash(new(big.Int).SetInt64(time.Now().Unix()))
	l1Block := atomic.LoadUint64(&a.l1BlockNumber)
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

	return a.txStreamer.SequenceMessages([]*arbos.L1IncomingMessage{message})
}

func (a *ArbInterface) TransactionStreamer() *TransactionStreamer {
	return a.txStreamer
}

func (a *ArbInterface) BlockChain() *core.BlockChain {
	return a.txStreamer.bc
}

func (a *ArbInterface) Initialize(ctx context.Context) error {
	if a.l1Client == nil {
		return nil
	}

	block, err := a.l1Client.HeaderByNumber(ctx, nil)
	if err != nil {
		return err
	}
	atomic.StoreUint64(&a.l1BlockNumber, block.Number.Uint64())
	return nil
}

func (a *ArbInterface) Start(parentCtx context.Context) (*Stopper, error) {
	if a.l1Client == nil {
		return nil, nil
	}

	initialBlockNr := atomic.LoadUint64(&a.l1BlockNumber)
	if initialBlockNr == 0 {
		return nil, errors.New("ArbInterface: not initialized")
	}

	headerChan := make(chan *types.Header)
	headerSubscription, err := a.l1Client.SubscribeNewHead(parentCtx, headerChan)
	if err != nil {
		return nil, err
	}

	stopper, ctx := NewStopper(parentCtx, "Arb Interface")
	go func() {
		defer stopper.Close()
		for {
			select {
			case header := <-headerChan:
				atomic.StoreUint64(&a.l1BlockNumber, header.Number.Uint64())
			case err := <-headerSubscription.Err():
				log.Warn("error in subscription to L1 headers", "err", err)
				for {
					headerSubscription, err = a.l1Client.SubscribeNewHead(ctx, headerChan)
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
	}()

	return stopper, nil
}
