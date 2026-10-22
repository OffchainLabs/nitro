// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package dataposter

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbnode/dataposter/fees"
	datapostermetrics "github.com/offchainlabs/nitro/arbnode/dataposter/metrics"
	"github.com/offchainlabs/nitro/arbnode/dataposter/state"
	"github.com/offchainlabs/nitro/arbnode/dataposter/storage"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/blobs"
	"github.com/offchainlabs/nitro/util/rpcclient"
)

type PostTxOpts struct {
	DataCreatedAt time.Time
	Nonce         uint64
	Meta          []byte
	To            common.Address
	Calldata      []byte
	GasLimit      uint64
	Value         *big.Int
	KzgBlobs      []kzg4844.Blob
	AccessList    types.AccessList
}

// Create and post a simple transaction with given parameters.
func (p *DataPoster) PostSimpleTransaction(ctx context.Context, to common.Address, calldata []byte, gasLimit uint64, value *big.Int) (*types.Transaction, error) {
	lockedState := p.internalState.Lock()
	defer p.internalState.Unlock()

	nonce, err := p.getNextNonceAndMaybeMeta(ctx, lockedState, 1)
	if err != nil {
		return nil, err
	}

	opts := PostTxOpts{
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
	return p.postTx(ctx, lockedState, &opts)
}

// Post the next transaction.
//
// Handles locking the internal state and performing the transaction posting.
func (p *DataPoster) PostTransaction(ctx context.Context, dataCreatedAt time.Time, nonce uint64, meta []byte, to common.Address, calldata []byte, gasLimit uint64, value *big.Int, kzgBlobs []kzg4844.Blob, accessList types.AccessList) (*types.Transaction, error) {
	lockedState := p.internalState.Lock()
	defer p.internalState.Unlock()
	opts := PostTxOpts{
		DataCreatedAt: dataCreatedAt,
		Nonce:         nonce,
		Meta:          meta,
		To:            to,
		Calldata:      calldata,
		GasLimit:      gasLimit,
		Value:         value,
		KzgBlobs:      kzgBlobs,
		AccessList:    accessList,
	}
	return p.postTx(ctx, lockedState, &opts)
}

// Post a transaction, with the internal dataposter state already locked.
func (p *DataPoster) postTx(ctx context.Context, s *state.LockedInternalState, tx *PostTxOpts) (*types.Transaction, error) {
	if p.config().DisableNewTx {
		return nil, fmt.Errorf("posting new transaction is disabled")
	}

	var weight uint64 = 1
	if len(tx.KzgBlobs) > 0 {
		weight = uint64(len(tx.KzgBlobs))
	}
	expectedNonce, err := p.getNextNonceAndMaybeMeta(ctx, s, weight)
	if err != nil {
		return nil, err
	}
	if tx.Nonce != expectedNonce.Nonce {
		return nil, fmt.Errorf("%w: data poster expected next transaction to have nonce %v but was requested to post transaction with nonce %v", storage.ErrStorageRace, expectedNonce.Nonce, tx.Nonce)
	}

	err = p.updateBalance(ctx, s)
	if err != nil {
		return nil, fmt.Errorf("failed to update data poster balance: %w", err)
	}

	latestHeader, err := p.headerReader.LastHeader(ctx)
	if err != nil {
		return nil, err
	}

	cfg := p.config()
	numBlobs := uint64(len(tx.KzgBlobs))

	softConfBlock := arbmath.BigSubByUint(latestHeader.Number, cfg.NonceRbfSoftConfs)
	softConfNonce, err := p.client.NonceAt(ctx, p.Sender(), softConfBlock)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest nonce %v blocks ago (block %v): %w", cfg.NonceRbfSoftConfs, softConfBlock, err)
	}
	// #nosec G115
	datapostermetrics.LatestSoftConfirmedNonceGauge.Update(int64(softConfNonce))

	suggestedTip, err := p.client.SuggestGasTipCap(ctx)
	if err != nil {
		return nil, err
	}
	datapostermetrics.SuggestedTipCapGauge.Update(suggestedTip.Int64())

	var currentBlobFee *big.Int
	if numBlobs > 0 {
		if latestHeader.ExcessBlobGas == nil || latestHeader.BlobGasUsed == nil {
			return nil, fmt.Errorf(
				"latest parent chain block %v missing ExcessBlobGas or BlobGasUsed but blobs were specified in data poster transaction "+
					"(either the parent chain node is not synced or the EIP-4844 was improperly activated)",
				latestHeader.Number,
			)
		}
		currentBlobFee, err = p.parentChain.BlobFeePerByte(ctx, latestHeader)
		if err != nil {
			return nil, fmt.Errorf("failed to get blob base fee: %w", err)
		}
	}

	caps, err := fees.FeeAndTipCaps(&fees.FeeCalcOpts{
		Config:              cfg,
		Balance:             s.Balance,
		SoftConfNonce:       softConfNonce,
		SuggestedTip:        suggestedTip,
		CurrentBlobFee:      currentBlobFee,
		ExtraBacklog:        p.extraBacklog(),
		MaxFeeCapExpression: p.maxFeeCapExpression,
		UsingNoOpStorage:    p.usingNoOpStorage,
		Nonce:               tx.Nonce,
		GasLimit:            tx.GasLimit,
		NumBlobs:            numBlobs,
		LastTx:              nil, // new transaction, not RBF
		DataCreatedAt:       tx.DataCreatedAt,
		DataPosterBacklog:   0,
		LatestHeader:        latestHeader,
	})
	if err != nil {
		return nil, err
	}

	var deprecatedData types.DynamicFeeTx
	var inner types.TxData
	replacementTimes := cfg.ReplacementTimes
	if len(tx.KzgBlobs) > 0 {
		replacementTimes = cfg.BlobTxReplacementTimes
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
			ChainID:    p.parentChainID256,
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
			ChainID:    p.parentChain.ChainID,
		}
		inner = &deprecatedData
	}
	fullTx, err := p.signer(ctx, p.Sender(), types.NewTx(inner))
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
	return fullTx, p.sendTx(ctx, s, nil, &queuedTx)
}

func (p *DataPoster) replaceTx(ctx context.Context, s *state.LockedInternalState, prevTx *storage.QueuedTransaction, backlogWeight uint64) error {
	latestHeader, err := p.headerReader.LastHeader(ctx)
	if err != nil {
		return err
	}

	numBlobs := uint64(len(prevTx.FullTx.BlobHashes()))
	cfg := p.config()

	softConfBlock := arbmath.BigSubByUint(latestHeader.Number, cfg.NonceRbfSoftConfs)
	softConfNonce, err := p.client.NonceAt(ctx, p.Sender(), softConfBlock)
	if err != nil {
		return fmt.Errorf("failed to get latest nonce %v blocks ago (block %v): %w", cfg.NonceRbfSoftConfs, softConfBlock, err)
	}
	// #nosec G115
	datapostermetrics.LatestSoftConfirmedNonceGauge.Update(int64(softConfNonce))

	suggestedTip, err := p.client.SuggestGasTipCap(ctx)
	if err != nil {
		return err
	}
	datapostermetrics.SuggestedTipCapGauge.Update(suggestedTip.Int64())

	var currentBlobFee *big.Int
	if numBlobs > 0 {
		if latestHeader.ExcessBlobGas == nil || latestHeader.BlobGasUsed == nil {
			return fmt.Errorf(
				"latest parent chain block %v missing ExcessBlobGas or BlobGasUsed but blobs were specified in data poster transaction "+
					"(either the parent chain node is not synced or the EIP-4844 was improperly activated)",
				latestHeader.Number,
			)
		}
		currentBlobFee, err = p.parentChain.BlobFeePerByte(ctx, latestHeader)
		if err != nil {
			return fmt.Errorf("failed to get blob base fee: %w", err)
		}
	}

	caps, err := fees.FeeAndTipCaps(&fees.FeeCalcOpts{
		Config:              cfg,
		Balance:             s.Balance,
		SoftConfNonce:       softConfNonce,
		SuggestedTip:        suggestedTip,
		CurrentBlobFee:      currentBlobFee,
		ExtraBacklog:        p.extraBacklog(),
		MaxFeeCapExpression: p.maxFeeCapExpression,
		UsingNoOpStorage:    p.usingNoOpStorage,
		Nonce:               prevTx.FullTx.Nonce(),
		GasLimit:            prevTx.FullTx.Gas(),
		NumBlobs:            numBlobs,
		LastTx:              prevTx.FullTx,
		DataCreatedAt:       prevTx.Created,
		DataPosterBacklog:   backlogWeight,
		LatestHeader:        latestHeader,
	})
	if err != nil {
		return err
	}

	minRbfIncrease := fees.MinRbfIncrease.SelectIfBlobs(numBlobs > 0)
	newTx := *prevTx
	if (prevTx.FullTx.GasFeeCap().Sign() > 0 && arbmath.BigDivToBips(caps.Fee.NonBlob, prevTx.FullTx.GasFeeCap()) < minRbfIncrease) ||
		(prevTx.FullTx.BlobGasFeeCap() != nil && prevTx.FullTx.BlobGasFeeCap().Sign() > 0 && arbmath.BigDivToBips(caps.Fee.Blob, prevTx.FullTx.BlobGasFeeCap()) < minRbfIncrease) {
		log.Debug(
			"no need to replace by fee transaction",
			"nonce", prevTx.FullTx.Nonce(),
			"lastFeeCap", prevTx.FullTx.GasFeeCap(),
			"recommendedFeeCap", caps.Fee.NonBlob,
			"lastTipCap", prevTx.FullTx.GasTipCap(),
			"recommendedTipCap", caps.Tip,
			"lastBlobFeeCap", prevTx.FullTx.BlobGasFeeCap(),
			"recommendedBlobFeeCap", caps.Fee.Blob,
		)
		datapostermetrics.DeferredTipBumpsCounter.Inc(1)
		newTx.NextReplacement = time.Now().Add(time.Minute)
		return p.sendTx(ctx, s, prevTx, &newTx)
	}

	replacementTimes := cfg.ReplacementTimes
	if numBlobs > 0 {
		replacementTimes = cfg.BlobTxReplacementTimes
	}

	elapsed := time.Since(prevTx.Created)
	for _, replacement := range replacementTimes {
		if elapsed >= replacement {
			continue
		}
		newTx.NextReplacement = prevTx.Created.Add(replacement)
		break
	}
	newTx.Sent = false
	newTx.DeprecatedData.GasFeeCap = caps.Fee.NonBlob
	newTx.DeprecatedData.GasTipCap = caps.Tip
	unsignedTx, err := fees.UpdateGasCaps(newTx.FullTx, caps.Fee.NonBlob, caps.Tip, caps.Fee.Blob)
	if err != nil {
		return err
	}
	newTx.FullTx, err = p.signer(ctx, p.Sender(), unsignedTx)
	if err != nil {
		return err
	}

	datapostermetrics.TipBumpsCounter.Inc(1)
	return p.sendTx(ctx, s, prevTx, &newTx)
}

func (p *DataPoster) sendTx(ctx context.Context, s *state.LockedInternalState, prevTx *storage.QueuedTransaction, newTx *storage.QueuedTransaction) error {
	latestHeader, err := p.client.HeaderByNumber(ctx, nil)
	if err != nil {
		return err
	}
	var currentBlobFee *big.Int
	if p.config().Post4844Blobs && latestHeader.ExcessBlobGas != nil && latestHeader.BlobGasUsed != nil {
		currentBlobFee, err = p.parentChain.BlobFeePerByte(ctx, latestHeader)
		if err != nil {
			return err
		}
	}

	if arbmath.BigLessThan(newTx.FullTx.GasFeeCap(), latestHeader.BaseFee) {
		log.Info(
			"submitting transaction with GasFeeCap less than latest basefee",
			"txBasefeeCap", newTx.FullTx.GasFeeCap(),
			"latestBasefee", latestHeader.BaseFee,
			"elapsed", time.Since(newTx.Created),
		)
	}

	if newTx.FullTx.BlobGasFeeCap() != nil && currentBlobFee != nil && arbmath.BigLessThan(newTx.FullTx.BlobGasFeeCap(), currentBlobFee) {
		log.Info(
			"submitting transaction with BlobGasFeeCap less than latest blobfee",
			"txBlobGasFeeCap", newTx.FullTx.BlobGasFeeCap(),
			"latestBlobFee", currentBlobFee,
			"elapsed", time.Since(newTx.Created),
		)
	}

	if err := saveTx(ctx, s, prevTx, newTx); err != nil {
		return err
	}

	// The following check is to avoid sending transactions of a different type (e.g. DynamicFeeTxType vs BlobTxType)
	// to the previous tx if the previous tx is not yet included in a reorg resistant block, in order to avoid issues
	// where eventual consistency of parent chain mempools causes a tx with higher nonce blocking a tx of a
	// different type with a lower nonce.
	// If we decide not to send this tx yet, just leave it queued and with Sent set to false.
	// The resending/repricing loop in DataPoster.Start will keep trying.
	previouslySent := newTx.Sent || (prevTx != nil && prevTx.Sent) // if we've previously sent this nonce
	if !previouslySent && newTx.FullTx.Nonce() > 0 {
		precedingTx, err := s.Queue.Get(ctx, arbmath.SaturatingUSub(newTx.FullTx.Nonce(), 1))
		if err != nil {
			return fmt.Errorf("couldn't get preceding tx in DataPoster to check if should send tx with nonce %d: %w", newTx.FullTx.Nonce(), err)
		}
		if precedingTx != nil { // precedingTx == nil -> the actual preceding tx was already confirmed
			var latestBlockNumber, prevBlockNumber, reorgResistantTxCount uint64
			if precedingTx.FullTx.Type() != newTx.FullTx.Type() || !precedingTx.Sent {
				latestBlockNumber, err = p.client.BlockNumber(ctx)
				if err != nil {
					return fmt.Errorf("couldn't get block number in DataPoster to check if should send tx with nonce %d: %w", newTx.FullTx.Nonce(), err)
				}
				prevBlockNumber = arbmath.SaturatingUSub(latestBlockNumber, 1)
				reorgResistantTxCount, err = p.client.NonceAt(ctx, p.Sender(), new(big.Int).SetUint64(prevBlockNumber))
				if err != nil {
					return fmt.Errorf("couldn't determine reorg resistant nonce in DataPoster to check if should send tx with nonce %d: %w", newTx.FullTx.Nonce(), err)
				}

				if newTx.FullTx.Nonce() > reorgResistantTxCount {
					log.Info("DataPoster is avoiding creating a mempool nonce gap (the tx remains queued and will be retried)", "nonce", newTx.FullTx.Nonce(), "prevType", precedingTx.FullTx.Type(), "type", newTx.FullTx.Type(), "prevSent", precedingTx.Sent, "latestBlockNumber", latestBlockNumber, "prevBlockNumber", prevBlockNumber, "reorgResistantTxCount", reorgResistantTxCount)
					return nil
				}
			}
			log.Debug("DataPoster will send previously unsent batch tx", "nonce", newTx.FullTx.Nonce(), "prevType", precedingTx.FullTx.Type(), "type", newTx.FullTx.Type(), "prevSent", precedingTx.Sent, "latestBlockNumber", latestBlockNumber, "prevBlockNumber", prevBlockNumber, "reorgResistantTxCount", reorgResistantTxCount)
		}
	}

	if err := p.client.SendTransaction(ctx, newTx.FullTx); err != nil {
		isAlreadyKnown := rpcclient.IsAlreadyKnownError(err)
		isAlreadyKnown = isAlreadyKnown || strings.Contains(err.Error(), "nonce too low")
		// If we previously sent this nonce and the same tx, some L1 clients may return ReplacementNotAllowed instead of
		// an already known error (might be due to their cache size constraints) so we dont return an error in such a case
		_, _, errTxByHash := p.client.TransactionByHash(ctx, newTx.FullTx.Hash())
		isAlreadyKnown = isAlreadyKnown || (strings.Contains(err.Error(), "ReplacementNotAllowed") && errTxByHash == nil)
		if !isAlreadyKnown {
			log.Warn("DataPoster failed to send transaction", "err", err, "nonce", newTx.FullTx.Nonce(), "feeCap", newTx.FullTx.GasFeeCap(), "tipCap", newTx.FullTx.GasTipCap(), "blobFeeCap", newTx.FullTx.BlobGasFeeCap(), "gas", newTx.FullTx.Gas())
			return err
		}
		log.Info("DataPoster transaction already known", "err", err, "nonce", newTx.FullTx.Nonce(), "hash", newTx.FullTx.Hash())
	} else {
		log.Info("DataPoster sent transaction", "nonce", newTx.FullTx.Nonce(), "hash", newTx.FullTx.Hash(), "feeCap", newTx.FullTx.GasFeeCap(), "tipCap", newTx.FullTx.GasTipCap(), "blobFeeCap", newTx.FullTx.BlobGasFeeCap(), "gas", newTx.FullTx.Gas())
	}
	datapostermetrics.LastTipCapGauge.Update(newTx.FullTx.GasTipCap().Int64())
	datapostermetrics.LastFeeCapGauge.Update(newTx.FullTx.GasFeeCap().Int64())
	if blobFeeCap := newTx.FullTx.BlobGasFeeCap(); blobFeeCap != nil {
		datapostermetrics.LastBlobFeeCapGauge.Update(blobFeeCap.Int64())
	}
	newerTx := *newTx
	newerTx.Sent = true
	return saveTx(ctx, s, newTx, &newerTx)
}

func saveTx(ctx context.Context, s *state.LockedInternalState, prevTx, newTx *storage.QueuedTransaction) error {
	if prevTx != nil {
		if prevTx.FullTx.Nonce() != newTx.FullTx.Nonce() {
			return fmt.Errorf("prevTx nonce %v doesn't match newTx nonce %v", prevTx.FullTx.Nonce(), newTx.FullTx.Nonce())
		}

		// Check if prevTx is the same as newTx and we don't need to do anything
		oldEnc, err := rlp.EncodeToBytes(prevTx)
		if err != nil {
			return fmt.Errorf("failed to encode prevTx: %w", err)
		}
		newEnc, err := rlp.EncodeToBytes(newTx)
		if err != nil {
			return fmt.Errorf("failed to encode newTx: %w", err)
		}
		if bytes.Equal(oldEnc, newEnc) {
			// No need to save newTx as it's the same as prevTx
			return nil
		}
	}
	if err := s.Queue.Put(ctx, newTx.FullTx.Nonce(), prevTx, newTx); err != nil {
		return fmt.Errorf("putting new tx in the queue: %w", err)
	}
	return nil
}
