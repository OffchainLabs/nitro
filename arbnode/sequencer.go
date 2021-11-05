//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbnode

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/arbstate"
)

type Sequencer struct {
	inbox *InboxState
}

func NewSequencer(inbox *InboxState) *Sequencer {
	return &Sequencer{
		inbox: inbox,
	}
}

func (s *Sequencer) PublishTransaction(tx *types.Transaction) error {
	txBytes, err := tx.MarshalBinary()
	if err != nil {
		return err
	}
	var l2Message []byte
	l2Message = append(l2Message, arbos.L2MessageKind_SignedTx)
	l2Message = append(l2Message, txBytes...)
	timestamp := common.BigToHash(new(big.Int).SetInt64(time.Now().Unix()))
	message := &arbos.L1IncomingMessage{
		Header: &arbos.L1IncomingMessageHeader{
			Kind:        arbos.L1MessageType_L2Message,
			Sender:      arbstate.SequencerAddress,
			BlockNumber: common.Hash{}, // TODO L1 block number
			Timestamp:   timestamp,
			RequestId:   common.Hash{},
			GasPriceL1:  common.Hash{},
		},
		L2msg: l2Message,
	}

	return s.inbox.SequenceMessages([]*arbos.L1IncomingMessage{message})
}

func (s *Sequencer) InboxState() *InboxState {
	return s.inbox
}

func (s *Sequencer) BlockChain() *core.BlockChain {
	return s.inbox.bc
}
