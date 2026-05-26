// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package txs

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/holiman/uint256"
	"github.com/offchainlabs/nitro/arbnode/dataposter/fees"
	"github.com/offchainlabs/nitro/arbnode/dataposter/lifecycle"
	"github.com/offchainlabs/nitro/arbnode/dataposter/state"
	"github.com/offchainlabs/nitro/arbnode/dataposter/storage"
	"github.com/offchainlabs/nitro/util/blobs"
)

// Create and post a simple transaction with given parameters.
func PostSimpleTx(ctx context.Context, p dataPoster, to common.Address, calldata []byte, gasLimit uint64, value *big.Int) (*types.Transaction, error) {
	lockedState := p.InternalState().Lock()
	defer p.InternalState().Unlock()

	nonce, err := lifecycle.GetNextNonceAndMaybeMeta(ctx, p, lockedState, 1)
	if err != nil {
		return nil, err
	}

	tx := Tx{
		DataCreatedAt: time.Now(),
		Nonce:         nonce.Nonce,
		Meta:          nil,
		To:            to,
		Calldata:      calldata,
		GasLimit:      gasLimit,
		Value:         value,
		KzgBlobs:      nil,
		AccessList:    nil,
	}
	return tx.post(ctx, p, lockedState)
}

// Post the next transaction.
//
// Handles locking the internal state and performing the transaction posting.
func (tx *Tx) Post(ctx context.Context, p dataPoster) (*types.Transaction, error) {
	lockedState := p.InternalState().Lock()
	defer p.InternalState().Unlock()
	return tx.post(ctx, p, lockedState)
}

// Post a transaction, with the internal dataposter state already locked.
func (tx *Tx) post(ctx context.Context, p dataPoster, s *state.LockedInternalState) (*types.Transaction, error) {
	if p.Config().DisableNewTx {
		return nil, fmt.Errorf("posting new transaction is disabled")
	}

	var weight uint64 = 1
	if len(tx.KzgBlobs) > 0 {
		weight = uint64(len(tx.KzgBlobs))
	}
	expectedNonce, err := lifecycle.GetNextNonceAndMaybeMeta(ctx, p, s, weight)
	if err != nil {
		return nil, err
	}
	if tx.Nonce != expectedNonce.Nonce {
		return nil, fmt.Errorf("%w: data poster expected next transaction to have nonce %v but was requested to post transaction with nonce %v", storage.ErrStorageRace, expectedNonce.Nonce, tx.Nonce)
	}

	err = lifecycle.UpdateBalance(ctx, p, s)
	if err != nil {
		return nil, fmt.Errorf("failed to update data poster balance: %w", err)
	}

	latestHeader, err := p.HeaderReader().LastHeader(ctx)
	if err != nil {
		return nil, err
	}

	caps, err := fees.FeeAndTipCaps(ctx, p, s, tx.Nonce, tx.GasLimit, uint64(len(tx.KzgBlobs)), nil, tx.DataCreatedAt, 0, latestHeader)
	if err != nil {
		return nil, err
	}

	var deprecatedData types.DynamicFeeTx
	var inner types.TxData
	replacementTimes := p.Config().ReplacementTimes
	if len(tx.KzgBlobs) > 0 {
		replacementTimes = p.Config().BlobTxReplacementTimes
		value256, overflow := uint256.FromBig(tx.Value)
		if overflow {
			return nil, fmt.Errorf("blob transaction callvalue %v overflows uint256", tx.Value)
		}
		// Intentionally break out of date data poster redis clients,
		// so they don't try to replace by fee a tx they don't understand
		deprecatedData.Nonce = ^uint64(0)
		commitments, blobHashes, err := blobs.ComputeCommitmentsAndHashes(tx.KzgBlobs)
		if err != nil {
			return nil, fmt.Errorf("failed to compute KZG commitments: %w", err)
		}
		proofs, version, err := blobs.ComputeProofs(tx.KzgBlobs, commitments)
		if err != nil {
			return nil, fmt.Errorf("failed to compute KZG proofs: %w", err)
		}
		inner = &types.BlobTx{
			Nonce: tx.Nonce,
			Gas:   tx.GasLimit,
			To:    tx.To,
			Value: value256,
			Data:  tx.Calldata,
			Sidecar: &types.BlobTxSidecar{
				Version:     version,
				Blobs:       tx.KzgBlobs,
				Commitments: commitments,
				Proofs:      proofs,
			},
			BlobHashes: blobHashes,
			AccessList: tx.AccessList,
			ChainID:    p.ParentChainID256(),
		}
		// reuse the code to convert gas fee and tip caps to uint256s
		err = fees.UpdateTxDataGasCaps(inner, caps.Fee.NonBlob, caps.Tip, caps.Fee.Blob)
		if err != nil {
			return nil, err
		}
	} else {
		deprecatedData = types.DynamicFeeTx{
			Nonce:      tx.Nonce,
			GasFeeCap:  caps.Fee.NonBlob,
			GasTipCap:  caps.Tip,
			Gas:        tx.GasLimit,
			To:         &tx.To,
			Value:      tx.Value,
			Data:       tx.Calldata,
			AccessList: tx.AccessList,
			ChainID:    p.ParentChain().ChainID,
		}
		inner = &deprecatedData
	}
	fullTx, err := p.Signer(ctx, p.Sender(), types.NewTx(inner))
	if err != nil {
		return nil, fmt.Errorf("signing transaction: %w", err)
	}
	cumulativeWeight := expectedNonce.CumulativeWeight + weight
	queuedTx := storage.QueuedTransaction{
		DeprecatedData:         deprecatedData,
		FullTx:                 fullTx,
		Meta:                   tx.Meta,
		Sent:                   false,
		Created:                tx.DataCreatedAt,
		NextReplacement:        time.Now().Add(replacementTimes[0]),
		StoredCumulativeWeight: &cumulativeWeight,
	}
	return fullTx, SendTx(ctx, p, s, nil, &queuedTx)
}
