package melrunner

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/arbnode/mel"
	melextraction "github.com/offchainlabs/nitro/arbnode/mel/extraction"
)

type logsFetcher struct {
	parentChainReader ParentChainReader
	blockHeight       uint64
	blocksToFetch     uint64
	chainHeight       uint64
	logsByTxIndex     map[common.Hash]map[uint][]*types.Log
	logsByBlockHash   map[common.Hash][]*types.Log
}

func newLogsFetcher(parentChainReader ParentChainReader, blocksToFetch uint64) *logsFetcher {
	return &logsFetcher{
		parentChainReader: parentChainReader,
		blocksToFetch:     blocksToFetch,
		logsByTxIndex:     make(map[common.Hash]map[uint][]*types.Log),
		logsByBlockHash:   make(map[common.Hash][]*types.Log),
	}
}

func (f *logsFetcher) LogsForBlockHash(_ context.Context, parentChainBlockHash common.Hash) ([]*types.Log, error) {
	return f.logsByBlockHash[parentChainBlockHash], nil
}

func (f *logsFetcher) LogsForTxIndex(_ context.Context, parentChainBlockHash common.Hash, txIndex uint) ([]*types.Log, error) {
	if txIndexToLogs, ok := f.logsByTxIndex[parentChainBlockHash]; ok {
		if logs, ok := txIndexToLogs[txIndex]; ok {
			return logs, nil
		}
	}
	return nil, fmt.Errorf("logs for tx in block: %v with index: %d not found", parentChainBlockHash, txIndex)
}

func (f *logsFetcher) fetch(ctx context.Context, preState *mel.State) error {
	parentChainBlockNumber := preState.ParentChainBlockNumber + 1
	if parentChainBlockNumber <= f.blockHeight {
		return nil
	}
	f.reset() // prune old logs
	toBlockheight := parentChainBlockNumber + f.blocksToFetch
	if toBlockheight > f.chainHeight {
		head, err := f.parentChainReader.HeaderByNumber(ctx, nil)
		if err != nil {
			return err
		}
		if head.Number.Uint64() < parentChainBlockNumber {
			return fmt.Errorf("reorg detected inside logsFetcher")
		}
		f.chainHeight = head.Number.Uint64()
		toBlockheight = min(f.chainHeight, toBlockheight)
	}
	if err := f.fetchSequencerBatchLogs(ctx, parentChainBlockNumber, toBlockheight); err != nil {
		return err
	}
	if err := f.fetchDelayedMessageLogs(ctx, parentChainBlockNumber, toBlockheight, preState.DelayedMessagePostingTargetAddress); err != nil {
		return err
	}
	f.blockHeight = toBlockheight
	return nil
}

func (f *logsFetcher) fetchSequencerBatchLogs(ctx context.Context, from, to uint64) error {
	sequencerBatchDataABI := melextraction.SeqInboxABI.Events["SequencerBatchData"].ID
	query := ethereum.FilterQuery{
		FromBlock: new(big.Int).SetUint64(from),
		ToBlock:   new(big.Int).SetUint64(to),
		Topics:    [][]common.Hash{{melextraction.BatchDeliveredID, sequencerBatchDataABI}},
	}
	logs, err := f.parentChainReader.FilterLogs(ctx, query)
	if err != nil {
		return err
	}
	for _, log := range logs {
		f.logsByBlockHash[log.BlockHash] = append(f.logsByBlockHash[log.BlockHash], &log)
		if _, ok := f.logsByTxIndex[log.BlockHash]; !ok {
			f.logsByTxIndex[log.BlockHash] = make(map[uint][]*types.Log)
		}
		f.logsByTxIndex[log.BlockHash][log.TxIndex] = append(f.logsByTxIndex[log.BlockHash][log.TxIndex], &log)
	}
	return nil
}

func (f *logsFetcher) fetchDelayedMessageLogs(ctx context.Context, from, to uint64, delayedMessagePostingTargetAddress common.Address) error {
	conditionalFetch := func(addresses []common.Address, topics [][]common.Hash) error {
		query := ethereum.FilterQuery{
			FromBlock: new(big.Int).SetUint64(from),
			ToBlock:   new(big.Int).SetUint64(to),
			Addresses: addresses,
			Topics:    topics,
		}
		logs, err := f.parentChainReader.FilterLogs(ctx, query)
		if err != nil {
			return err
		}
		for _, log := range logs {
			f.logsByBlockHash[log.BlockHash] = append(f.logsByBlockHash[log.BlockHash], &log)
		}
		return nil
	}
	messageDeliveredID := melextraction.IBridgeABI.Events["MessageDelivered"].ID
	if err := conditionalFetch([]common.Address{delayedMessagePostingTargetAddress}, [][]common.Hash{{messageDeliveredID}}); err != nil {
		return err
	}
	return conditionalFetch(nil, [][]common.Hash{{melextraction.InboxMessageDeliveredID, melextraction.InboxMessageFromOriginID}})
}

func (f *logsFetcher) reset() {
	f.blockHeight = 0
	f.logsByTxIndex = make(map[common.Hash]map[uint][]*types.Log)
	f.logsByBlockHash = make(map[common.Hash][]*types.Log)
}
