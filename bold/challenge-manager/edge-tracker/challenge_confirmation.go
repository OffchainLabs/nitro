// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

package edgetracker

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ccoveille/go-safecast"
	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"

	"github.com/offchainlabs/nitro/bold/chain-abstraction"
	"github.com/offchainlabs/nitro/bold/runtime"
)

var onchainTimerDifferAfterConfirmationJobCounter = metrics.NewRegisteredCounter("arb/validator/tracker/onchain_timer_differed_after_confirmation_job", nil)

// Defines a struct which can handle confirming of an entire challenge tree
// in the BOLD protocol. It does so by updating the inherited timers of royal edges
// onchain until the root of the tree has a timer >= a challenge period. At that point,
// it ensures to confirm that edge. If this is not the case, it will return an error
// and write data to disk to help with debugging the issue.
type challengeConfirmer struct {
	reader                      HonestChallengeTreeReader
	writer                      ChainWriter
	backend                     protocol.ChainBackend
	validatorName               string
	averageTimeForBlockCreation time.Duration
	chain                       protocol.Protocol
}

// Defines a chain writer interface that is
// used to update the cached inherited timers of edges
// onchain.
type ChainWriter interface {
	MultiUpdateInheritedTimers(
		ctx context.Context,
		challengeBranch []protocol.ReadOnlyEdge,
		desiredNewTimerForLastEdge uint64,
	) (*types.Transaction, error)
}

func newChallengeConfirmer(
	challengeReader HonestChallengeTreeReader,
	chainWriter ChainWriter,
	backend protocol.ChainBackend,
	validatorName string,
	chain protocol.Protocol,
) *challengeConfirmer {
	return &challengeConfirmer{
		reader:        challengeReader,
		writer:        chainWriter,
		validatorName: validatorName,
		backend:       backend,
		chain:         chain,
	}
}

// A challenge confirmation job will attempt to confirm a challenge all the way up to the top,
// block challenge root edge by updating all the inherited timers of royal edges along the way,
// across all open subchallenges, until the onchain timer of the block challenge root edge
// is greater than or equal to a challenge period.
//
// It works by updating royal branches of the challenge tree, starting from the bottom-most,
// deepest level royal edges. For each branch, update the onchain inherited timers
// of the ancestors along the way.
//
// This function must only be called once the locally computed value of the block challenge, royal root
// edge has an inherited timer that is confirmable. This function MUST complete, and it will retry
// any external call if it errors during its execution.
func (cc *challengeConfirmer) beginConfirmationJob(
	ctx context.Context,
	challengedAssertionHash protocol.AssertionHash,
	computedTimer uint64,
	royalRootEdge protocol.VerifiedRoyalEdge,
	claimedAssertionHash protocol.AssertionHash,
	challengePeriodBlocks uint64,
) error {
	fields := []any{
		"validatorName", cc.validatorName,
		"challengedAssertion", fmt.Sprintf("%#x", challengedAssertionHash.Hash[:4]),
		"essentialEdgeId", fmt.Sprintf("%#x", royalRootEdge.Id().Bytes()[:4]),
		"challengeLevel", royalRootEdge.GetChallengeLevel(),
	}
	log.Info("Starting challenge confirmation job", fields...)
	// Find the bottom-most royal edges that exist in our local challenge tree, each one
	// will be the base of a branch we will update.
	royalTreeLeaves, err := retry.UntilSucceeds(ctx, func() ([]protocol.SpecEdge, error) {
		edges, innerErr := cc.reader.LowerMostRoyalEdges(ctx, challengedAssertionHash)
		if innerErr != nil {
			fields = append(fields, "err", innerErr)
			log.Error("Could not fetch lower-most royal edges", fields)
			return nil, innerErr
		}
		return edges, nil
	})
	if err != nil {
		return err
	}

	log.Info(fmt.Sprintf("Obtained all %d royal tree leaves for confirmation job", len(royalTreeLeaves)), fields...)
	// For each branch, compute the royal ancestor branch up to the root of the tree.
	// The branch should contain royal ancestors ordered from a bottom-most leaf edge to the root edge
	// of the block level challenge, meaning it should also include claim id links.
	royalBranches := make([][]protocol.ReadOnlyEdge, 0)
	for _, edge := range royalTreeLeaves {
		branch := []protocol.ReadOnlyEdge{edge}
		ancestors, err2 := retry.UntilSucceeds(ctx, func() ([]protocol.ReadOnlyEdge, error) {
			resp, innerErr := cc.reader.ComputeAncestors(
				ctx, challengedAssertionHash, edge.Id(),
			)
			if innerErr != nil {
				fields = append(fields, "err", innerErr)
				log.Error("Could not compute ancestors for edge", fields)
				return nil, innerErr
			}
			return resp, nil
		})
		if err2 != nil {
			return err2
		}
		branch = append(branch, ancestors...)
		royalBranches = append(royalBranches, branch)
	}
	log.Info("Computed all the royal branches to update onchain", fields...)

	// For each branch, update the inherited timers onchain via transactions and don't
	// wait for them to reach safe head.
	var lastPropagationTx *types.Transaction
	for i, branch := range royalBranches {
		tx, innerErr := cc.propageTimerUpdateToBranch(
			ctx,
			royalRootEdge,
			computedTimer,
			challengedAssertionHash,
			i,
			len(royalBranches),
			branch,
			challengePeriodBlocks,
			claimedAssertionHash,
		)
		if innerErr != nil {
			return innerErr
		}
		lastPropagationTx = tx
	}

	// Instead, we wait for the last transaction we made to reach `safe` head if it is not nil
	// so that we can avoid unnecessary delays per tx.
	if lastPropagationTx != nil {
		receipt, innerErr := cc.backend.TransactionReceipt(ctx, lastPropagationTx.Hash())
		if innerErr != nil {
			return innerErr
		}
		if err = cc.waitForTxToBeSafe(ctx, cc.backend, lastPropagationTx, receipt); err != nil {
			return err
		}
	}

	onchainInheritedTimer, err := retry.UntilSucceeds(ctx, func() (protocol.InheritedTimer, error) {
		timer, innerErr := royalRootEdge.LatestInheritedTimer(ctx)
		if innerErr != nil {
			fields = append(fields, "err", innerErr)
			log.Error("Could not get inherited timer for edge", fields)
			return 0, innerErr
		}
		return timer, nil
	})
	if err != nil {
		return err
	}

	// If the onchain timer is not >= a challenge period by the end of this job,
	// it means the challenge has yet to complete and our local computation was incorrect.
	// In this scenario, we can dump the confirmation job of royal edges for manual
	// inspection and debugging
	if onchainInheritedTimer < protocol.InheritedTimer(challengePeriodBlocks) {
		onchainTimerDifferAfterConfirmationJobCounter.Inc(1)
		log.Error(
			fmt.Sprintf("Onchain timer %d was not >= %d after confirmation job", onchainInheritedTimer, challengePeriodBlocks),
			fields...,
		)
		return fmt.Errorf(
			"onchain timer %d after confirmation job was executed < challenge period %d",
			onchainInheritedTimer,
			challengePeriodBlocks,
		)
	}
	log.Info("Confirming essential root edge by timer", fields...)
	if _, err = retry.UntilSucceeds(ctx, func() (bool, error) {
		if _, innerErr := royalRootEdge.ConfirmByTimer(ctx, claimedAssertionHash); innerErr != nil {
			fields = append(fields, "err", innerErr)
			log.Error("Could not confirm edge by timer", fields)
			return false, innerErr
		}
		return false, nil
	}); err != nil {
		return err
	}
	log.Info("Essential root edge confirmed by timer", fields...)
	return nil
}

func (cc *challengeConfirmer) propageTimerUpdateToBranch(
	ctx context.Context,
	royalRootEdge protocol.VerifiedRoyalEdge,
	computedLocalTimer uint64,
	challengedAssertionHash protocol.AssertionHash,
	branchIdx,
	totalBranches int,
	branch []protocol.ReadOnlyEdge,
	challengePeriodBlocks uint64,
	claimedAssertionHash protocol.AssertionHash,
) (*types.Transaction, error) {
	if len(branch) == 0 {
		return nil, nil
	}
	fields := []any{
		"validatorName", cc.validatorName,
		"challengedAssertionHash", fmt.Sprintf("%#x", challengedAssertionHash.Hash[:4]),
		"claimedAssertionHash", fmt.Sprintf("%#x", claimedAssertionHash.Hash[:4]),
		"royalRootBlockChallengeEdge", fmt.Sprintf("%#x", royalRootEdge.Id().Bytes()[:4]),
		"branch", fmt.Sprintf("%d/%d", branchIdx, totalBranches-1),
		"challengeLevel", fmt.Sprintf("%d", royalRootEdge.GetChallengeLevel()),
	}
	tx, err := retry.UntilSucceeds(ctx, func() (*types.Transaction, error) {
		tx, innerErr := cc.writer.MultiUpdateInheritedTimers(ctx, branch, computedLocalTimer)
		if innerErr != nil {
			// If are trying to update the inherited timers of a branch to a value that
			// is less than what already exists onchain, we will receive the below error.
			// This means we can finish early as the onchain timer is already sufficient,
			// and our transaction reverted. We can gracefully continue if so.
			if strings.Contains(innerErr.Error(), protocol.ErrCachedTimeSufficient) {
				log.Info("Onchain, cached timer for branch is already sufficient, so no need to transact", fields...)
				return nil, nil
			}
			fields = append(fields, "err", innerErr)
			log.Error("Could not transact multi-update inherited timers", fields...)
			return nil, innerErr
		}
		return tx, nil
	})
	if err != nil {
		return nil, err
	}

	// In each iteration, check if the root edge has a timer >= a challenge period
	rootTimer, err := retry.UntilSucceeds(ctx, func() (protocol.InheritedTimer, error) {
		timer, innerErr := royalRootEdge.LatestInheritedTimer(ctx)
		if innerErr != nil {
			fields = append(fields, "err", innerErr)
			log.Error("Could not get inherited timer for edge", fields...)
			return 0, innerErr
		}
		return timer, nil
	})
	if err != nil {
		return nil, err
	}

	fields = append(fields, "onchainTimer", rootTimer)
	log.Info("Updated the onchain inherited timer for royal branch", fields...)

	if uint64(rootTimer) < challengePeriodBlocks {
		return tx, nil
	}

	// If yes, we confirm the root edge and finish early, we do so.
	log.Info("Branch was confirmable by timer", fields...)
	tx, err = retry.UntilSucceeds(ctx, func() (*types.Transaction, error) {
		innerTx, innerErr := royalRootEdge.ConfirmByTimer(ctx, claimedAssertionHash)
		if innerErr != nil {
			fields = append(fields, "err", innerErr)
			log.Error("Could not confirm edge by timer early with confirmable branch", fields...)
			return nil, innerErr
		}
		return innerTx, nil
	})
	if err != nil {
		return nil, err
	}
	log.Info("Essential root edge confirmed by timer", fields...)
	return tx, nil
}

// waitForTxToBeSafe waits for the transaction to be mined in a block that is safe.
func (cc *challengeConfirmer) waitForTxToBeSafe(
	ctx context.Context,
	backend protocol.ChainBackend,
	tx *types.Transaction,
	receipt *types.Receipt,
) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		safeBlockNum := cc.chain.GetDesiredRpcHeadBlockNumber()
		latestSafeHeader, err := backend.HeaderByNumber(ctx, big.NewInt(int64(safeBlockNum)))
		if err != nil {
			return err
		}
		if !latestSafeHeader.Number.IsUint64() {
			return errors.New("block number is not uint64")
		}
		latestSafeHeaderNumber := latestSafeHeader.Number.Uint64()
		txSafe := latestSafeHeaderNumber >= receipt.BlockNumber.Uint64()

		// If the tx is not yet safe, we can simply wait.
		if !txSafe {
			var blocksLeftForTxToBeSafe int64
			if receipt.BlockNumber.Uint64() > latestSafeHeaderNumber {
				blocksLeftForTxToBeSafe = 0
			} else {
				blocksLeftForTxToBeSafe, err = safecast.ToInt64(latestSafeHeaderNumber - receipt.BlockNumber.Uint64())
				if err != nil {
					return errors.Wrap(err, "could not convert blocks left for tx to be safe to int64")
				}
			}
			timeToWait := cc.averageTimeForBlockCreation * time.Duration(blocksLeftForTxToBeSafe)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(timeToWait):
			}
		} else {
			break
		}
	}

	// This is to handle the case where the transaction is mined in a block, but then the block is reorged.
	// In this case, we want to wait for the transaction to be mined again.
	receiptLatest, err := bind.WaitMined(ctx, backend, tx)
	if err != nil {
		return err
	}
	// If the receipt block number is different from the latest receipt block number, we wait for the transaction
	// to be in the safe block again.
	if receiptLatest.BlockNumber.Cmp(receipt.BlockNumber) != 0 {
		return cc.waitForTxToBeSafe(ctx, backend, tx, receiptLatest)
	}
	return nil
}
