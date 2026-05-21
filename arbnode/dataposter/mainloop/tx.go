// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package mainloop

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbnode/dataposter/fees"
	"github.com/offchainlabs/nitro/arbnode/dataposter/state"
	"github.com/offchainlabs/nitro/arbnode/dataposter/storage"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/rpcclient"
)

func ReplaceTx(ctx context.Context, p dataPoster, s *state.LockedInternalState, prevTx *storage.QueuedTransaction, backlogWeight uint64) error {
	latestHeader, err := p.HeaderReader().LastHeader(ctx)
	if err != nil {
		return err
	}

	caps, err := fees.FeeAndTipCaps(ctx, p, s, prevTx.FullTx.Nonce(), prevTx.FullTx.Gas(), uint64(len(prevTx.FullTx.BlobHashes())), prevTx.FullTx, prevTx.Created, backlogWeight, latestHeader)
	if err != nil {
		return err
	}

	minRbfIncrease := fees.MinRbfIncrease.SelectIfBlobs(len(prevTx.FullTx.BlobHashes()) > 0)
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
		newTx.NextReplacement = time.Now().Add(time.Minute)
		return SendTx(ctx, p, s, prevTx, &newTx)
	}

	replacementTimes := p.Config().ReplacementTimes
	if len(prevTx.FullTx.BlobHashes()) > 0 {
		replacementTimes = p.Config().BlobTxReplacementTimes
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
	newTx.FullTx, err = p.Signer(ctx, p.Sender(), unsignedTx)
	if err != nil {
		return err
	}

	return SendTx(ctx, p, s, prevTx, &newTx)
}

func SendTx(ctx context.Context, p dataPoster, s *state.LockedInternalState, prevTx *storage.QueuedTransaction, newTx *storage.QueuedTransaction) error {
	latestHeader, err := p.Client().HeaderByNumber(ctx, nil)
	if err != nil {
		return err
	}
	var currentBlobFee *big.Int
	if p.Config().Post4844Blobs && latestHeader.ExcessBlobGas != nil && latestHeader.BlobGasUsed != nil {
		currentBlobFee, err = p.ParentChain().BlobFeePerByte(ctx, latestHeader)
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

	if err := SaveTx(ctx, p, s, prevTx, newTx); err != nil {
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
				latestBlockNumber, err = p.Client().BlockNumber(ctx)
				if err != nil {
					return fmt.Errorf("couldn't get block number in DataPoster to check if should send tx with nonce %d: %w", newTx.FullTx.Nonce(), err)
				}
				prevBlockNumber = arbmath.SaturatingUSub(latestBlockNumber, 1)
				reorgResistantTxCount, err = p.Client().NonceAt(ctx, p.Sender(), new(big.Int).SetUint64(prevBlockNumber))
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

	if err := p.Client().SendTransaction(ctx, newTx.FullTx); err != nil {
		isAlreadyKnown := rpcclient.IsAlreadyKnownError(err)
		isAlreadyKnown = isAlreadyKnown || strings.Contains(err.Error(), "nonce too low")
		// If we previously sent this nonce and the same tx, some L1 clients may return ReplacementNotAllowed instead of
		// an already known error (might be due to their cache size constraints) so we dont return an error in such a case
		_, _, errTxByHash := p.Client().TransactionByHash(ctx, newTx.FullTx.Hash())
		isAlreadyKnown = isAlreadyKnown || (strings.Contains(err.Error(), "ReplacementNotAllowed") && errTxByHash == nil)
		if !isAlreadyKnown {
			log.Warn("DataPoster failed to send transaction", "err", err, "nonce", newTx.FullTx.Nonce(), "feeCap", newTx.FullTx.GasFeeCap(), "tipCap", newTx.FullTx.GasTipCap(), "blobFeeCap", newTx.FullTx.BlobGasFeeCap(), "gas", newTx.FullTx.Gas())
			return err
		}
		log.Info("DataPoster transaction already known", "err", err, "nonce", newTx.FullTx.Nonce(), "hash", newTx.FullTx.Hash())
	} else {
		log.Info("DataPoster sent transaction", "nonce", newTx.FullTx.Nonce(), "hash", newTx.FullTx.Hash(), "feeCap", newTx.FullTx.GasFeeCap(), "tipCap", newTx.FullTx.GasTipCap(), "blobFeeCap", newTx.FullTx.BlobGasFeeCap(), "gas", newTx.FullTx.Gas())
	}
	newerTx := *newTx
	newerTx.Sent = true
	return SaveTx(ctx, p, s, newTx, &newerTx)
}

func SaveTx(ctx context.Context, p dataPoster, s *state.LockedInternalState, prevTx, newTx *storage.QueuedTransaction) error {
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
