// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package externalsigner

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

// Converts transaction to SendTxArgs.
//
// This is needed for external signer to specify From field.
func txToSignTxArgs(addr common.Address, tx *types.Transaction) (*apitypes.SendTxArgs, error) {
	var to *common.MixedcaseAddress
	if tx.To() != nil {
		to = new(common.MixedcaseAddress)
		*to = common.NewMixedcaseAddress(*tx.To())
	}
	data := (hexutil.Bytes)(tx.Data())
	val := (*hexutil.Big)(tx.Value())
	if val == nil {
		val = (*hexutil.Big)(big.NewInt(0))
	}
	al := tx.AccessList()
	var (
		blobs       []kzg4844.Blob
		commitments []kzg4844.Commitment
		proofs      []kzg4844.Proof
		blobVersion byte
	)
	if tx.BlobTxSidecar() != nil {
		blobs = tx.BlobTxSidecar().Blobs
		commitments = tx.BlobTxSidecar().Commitments
		proofs = tx.BlobTxSidecar().Proofs
		blobVersion = tx.BlobTxSidecar().Version
	}
	return &apitypes.SendTxArgs{
		From:                 common.NewMixedcaseAddress(addr),
		To:                   to,
		Gas:                  hexutil.Uint64(tx.Gas()),
		GasPrice:             (*hexutil.Big)(tx.GasPrice()),
		MaxFeePerGas:         (*hexutil.Big)(tx.GasFeeCap()),
		MaxPriorityFeePerGas: (*hexutil.Big)(tx.GasTipCap()),
		Value:                *val,
		Nonce:                hexutil.Uint64(tx.Nonce()),
		Data:                 &data,
		AccessList:           &al,
		ChainID:              (*hexutil.Big)(tx.ChainId()),
		BlobFeeCap:           (*hexutil.Big)(tx.BlobGasFeeCap()),
		BlobHashes:           tx.BlobHashes(),
		BlobVersion:          blobVersion,
		Blobs:                blobs,
		Commitments:          commitments,
		Proofs:               proofs,
	}, nil
}
