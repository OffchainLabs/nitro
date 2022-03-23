// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package l1pricing

import (
	"math"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

func TestTxFixedCost(t *testing.T) {
	maxChainId := new(big.Int).SetUint64(math.MaxUint64)
	maxValue := big.NewInt(1_000_000)
	maxValue.Mul(maxValue, big.NewInt(params.Ether))
	var address common.Address
	for i := range address {
		address[i] = 0xFF
	}
	maxSigVal := big.NewInt(2)
	maxSigVal.Exp(maxSigVal, big.NewInt(256), nil)
	maxSigVal.Sub(maxSigVal, common.Big1)
	maxGasPrice := big.NewInt(1000 * params.GWei)
	largeTx := types.NewTx(&types.DynamicFeeTx{
		ChainID:    maxChainId,
		Nonce:      1 << 32,
		GasTipCap:  maxGasPrice,
		GasFeeCap:  maxGasPrice,
		Gas:        100_000_000,
		To:         &address,
		Value:      maxValue,
		Data:       []byte{},
		AccessList: []types.AccessTuple{},
		V:          common.Big1,
		R:          maxSigVal,
		S:          maxSigVal,
	})
	largeTxEncoded, err := largeTx.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if len(largeTxEncoded) > TxFixedCost {
		t.Fatal("large tx is", len(largeTxEncoded), "bytes but tx fixed cost is", TxFixedCost)
	}
}
