package storage

import (
	"bytes"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
)

var (
	ErrStorageRace = errors.New("storage race error")

	BlockValidatorPrefix string = "v" // the prefix for all block validator keys
	BatchPosterPrefix    string = "b" // the prefix for all batch poster keys
	// TODO(anodar): move everything else from schema.go file to here once
	// execution split is complete.
)

type QueuedTransaction struct {
	FullTx          *types.Transaction `rlp:"nil"`
	Data            types.DynamicFeeTx
	Meta            []byte
	Sent            bool
	Created         time.Time // may be earlier than the tx was given to the tx poster
	NextReplacement time.Time
}

func (qt *QueuedTransaction) Equals(v *QueuedTransaction) bool {
	if (qt != nil) != (v != nil) {
		return false
	}
	if qt == nil {
		return true
	}
	if (qt.FullTx != nil) != (v.FullTx != nil) {
		return false
	}
	if qt.FullTx != nil && qt.FullTx.Hash() != v.FullTx.Hash() {
		return false
	}
	return bytes.Equal(qt.Meta, v.Meta) &&
		qt.Sent == v.Sent &&
		qt.Created.Equal(v.Created) &&
		qt.NextReplacement.Equal(v.NextReplacement)
}
