// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"encoding/binary"
	"errors"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
)

// TxResult pairs an attempted tx with its sequencing error (nil if it made it into the block).
type TxResult struct {
	Tx  *types.Transaction
	Err error
	// Timeboosted reports whether the tx was sequenced through the express lane.
	Timeboosted bool
}

// BlockSequencingHooks is the per-block hooks view the execution engine uses to sequence transactions.
type BlockSequencingHooks interface {
	arbos.SequencingHooks
	// SequencedTxes returns one entry per attempted tx, in order.
	SequencedTxes() ([]TxResult, error)
}

// MessageFromTxes builds the L2 message from the txs that made it into the block.
func MessageFromTxes(header *arbostypes.L1IncomingMessageHeader, txes []TxResult) (*arbostypes.L1IncomingMessage, error) {
	var l2Message []byte
	if len(txes) == 1 && txes[0].Err == nil {
		txBytes, err := txes[0].Tx.MarshalBinary()
		if err != nil {
			return nil, err
		}
		l2Message = make([]byte, 0, 1+len(txBytes))
		l2Message = append(l2Message, arbos.L2MessageKind_SignedTx)
		l2Message = append(l2Message, txBytes...)
	} else {
		msgSize := 1
		for _, res := range txes {
			if res.Err == nil {
				// #nosec G115
				msgSize += 9 + int(res.Tx.Size())
			}
		}
		l2Message = make([]byte, 0, msgSize)
		l2Message = append(l2Message, arbos.L2MessageKind_Batch)
		sizeBuf := make([]byte, 8)
		for _, res := range txes {
			if res.Err != nil {
				continue
			}
			txBytes, err := res.Tx.MarshalBinary()
			if err != nil {
				return nil, err
			}
			// #nosec G115
			binary.BigEndian.PutUint64(sizeBuf, uint64(len(txBytes)+1))
			l2Message = append(l2Message, sizeBuf...)
			l2Message = append(l2Message, arbos.L2MessageKind_SignedTx)
			l2Message = append(l2Message, txBytes...)
		}
	}
	if len(l2Message) > arbostypes.MaxL2MessageSize {
		return nil, errors.New("l2message too long")
	}
	return &arbostypes.L1IncomingMessage{
		Header: header,
		L2msg:  l2Message,
	}, nil
}
