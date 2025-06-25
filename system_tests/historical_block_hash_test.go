package arbtest

import (
	"bytes"
	"context"
	"encoding/binary"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

func TestHistoricalBlockHash(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	cleanup := builder.Build(t)
	defer cleanup()

	for {
		builder.L2.TransferBalance(t, "Faucet", "Faucet", common.Big1, builder.L2Info)
		number, err := builder.L2.Client.BlockNumber(ctx)
		Require(t, err)
		if number > 300 {
			break
		}
	}

	block, err := builder.L2.Client.BlockByNumber(ctx, nil)
	Require(t, err)

	for i := uint64(0); i < block.Number().Uint64(); i++ {
		var key common.Hash
		binary.BigEndian.PutUint64(key[24:], i)
		expectedBlock, err := builder.L2.Client.BlockByNumber(ctx, new(big.Int).SetUint64(i))
		Require(t, err)
		blockHash := sendContractCall(t, ctx, params.HistoryStorageAddress, builder.L2.Client, key.Bytes())
		if !bytes.Equal(blockHash, expectedBlock.Hash().Bytes()) {
			t.Fatalf("Expected block hash %s, got %s", expectedBlock.Hash(), blockHash)
		}
	}

}
