package storage

import (
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
)

var (
	ErrStorageRace = errors.New("storage race error")

	DataPosterPrefix string = "d" // the prefix for all data poster keys
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
