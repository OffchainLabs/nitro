// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package txs

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbnode/dataposter/fees"
	"github.com/offchainlabs/nitro/arbnode/dataposter/state"
	"github.com/offchainlabs/nitro/arbnode/dataposter/storage"
	"github.com/offchainlabs/nitro/util/arbmath"
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
