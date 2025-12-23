package melrecording

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/stretchr/testify/require"
)

type mockBlockReader struct {
	blocks map[common.Hash]*types.Block
}

func (mbr *mockBlockReader) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	block, exists := mbr.blocks[hash]
	if !exists {
		return nil, nil
	}
	return block, nil
}

func TestTransactionByLog(t *testing.T) {
	ctx := context.Background()
	toAddr := common.HexToAddress("0x0000000000000000000000000000000000DeaDBeef")
	blockHeader := &types.Header{}
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
	}
	blockBody := &types.Body{
		Transactions: txs,
	}
	receipts := []*types.Receipt{}
	block := types.NewBlock(
		blockHeader,
		blockBody,
		receipts,
		trie.NewStackTrie(nil),
	)
	blockReader := &mockBlockReader{
		blocks: map[common.Hash]*types.Block{
			block.Hash(): block,
		},
	}
	preimages := make(daprovider.PreimagesMap)
	recorder := NewTransactionRecorder(blockReader, block.Hash(), preimages)
	require.NoError(t, recorder.Initialize(ctx))

	log := &types.Log{
		TxIndex: 5,
	}
	tx, err := recorder.TransactionByLog(ctx, log)
	require.NoError(t, err)
	have, err := tx.MarshalJSON()
	require.NoError(t, err)
	want, err := block.Transactions()[5].MarshalJSON()
	require.NoError(t, err)
	require.Equal(t, want, have)
}
