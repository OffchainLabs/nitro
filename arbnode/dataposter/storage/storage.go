// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package storage

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/nitro/arbutil"
)

var (
	ErrStorageRace = errors.New("storage race error")

	BlockValidatorPrefix string = "v" // the prefix for all block validator keys
	StakerPrefix         string = "S" // the prefix for all staker keys
	BatchPosterPrefix    string = "b" // the prefix for all batch poster keys
	// TODO(anodar): move everything else from schema.go file to here once
	// execution split is complete.
)

type QueuedTransaction struct {
	FullTx           *types.Transaction
	DeprecatedData   types.DynamicFeeTx // FullTx should be used instead
	Meta             []byte
	Sent             bool
	Created          time.Time // may be earlier than the tx was given to the tx poster
	NextReplacement  time.Time
	CumulativeWeight uint64 // a rough estimate of the total number of batches submitted at this point, not guaranteed to be exact
}

type queuedTransactionForEncoding struct {
	FullTx           *types.Transaction
	Data             types.DynamicFeeTx
	Meta             []byte
	Sent             bool
	Created          RlpTime
	NextReplacement  RlpTime
	CumulativeWeight *uint64 `rlp:"optional"`
}

func (qt *QueuedTransaction) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, queuedTransactionForEncoding{
		FullTx:           qt.FullTx,
		Data:             qt.DeprecatedData,
		Meta:             qt.Meta,
		Sent:             qt.Sent,
		Created:          (RlpTime)(qt.Created),
		NextReplacement:  (RlpTime)(qt.NextReplacement),
		CumulativeWeight: &qt.CumulativeWeight,
	})
}

func (qt *QueuedTransaction) DecodeRLP(s *rlp.Stream) error {
	var qtEnc queuedTransactionForEncoding
	if err := s.Decode(&qtEnc); err != nil {
		return err
	}
	qt.FullTx = qtEnc.FullTx
	qt.DeprecatedData = qtEnc.Data
	qt.Meta = qtEnc.Meta
	qt.Sent = qtEnc.Sent
	qt.Created = time.Time(qtEnc.Created)
	qt.NextReplacement = time.Time(qtEnc.NextReplacement)
	if qtEnc.CumulativeWeight != nil {
		qt.CumulativeWeight = *qtEnc.CumulativeWeight
	} else {
		qt.CumulativeWeight = qt.FullTx.Nonce()
	}
	return nil
}

// LegacyQueuedTransaction is used for backwards compatibility.
// Before https://github.com/OffchainLabs/nitro/pull/1773: the queuedTransaction
// looked like this and was rlp encoded directly. After the pr, we are store
// rlp encoding of Meta into queuedTransaction and rlp encoding it once more
// to store it.
type LegacyQueuedTransaction struct {
	FullTx          *types.Transaction
	Data            types.DynamicFeeTx
	Meta            BatchPosterPosition
	Sent            bool
	Created         time.Time // may be earlier than the tx was given to the tx poster
	NextReplacement time.Time
}

// This is also for legacy reason. Since Batchposter is in arbnode package,
// we can't refer to BatchPosterPosition type there even if we export it (that
// would create cyclic dependency).
// We'll drop this struct in a few releases when we drop legacy encoding.
type BatchPosterPosition struct {
	MessageCount        arbutil.MessageIndex
	DelayedMessageCount uint64
	NextSeqNum          uint64
}

func DecodeLegacyQueuedTransaction(data []byte) (*LegacyQueuedTransaction, error) {
	var val LegacyQueuedTransaction
	if err := rlp.DecodeBytes(data, &val); err != nil {
		return nil, fmt.Errorf("decoding legacy queued transaction: %w", err)
	}
	return &val, nil
}

func LegacyToQueuedTransaction(legacyQT *LegacyQueuedTransaction) (*QueuedTransaction, error) {
	meta, err := rlp.EncodeToBytes(legacyQT.Meta)
	if err != nil {
		return nil, fmt.Errorf("converting legacy to queued transaction: %w", err)
	}
	return &QueuedTransaction{
		FullTx:          legacyQT.FullTx,
		DeprecatedData:  legacyQT.Data,
		Meta:            meta,
		Sent:            legacyQT.Sent,
		Created:         legacyQT.Created,
		NextReplacement: legacyQT.NextReplacement,
	}, nil
}

func QueuedTransactionToLegacy(qt *QueuedTransaction) (*LegacyQueuedTransaction, error) {
	if qt == nil {
		return nil, nil
	}
	var meta BatchPosterPosition
	if qt.Meta != nil {
		if err := rlp.DecodeBytes(qt.Meta, &meta); err != nil {
			return nil, fmt.Errorf("converting queued transaction to legacy: %w", err)
		}
	}
	return &LegacyQueuedTransaction{
		FullTx:          qt.FullTx,
		Data:            qt.DeprecatedData,
		Meta:            meta,
		Sent:            qt.Sent,
		Created:         qt.Created,
		NextReplacement: qt.NextReplacement,
	}, nil
}

// Decode tries to decode QueuedTransaction, if that fails it tries to decode
// into legacy queued transaction and converts to queued
func decode(data []byte) (*QueuedTransaction, error) {
	var item QueuedTransaction
	if err := rlp.DecodeBytes(data, &item); err != nil {
		log.Debug("Failed to decode QueuedTransaction, attempting to decide legacy queued transaction", "error", err)
		val, err := DecodeLegacyQueuedTransaction(data)
		if err != nil {
			return nil, fmt.Errorf("decoding legacy item: %w", err)
		}
		log.Debug("Succeeded decoding QueuedTransaction with legacy encoder")
		return LegacyToQueuedTransaction(val)
	}
	return &item, nil
}

type EncoderDecoder struct{}

func (e *EncoderDecoder) Encode(qt *QueuedTransaction) ([]byte, error) {
	return rlp.EncodeToBytes(qt)
}

func (e *EncoderDecoder) Decode(data []byte) (*QueuedTransaction, error) {
	return decode(data)
}

type LegacyEncoderDecoder struct{}

func (e *LegacyEncoderDecoder) Encode(qt *QueuedTransaction) ([]byte, error) {
	legacyQt, err := QueuedTransactionToLegacy(qt)
	if err != nil {
		return nil, fmt.Errorf("encoding legacy item: %w", err)
	}
	return rlp.EncodeToBytes(legacyQt)
}

func (le *LegacyEncoderDecoder) Decode(data []byte) (*QueuedTransaction, error) {
	return decode(data)
}

// Typically interfaces belong to where they are being used, not at implementing
// site, but this is used in all storages (besides no-op) and all of them
// require all the functions for this interface.
type EncoderDecoderInterface interface {
	Encode(*QueuedTransaction) ([]byte, error)
	Decode([]byte) (*QueuedTransaction, error)
}

// EncoderDecoderF is a function type that returns encoder/decoder interface.
// This is needed to implement hot-reloading flag to switch encoding/decoding
// strategy on the fly.
type EncoderDecoderF func() EncoderDecoderInterface
