// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package fees

import (
	"fmt"
	"math/big"

	"github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/core/types"
)

func UpdateTxDataGasCaps(data types.TxData, newFeeCap, newTipCap, newBlobFeeCap *big.Int) error {
	switch data := data.(type) {
	case *types.DynamicFeeTx:
		data.GasFeeCap = newFeeCap
		data.GasTipCap = newTipCap
		return nil
	case *types.BlobTx:
		var overflow bool
		data.GasFeeCap, overflow = uint256.FromBig(newFeeCap)
		if overflow {
			return fmt.Errorf("blob tx fee cap %v exceeds uint256", newFeeCap)
		}
		data.GasTipCap, overflow = uint256.FromBig(newTipCap)
		if overflow {
			return fmt.Errorf("blob tx tip cap %v exceeds uint256", newTipCap)
		}
		data.BlobFeeCap, overflow = uint256.FromBig(newBlobFeeCap)
		if overflow {
			return fmt.Errorf("blob tx blob fee cap %v exceeds uint256", newBlobFeeCap)
		}
		return nil
	default:
		return fmt.Errorf("unexpected transaction data type %T", data)
	}
}

func UpdateGasCaps(tx *types.Transaction, newFeeCap, newTipCap, newBlobFeeCap *big.Int) (*types.Transaction, error) {
	data := tx.GetInner()
	err := UpdateTxDataGasCaps(data, newFeeCap, newTipCap, newBlobFeeCap)
	if err != nil {
		return nil, err
	}
	return types.NewTx(data), nil
}
