// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package statetransfer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	concurrently "github.com/tejzpr/ordered-concurrently/v3"
)

type TransactionResults struct {
	Transactions []types.ArbitrumLegacyTransactionResult `json:"transactions" gencodec:"required"`
}

type classicReceiptExtra struct {
	ReturnCode hexutil.Uint64 `json:"returnCode"`
}

func ReadBlockFromClassic(ctx context.Context, rpcClient *rpc.Client, blockNumber *big.Int) (*StoredBlock, error) {
	var raw json.RawMessage
	err := rpcClient.CallContext(ctx, &raw, "eth_getBlockByNumber", hexutil.EncodeBig(blockNumber), true)
	if err != nil {
		return nil, err
	}
	var blockHeader types.Header
	var transactionResults TransactionResults // dont calculate txhashes alone
	if err := json.Unmarshal(raw, &blockHeader); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &transactionResults); err != nil {
		return nil, err
	}
	var receipts types.Receipts
	var txs []types.ArbitrumLegacyTransactionResult
	for _, tx := range transactionResults.Transactions {
		err := rpcClient.CallContext(ctx, &raw, "eth_getTransactionReceipt", tx.Hash)
		if err != nil {
			return nil, err
		}
		var extra classicReceiptExtra
		if err := json.Unmarshal(raw, &extra); err != nil {
			return nil, err
		}
		if extra.ReturnCode >= 2 {
			// possible duplicate Txhash. Skip.
			continue
		}
		var receipt *types.Receipt
		if err := json.Unmarshal(raw, &receipt); err != nil {
			return nil, err
		}
		txs = append(txs, tx)
		receipts = append(receipts, receipt)
	}
	return &StoredBlock{
		Header:       blockHeader,
		Transactions: txs,
		Reciepts:     receipts,
	}, nil
}

func scanAndCopyBlocks(reader StoredBlockReader, writer *JsonListWriter) (int64, common.Hash, error) {
	blockNum := int64(0)
	lastHash := common.Hash{}
	for reader.More() {
		block, err := reader.GetNext()
		if err != nil {
			return blockNum, lastHash, err
		}
		if block.Header.Number.Cmp(big.NewInt(blockNum)) != 0 {
			return blockNum, lastHash, fmt.Errorf("unexpected block number in input: %v", block.Header.Number)
		}
		if block.Header.ParentHash != lastHash {
			return blockNum, lastHash, fmt.Errorf("unexpected prev block hash in input: %v", block.Header.ParentHash)
		}
		err = writer.Write(block)
		if err != nil {
			return blockNum, lastHash, err
		}
		lastHash = block.Header.Hash()
		blockNum++
	}
	return blockNum, lastHash, nil
}

const parallelBlockQueries = 64

type blockQuery struct {
	rpcClient *rpc.Client
	block     uint64
}

type blockQueryResult struct {
	block *StoredBlock
	err   error
}

func (q blockQuery) Run(ctx context.Context) interface{} {
	block, err := ReadBlockFromClassic(ctx, q.rpcClient, new(big.Int).SetUint64(q.block))
	return blockQueryResult{block, err}
}

func fillBlocks(ctx context.Context, rpcClient *rpc.Client, fromBlock, toBlock uint64, prevHash common.Hash, writer *JsonListWriter) error {
	inputChan := make(chan concurrently.WorkFunction)
	output := concurrently.Process(ctx, inputChan, &concurrently.Options{PoolSize: parallelBlockQueries, OutChannelBuffer: parallelBlockQueries})
	go func() {
		for block := fromBlock; block <= toBlock; block++ {
			inputChan <- blockQuery{rpcClient, block}
		}
		close(inputChan)
	}()
	for out := range output {
		res, ok := out.Value.(blockQueryResult)
		if !ok {
			return errors.New("unexpected result type from block query")
		}
		if res.err != nil {
			return res.err
		}
		block := res.block
		completed := block.Header.Number.Uint64() - fromBlock
		totalBlocks := toBlock - fromBlock
		if completed%10 == 0 {
			fmt.Printf("\rRead block %v/%v (%.2f%%)", completed, totalBlocks, 100*float64(completed)/float64(totalBlocks))
		}
		if block.Header.ParentHash != prevHash {
			return fmt.Errorf("unexpected block hash: %v", prevHash)
		}
		err := writer.Write(&block)
		if err != nil {
			return err
		}
		prevHash = block.Header.Hash()
	}
	fmt.Printf("\rDone reading blocks!                    \n")
	return nil
}
