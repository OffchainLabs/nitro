// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package txs

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbnode/dataposter/fees"
	datapostermetrics "github.com/offchainlabs/nitro/arbnode/dataposter/metrics"
	"github.com/offchainlabs/nitro/arbnode/dataposter/state"
	"github.com/offchainlabs/nitro/arbnode/dataposter/storage"
	"github.com/offchainlabs/nitro/util/arbmath"
)

func ReplaceTx(ctx context.Context, p dataPoster, s *state.LockedInternalState, prevTx *storage.QueuedTransaction, backlogWeight uint64) error {
	latestHeader, err := p.HeaderReader().LastHeader(ctx)
	if err != nil {
		return err
	}

	numBlobs := uint64(len(prevTx.FullTx.BlobHashes()))
	cfg := p.Config()

	softConfBlock := arbmath.BigSubByUint(latestHeader.Number, cfg.NonceRbfSoftConfs)
	softConfNonce, err := p.Client().NonceAt(ctx, p.Sender(), softConfBlock)
	if err != nil {
		return fmt.Errorf("failed to get latest nonce %v blocks ago (block %v): %w", cfg.NonceRbfSoftConfs, softConfBlock, err)
	}
	// #nosec G115
	datapostermetrics.LatestSoftConfirmedNonceGauge.Update(int64(softConfNonce))

	suggestedTip, err := p.Client().SuggestGasTipCap(ctx)
	if err != nil {
		return err
	}

	var currentBlobFee *big.Int
	if numBlobs > 0 {
		if latestHeader.ExcessBlobGas == nil || latestHeader.BlobGasUsed == nil {
			return fmt.Errorf(
				"latest parent chain block %v missing ExcessBlobGas or BlobGasUsed but blobs were specified in data poster transaction "+
					"(either the parent chain node is not synced or the EIP-4844 was improperly activated)",
				latestHeader.Number,
			)
		}
		currentBlobFee, err = p.ParentChain().BlobFeePerByte(ctx, latestHeader)
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
		ExtraBacklog:        p.ExtraBacklog(),
		MaxFeeCapExpression: p.MaxFeeCapExpression(),
		UsingNoOpStorage:    p.UsingNoOpStorage(),
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
		newTx.NextReplacement = time.Now().Add(time.Minute)
		return SendTx(ctx, p, s, prevTx, &newTx)
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
	newTx.FullTx, err = p.Signer(ctx, p.Sender(), unsignedTx)
	if err != nil {
		return err
	}

	return SendTx(ctx, p, s, prevTx, &newTx)
}
