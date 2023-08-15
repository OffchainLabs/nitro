package storage

import (
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
