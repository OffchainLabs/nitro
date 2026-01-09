package melrecording

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"

	"github.com/offchainlabs/nitro/daprovider"
)

func TestLogsForTxIndex(t *testing.T) {
	ctx := context.Background()
	blockReader := &mockBlockReader{
		blocks:          make(map[common.Hash]*types.Block),
		receiptByTxHash: map[common.Hash]*types.Receipt{},
	}
	toAddr := common.HexToAddress("0x0000000000000000000000000000000000DeaDBeef")
	blockHeader := &types.Header{}
	receipts := []*types.Receipt{}
	txs := make([]*types.Transaction, 0)
	for i := uint64(1); i < 10; i++ {
		txData := &types.DynamicFeeTx{
			To:        &toAddr,
			Nonce:     i,
			GasFeeCap: big.NewInt(1),
			GasTipCap: big.NewInt(1),
			Gas:       1,
			Value:     big.NewInt(0),
			Data:      nil,
		}
		tx := types.NewTx(txData)
		txs = append(txs, tx)
		receipt := &types.Receipt{
			TxHash:           tx.Hash(),
			TransactionIndex: uint(i - 1),
			Type:             types.DynamicFeeTxType,
			Logs: []*types.Log{
				{
					// Consensus fields:
					Address: common.HexToAddress("sample"),
					Topics:  []common.Hash{common.HexToHash("topic1"), common.HexToHash("topic2")},
					Data:    common.Hex2Bytes(fmt.Sprintf("data:%d", i)),

					// Derived Fields:
					TxIndex: uint(i - 1),
				},
			},
		}
		receipts = append(receipts, receipt)
		blockReader.receiptByTxHash[tx.Hash()] = receipt
	}
	blockBody := &types.Body{
		Transactions: txs,
	}
	block := types.NewBlock(
		blockHeader,
		blockBody,
		receipts,
		trie.NewStackTrie(nil),
	)
	blockReader.blocks[block.Hash()] = block
	preimages := make(daprovider.PreimagesMap)
	recorder, err := NewReceiptRecorder(blockReader, block.Hash(), preimages)
	require.NoError(t, err)
	require.NoError(t, recorder.Initialize(ctx))

	txIndex := uint(3)
	logs, err := recorder.LogsForTxIndex(ctx, block.Hash(), txIndex)
	require.NoError(t, err)
	have, err := logs[0].MarshalJSON()
	require.NoError(t, err)
	want, err := receipts[txIndex].Logs[0].MarshalJSON()
	require.NoError(t, err)
	require.Equal(t, want, have)
}
