package melrunner

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/nitro/arbnode/mel"
	melextraction "github.com/offchainlabs/nitro/arbnode/mel/extraction"
)

var _ = melextraction.MELDataProvider(&ArbOSExtractionRecorder{})

type ArbOSExtractionRecorder struct {
	melDB      *Database
	txFetcher  *txByLogFetcher
	preFetcher *logsFetcher
}

func (ar *ArbOSExtractionRecorder) ReadDelayedMessage(
	ctx context.Context,
	state *mel.State,
	index uint64,
) (*mel.DelayedInboxMessage, error) {
	return ar.melDB.ReadDelayedMessage(ctx, state, index)
}

// Defines methods that can fetch all the logs of a parent chain block
// and logs corresponding to a specific transaction in a parent chain block.
func (ar *ArbOSExtractionRecorder) LogsForBlockHash(
	ctx context.Context,
	parentChainBlockHash common.Hash,
) ([]*types.Log, error) {
	return ar.preFetcher.LogsForBlockHash(ctx, parentChainBlockHash)
}
func (ar *ArbOSExtractionRecorder) LogsForTxIndex(
	ctx context.Context,
	parentChainBlockHash common.Hash,
	txIndex uint,
) ([]*types.Log, error) {
	return ar.preFetcher.LogsForTxIndex(ctx, parentChainBlockHash, txIndex)
}

func (ar *ArbOSExtractionRecorder) TransactionByLog(
	ctx context.Context,
	log *types.Log,
) (*types.Transaction, error) {
	return ar.txFetcher.TransactionByLog(ctx, log)
}
