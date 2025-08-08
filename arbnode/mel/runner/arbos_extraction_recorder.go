package melrunner

import (
	"context"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/nitro/arbnode/mel"
	melextraction "github.com/offchainlabs/nitro/arbnode/mel/extraction"
)

var _ = melextraction.MELDataProvider(&ArbOSExtractionRecorder{})

type ArbOSExtractionRecorder struct {
	sync.Mutex
	melDB      *Database
	txFetcher  *txByLogFetcher
	preFetcher *logsFetcher
	caches     *caches
}

type caches struct {
	delayedMsgs   map[uint64]*mel.DelayedInboxMessage
	logsByHash    map[common.Hash][]*types.Log
	logsByTxIndex map[common.Hash]map[uint][]*types.Log
	txsByHash     map[common.Hash]*types.Transaction
}

// TODO: Should prune the caches to avoid memory leaks.
func (ar *ArbOSExtractionRecorder) Prune() {

}

func (ar *ArbOSExtractionRecorder) ReadDelayedMessage(
	ctx context.Context,
	state *mel.State,
	index uint64,
) (*mel.DelayedInboxMessage, error) {
	ar.Lock()
	defer ar.Unlock()
	msg, ok := ar.caches.delayedMsgs[index]
	if ok {
		return msg, nil
	}
	result, err := ar.melDB.ReadDelayedMessage(ctx, state, index)
	if err != nil {
		return nil, err
	}
	ar.caches.delayedMsgs[index] = result
	return result, nil
}

// Defines methods that can fetch all the logs of a parent chain block
// and logs corresponding to a specific transaction in a parent chain block.
func (ar *ArbOSExtractionRecorder) LogsForBlockHash(
	ctx context.Context,
	parentChainBlockHash common.Hash,
) ([]*types.Log, error) {
	ar.Lock()
	defer ar.Unlock()
	logs, ok := ar.caches.logsByHash[parentChainBlockHash]
	if ok {
		return logs, nil
	}
	result, err := ar.preFetcher.LogsForBlockHash(ctx, parentChainBlockHash)
	if err != nil {
		return nil, err
	}
	ar.caches.logsByHash[parentChainBlockHash] = result
	return result, nil
}

func (ar *ArbOSExtractionRecorder) LogsForTxIndex(
	ctx context.Context,
	parentChainBlockHash common.Hash,
	txIndex uint,
) ([]*types.Log, error) {
	ar.Lock()
	defer ar.Unlock()
	innerMap, ok := ar.caches.logsByTxIndex[parentChainBlockHash]
	if ok {
		logs, ok := innerMap[txIndex]
		if ok {
			return logs, nil
		}
	}
	logs, err := ar.preFetcher.LogsForTxIndex(ctx, parentChainBlockHash, txIndex)
	if err != nil {
		return nil, err
	}
	if ar.caches.logsByTxIndex[parentChainBlockHash] == nil {
		ar.caches.logsByTxIndex[parentChainBlockHash] = make(map[uint][]*types.Log)
	}
	ar.caches.logsByTxIndex[parentChainBlockHash][txIndex] = logs
	return logs, nil
}

func (ar *ArbOSExtractionRecorder) TransactionByLog(
	ctx context.Context,
	log *types.Log,
) (*types.Transaction, error) {
	ar.Lock()
	defer ar.Unlock()
	tx, ok := ar.caches.txsByHash[log.TxHash]
	if ok {
		return tx, nil
	}
	result, err := ar.txFetcher.TransactionByLog(ctx, log)
	if err != nil {
		return nil, err
	}
	ar.caches.txsByHash[log.TxHash] = result
	return result, nil
}
