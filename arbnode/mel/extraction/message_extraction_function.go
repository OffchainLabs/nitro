package melextraction

import (
	"context"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/daprovider"
)

// Defines a method that can read a delayed message from an external database.
type DelayedMessageDatabase interface {
	ReadDelayedMessage(
		ctx context.Context,
		state *mel.State,
		index uint64,
	) (*arbnode.DelayedInboxMessage, error)
}

// Defines a method that can fetch the receipt for a specific
// transaction index in a parent chain block.
type ReceiptFetcher interface {
	ReceiptForTransactionIndex(
		ctx context.Context,
		txIndex uint,
	) (*types.Receipt, error)
}

// ExtractMessages is a pure function that can read a parent chain block and
// and input MEL state to run a specific algorithm that extracts Arbitrum messages and
// delayed messages observed from transactions in the block. This function can be proven
// through a replay binary, and should also compile to WAVM in addition to running in native mode.
func ExtractMessages(
	ctx context.Context,
	inputState *mel.State,
	parentChainBlock *types.Block,
	dataProviders []daprovider.Reader,
	delayedMsgDatabase DelayedMessageDatabase,
	receiptFetcher ReceiptFetcher,
) (*mel.State, []*arbostypes.MessageWithMetadata, []*arbnode.DelayedInboxMessage, error) {
	return nil, nil, nil, nil
}
