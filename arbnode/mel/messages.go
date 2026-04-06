// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package mel

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/util/arbmath"
)

var ErrDelayedMessageNotYetFinalized = errors.New("delayed message not yet finalized")
var ErrDelayedAccumulatorMismatch = errors.New("delayed message accumulator mismatch")
var ErrDelayedMessagePreimageNotFound = errors.New("delayed message preimage not found")
var ErrNotImplementedUnderMEL = errors.New("not implemented under MEL")
var ErrAccumulatorNotFound = errors.New("accumulator not found")

type BatchDataLocation uint8

const (
	BatchDataTxInput BatchDataLocation = iota
	BatchDataSeparateEvent
	BatchDataNone
	BatchDataBlobHashes
)

type SequencerInboxBatch struct {
	BlockHash              common.Hash
	ParentChainBlockNumber uint64
	SequenceNumber         uint64
	BeforeInboxAcc         common.Hash
	AfterInboxAcc          common.Hash
	AfterDelayedAcc        common.Hash
	AfterDelayedCount      uint64
	TimeBounds             bridgegen.IBridgeTimeBounds
	RawLog                 types.Log
	DataLocation           BatchDataLocation
	BridgeAddress          common.Address
	Serialized             []byte // nil if serialization isn't cached yet
}

// DelayedInboxMessage represents a delayed message from the parent chain inbox.
// BlockHash may be zero for messages read from legacy (pre-MEL) database keys,
// since the legacy schema did not store this field. Consumers must handle the
// zero-hash case rather than assuming it is always populated.
type DelayedInboxMessage struct {
	BlockHash              common.Hash
	BeforeInboxAcc         common.Hash
	Message                *arbostypes.L1IncomingMessage
	ParentChainBlockNumber uint64
}

func (m *DelayedInboxMessage) AfterInboxAcc() (common.Hash, error) {
	if m.Message == nil || m.Message.Header == nil {
		return common.Hash{}, errors.New("cannot compute AfterInboxAcc: Message or Header is nil")
	}
	hash := crypto.Keccak256(
		[]byte{m.Message.Header.Kind},
		m.Message.Header.Poster.Bytes(),
		arbmath.UintToBytes(m.Message.Header.BlockNumber),
		arbmath.UintToBytes(m.Message.Header.Timestamp),
		m.Message.Header.RequestId.Bytes(),
		arbmath.U256Bytes(m.Message.Header.L1BaseFee),
		crypto.Keccak256(m.Message.L2msg),
	)
	return crypto.Keccak256Hash(m.BeforeInboxAcc[:], hash), nil
}

func (m *DelayedInboxMessage) Hash() (common.Hash, error) {
	encoded, err := rlp.EncodeToBytes(m)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to RLP-encode DelayedInboxMessage: %w", err)
	}
	return crypto.Keccak256Hash(encoded), nil
}

type BatchMetadata struct {
	Accumulator         common.Hash
	MessageCount        arbutil.MessageIndex
	DelayedMessageCount uint64
	ParentChainBlock    uint64
}

type MessageSyncProgress struct {
	BatchSeen           uint64
	BatchSeenIsEstimate bool // true when BatchSeen fell back to headState.BatchCount due to an RPC "header not found" error during on-chain batch count lookup
	BatchProcessed      uint64
	MsgCount            arbutil.MessageIndex
}
