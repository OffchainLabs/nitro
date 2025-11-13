package mel

import (
	"github.com/ethereum/go-ethereum/arbkeccak"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/util/arbmath"
)

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

type DelayedInboxMessage struct {
	BlockHash              common.Hash
	BeforeInboxAcc         common.Hash
	Message                *arbostypes.L1IncomingMessage
	ParentChainBlockNumber uint64
}

func (m *DelayedInboxMessage) AfterInboxAcc() common.Hash {
	hash := arbkeccak.Keccak256(
		[]byte{m.Message.Header.Kind},
		m.Message.Header.Poster.Bytes(),
		arbmath.UintToBytes(m.Message.Header.BlockNumber),
		arbmath.UintToBytes(m.Message.Header.Timestamp),
		m.Message.Header.RequestId.Bytes(),
		arbmath.U256Bytes(m.Message.Header.L1BaseFee),
		arbkeccak.Keccak256(m.Message.L2msg),
	)
	return arbkeccak.Keccak256Hash(m.BeforeInboxAcc[:], hash)
}

// Hash will replace AfterInboxAcc
func (m *DelayedInboxMessage) Hash() common.Hash {
	hash := arbkeccak.Keccak256(
		[]byte{m.Message.Header.Kind},
		m.Message.Header.Poster.Bytes(),
		arbmath.UintToBytes(m.Message.Header.BlockNumber),
		arbmath.UintToBytes(m.Message.Header.Timestamp),
		m.Message.Header.RequestId.Bytes(),
		arbmath.U256Bytes(m.Message.Header.L1BaseFee),
		arbkeccak.Keccak256(m.Message.L2msg),
	)
	return arbkeccak.Keccak256Hash(hash)
}

type BatchMetadata struct {
	Accumulator         common.Hash
	MessageCount        arbutil.MessageIndex
	DelayedMessageCount uint64
	ParentChainBlock    uint64
}
