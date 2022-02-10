package statetransfer

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

type TransactionResults struct {
	Transactions []types.ArbitrumLegacyTransactionResult `json:"transactions" gencodec:"required"`
}

func ReadBlockFromClassic(ctx context.Context, rpcClient *rpc.Client, blockNumber *big.Int) (*StoredBlock, error) {
	var raw json.RawMessage
	client := ethclient.NewClient(rpcClient)
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
	for _, tx := range transactionResults.Transactions {
		reciept, err := client.TransactionReceipt(ctx, tx.Hash)
		if err != nil {
			return nil, err
		}
		receipts = append(receipts, reciept)
	}
	return &StoredBlock{
		Header:       blockHeader,
		Transactions: transactionResults.Transactions,
		Reciepts:     receipts,
	}, nil
}

func scanAndCopyBlocks(reader *JsonMultiListReader, writer *JsonMultiListWriter) (int64, common.Hash, error) {
	blockNum := int64(0)
	lastHash := common.Hash{}
	if listName, err := writer.CurListName(); err != nil || listName != "Blocks" {
		return blockNum, lastHash, fmt.Errorf("unexpected listname: %v, %w", listName, err)
	}
	for reader.More() {
		var block StoredBlock
		err := reader.GetNextElement(&block)
		if err != nil {
			return blockNum, lastHash, err
		}
		if block.Header.Number.Cmp(big.NewInt(blockNum)) != 0 {
			return blockNum, lastHash, fmt.Errorf("unexpected block number in input: %v", block.Header.Number)
		}
		if block.Header.ParentHash != lastHash {
			return blockNum, lastHash, fmt.Errorf("unexpected prev block hash in input: %v", block.Header.ParentHash)
		}
		err = writer.AddElement(block)
		if err != nil {
			return blockNum, lastHash, err
		}
		lastHash = block.Header.Hash()
		blockNum++
	}
	return blockNum, lastHash, nil
}

func fillBlocks(ctx context.Context, rpcClient *rpc.Client, fromBlock, toBlock uint64, prevHash common.Hash, writer *JsonMultiListWriter) error {
	if listName, err := writer.CurListName(); err != nil || listName != "Blocks" {
		return fmt.Errorf("unexpected listname: %v, %w", listName, err)
	}
	for blockNum := fromBlock; blockNum <= toBlock; blockNum++ {
		storedBlock, err := ReadBlockFromClassic(ctx, rpcClient, new(big.Int).SetUint64(blockNum))
		if err != nil {
			return err
		}
		if storedBlock.Header.ParentHash != prevHash {
			return fmt.Errorf("unexpected block hash: %v", prevHash)
		}
		err = writer.AddElement(&storedBlock)
		if err != nil {
			return err
		}
		prevHash = storedBlock.Header.Hash()
	}
	return nil
}
