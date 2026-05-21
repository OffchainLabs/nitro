// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package mainloop

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbnode/dataposter/lifecycle"
	datapostermetrics "github.com/offchainlabs/nitro/arbnode/dataposter/metrics"
	"github.com/offchainlabs/nitro/util/arbmath"
)

const minWait = time.Second * 10

func Tick(ctx context.Context, p dataPoster) time.Duration {
	lockedState := p.InternalState().Lock()
	defer p.InternalState().Unlock()

	err := lifecycle.UpdateBalance(ctx, p, lockedState)
	if err != nil {
		log.Warn("failed to update tx poster balance", "err", err)
		return minWait
	}

	err = lifecycle.UpdateNonce(ctx, p, lockedState)
	if err != nil {
		// This is non-fatal because it's only needed for clearing out old queue items.
		log.Warn("failed to update tx poster nonce", "err", err)
	}

	now := time.Now()
	nextCheck := now.Add(p.Config().ReplacementTimes[0])
	if len(p.Config().BlobTxReplacementTimes) > 0 {
		nextCheck = now.Add(arbmath.MinInt(p.Config().ReplacementTimes[0], p.Config().BlobTxReplacementTimes[0]))
	}

	maxTxsToRbf := p.Config().MaxMempoolTransactions
	if maxTxsToRbf == 0 {
		maxTxsToRbf = 512
	}
	unconfirmedNonce, err := p.Client().NonceAt(ctx, p.Sender(), nil)
	if err != nil {
		log.Warn("Failed to get latest nonce", "err", err)
		return minWait
	}
	// #nosec G115
	datapostermetrics.LatestUnconfirmedNonceGauge.Update(int64(unconfirmedNonce))

	// We use unconfirmedNonce here to replace-by-fee transactions that aren't in a block,
	// excluding those that are in an unconfirmed block. If a reorg occurs, we'll continue
	// replacing them by fee.
	queueContents, err := lockedState.Queue.FetchContents(ctx, unconfirmedNonce, maxTxsToRbf)
	if err != nil {
		log.Error("Failed to fetch tx queue contents", "err", err)
		return minWait
	}

	latestQueued, err := lockedState.Queue.FetchLast(ctx)
	if err != nil {
		log.Error("Failed to fetch last queued tx", "err", err)
		return minWait
	}

	var latestCumulativeWeight, latestNonce uint64
	if latestQueued != nil {
		latestCumulativeWeight = latestQueued.CumulativeWeight()
		latestNonce = latestQueued.FullTx.Nonce()

		confirmedNonce := unconfirmedNonce - 1
		confirmedMeta, err := lockedState.Queue.Get(ctx, confirmedNonce)
		if err == nil && confirmedMeta != nil {
			// #nosec G115
			datapostermetrics.TotalQueueWeightGauge.Update(int64(arbmath.SaturatingUSub(latestCumulativeWeight, confirmedMeta.CumulativeWeight())))
			// #nosec G115
			datapostermetrics.TotalQueueLengthGauge.Update(int64(arbmath.SaturatingUSub(latestNonce, confirmedNonce)))
		} else {
			log.Error("Failed to fetch latest confirmed tx from queue", "confirmedNonce", confirmedNonce, "err", err, "confirmedMeta", confirmedMeta)
		}
	}

	for _, tx := range queueContents {
		if now.After(tx.NextReplacement) {
			weightBacklog := arbmath.SaturatingUSub(latestCumulativeWeight, tx.CumulativeWeight())
			nonceBacklog := arbmath.SaturatingUSub(latestNonce, tx.FullTx.Nonce())
			err := ReplaceTx(ctx, p, lockedState, tx, arbmath.MaxInt(nonceBacklog, weightBacklog))
			lifecycle.MaybeLogError(err, lockedState, tx, "failed to replace-by-fee transaction")
		} else {
			err := SendTx(ctx, p, lockedState, tx, tx)
			lifecycle.MaybeLogError(err, lockedState, tx, "failed to re-send transaction")
		}
		nonce := tx.FullTx.Nonce()
		tx, err = lockedState.Queue.Get(ctx, nonce)
		if err != nil {
			log.Error("Failed to fetch tx from queue to check updated status", "nonce", nonce, "err", err)
			return minWait
		}
		if tx == nil {
			log.Error("Failed to fetch tx from queue to check updated status, got tx == nil", "nonce", nonce)
			return minWait
		}
		if nextCheck.After(tx.NextReplacement) {
			nextCheck = tx.NextReplacement
		}
		if !tx.Sent {
			// We can't progress any further if we failed to send this tx
			// Retry sending this tx soon
			return minWait
		}
	}

	wait := time.Until(nextCheck)
	if wait < minWait {
		wait = minWait
	}
	return wait
}
