package melrunner

import (
	"context"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbnode/mel"
	melextraction "github.com/offchainlabs/nitro/arbnode/mel/extraction"
)

type logsAndHeadersFetcher struct {
	parentChainReader ParentChainReader
	fromBlock         uint64
	toBlock           uint64
	blocksToFetch     uint64
	chainHeight       uint64
	headers           []*types.Header
	logsByTxIndex     map[common.Hash]map[uint][]*types.Log
	logsByBlockHash   map[common.Hash][]*types.Log
}

func newLogsAndHeadersFetcher(parentChainReader ParentChainReader, blocksToFetch uint64) *logsAndHeadersFetcher {
	return &logsAndHeadersFetcher{
		parentChainReader: parentChainReader,
		blocksToFetch:     blocksToFetch,
		logsByTxIndex:     make(map[common.Hash]map[uint][]*types.Log),
		logsByBlockHash:   make(map[common.Hash][]*types.Log),
	}
}

func (f *logsAndHeadersFetcher) LogsForBlockHash(_ context.Context, parentChainBlockHash common.Hash) ([]*types.Log, error) {
	return f.logsByBlockHash[parentChainBlockHash], nil
}

func (f *logsAndHeadersFetcher) LogsForTxIndex(_ context.Context, parentChainBlockHash common.Hash, txIndex uint) ([]*types.Log, error) {
	if txIndexToLogs, ok := f.logsByTxIndex[parentChainBlockHash]; ok {
		if logs, ok := txIndexToLogs[txIndex]; ok {
			return logs, nil
		}
	}
	return nil, fmt.Errorf("logs for tx in block: %v with index: %d not found", parentChainBlockHash, txIndex)
}

func (f *logsAndHeadersFetcher) fetch(ctx context.Context, preState *mel.State) error {
	parentChainBlockNumber := preState.ParentChainBlockNumber + 1
	if parentChainBlockNumber <= f.toBlock {
		return nil
	}
	f.reset() // prune old logs
	toBlock := parentChainBlockNumber + f.blocksToFetch
	if toBlock > f.chainHeight {
		head, err := f.parentChainReader.HeaderByNumber(ctx, nil)
		if err != nil {
			return err
		}
		if head.Number.Uint64() < parentChainBlockNumber {
			return fmt.Errorf("reorg detected inside logsAndHeadersFetcher")
		}
		f.chainHeight = head.Number.Uint64()
		toBlock = min(f.chainHeight, toBlock)
	}
	wg := sync.WaitGroup{}
	wg.Add(2)
	var fetchHeadersErr, fetchLogsErr error
	go func() {
		fetchHeadersErr = f.fetchHeaders(ctx, parentChainBlockNumber, toBlock)
		wg.Done()
	}()
	go func() {
		fetchLogsErr = f.fetchSequencerBatchLogs(ctx, parentChainBlockNumber, toBlock)
		if fetchLogsErr == nil {
			fetchLogsErr = f.fetchDelayedMessageLogs(ctx, parentChainBlockNumber, toBlock, preState.DelayedMessagePostingTargetAddress)
		}
		wg.Done()
	}()
	wg.Wait()
	if fetchHeadersErr != nil {
		return fetchHeadersErr
	}
	if fetchLogsErr != nil {
		return fetchLogsErr
	}
	f.fromBlock = parentChainBlockNumber
	f.toBlock = toBlock
	return nil
}

func (f *logsAndHeadersFetcher) fetchHeaders(ctx context.Context, from, to uint64) error {
	client := f.parentChainReader.Client() // if parentChainReader doesn't support BatchCallContext then ignore fetching headers
	if client == nil {
		return nil
	}
	headers := make([]*types.Header, to-from+1)
	var requests []rpc.BatchElem
	for i := from; i <= to; i++ {
		requests = append(requests, rpc.BatchElem{
			Method: "eth_getBlockByNumber",
			Args:   []any{hexutil.EncodeUint64(i), false},
			Result: &headers[i-from],
		})
	}
	if err := client.BatchCallContext(ctx, requests); err != nil {
		return err
	}
	f.headers = headers
	return nil
}

func (f *logsAndHeadersFetcher) fetchSequencerBatchLogs(ctx context.Context, from, to uint64) error {
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

func (f *logsAndHeadersFetcher) fetchDelayedMessageLogs(ctx context.Context, from, to uint64, delayedMessagePostingTargetAddress common.Address) error {
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

func (f *logsAndHeadersFetcher) getHeaderByNumber(ctx context.Context, number uint64) (*types.Header, error) {
	if len(f.headers) == 0 || number < f.fromBlock || number > f.toBlock { // uninitialized or out of range queries should directly be forwarded to parentChainReader
		return f.parentChainReader.HeaderByNumber(ctx, new(big.Int).SetUint64(number))
	}
	// #nosec G115
	if len(f.headers) != int(f.toBlock-f.fromBlock+1) {
		return nil, fmt.Errorf("number of cached headers doesn't correlate with fromBlock and toBlock. len: %d, fromBlock: %d, toBlock: %d", len(f.headers), f.fromBlock, f.toBlock)
	}
	pos := number - f.fromBlock
	if header := f.headers[pos]; header != nil {
		return header, nil
	}
	header, err := f.parentChainReader.HeaderByNumber(ctx, new(big.Int).SetUint64(number))
	if err != nil {
		return nil, err
	}
	f.headers[pos] = header
	return header, nil
}

func (f *logsAndHeadersFetcher) reset() {
	f.fromBlock = 0
	f.toBlock = 0
	f.logsByTxIndex = make(map[common.Hash]map[uint][]*types.Log)
	f.logsByBlockHash = make(map[common.Hash][]*types.Log)
	f.headers = []*types.Header{}
}
