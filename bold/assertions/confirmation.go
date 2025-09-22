// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

package assertions

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ccoveille/go-safecast"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/bold/chain-abstraction"
	"github.com/offchainlabs/nitro/bold/chain-abstraction/sol-implementation"
	"github.com/offchainlabs/nitro/bold/challenge-manager/types"
	"github.com/offchainlabs/nitro/bold/containers/option"
	"github.com/offchainlabs/nitro/bold/logs/ephemeral"
	"github.com/offchainlabs/nitro/bold/runtime"
)

func (m *Manager) queueCanonicalAssertionsForConfirmation(ctx context.Context) {
	for {
		select {
		case canonical := <-m.observedCanonicalAssertions:
			m.LaunchThread(func(ctx context.Context) { m.keepTryingAssertionConfirmation(ctx, canonical) })
		case <-ctx.Done():
			return
		}
	}
}

func (m *Manager) keepTryingAssertionConfirmation(ctx context.Context, assertionHash protocol.AssertionHash) {
	// Only resolve mode strategies or higher should be confirming assertions.
	if m.mode < types.ResolveMode {
		return
	}
	creationInfo, err := retry.UntilSucceeds(ctx, func() (*protocol.AssertionCreatedInfo, error) {
		return m.chain.ReadAssertionCreationInfo(ctx, assertionHash)
	})
	if err != nil {
		log.Error("Could not get assertion creation info", "err", err)
		return
	}
	if m.enableFastConfirmation {
		var confirmed bool
		confirmed, err = m.chain.FastConfirmAssertion(ctx, creationInfo)
		if err != nil {
			log.Error("Could not fast confirm latest assertion", "err", err)
		} else if confirmed {
			assertionConfirmedCounter.Inc(1)
			log.Info("Fast Confirmed assertion", "assertionHash", creationInfo.AssertionHash)
			return
		}
	}
	prevCreationInfo, err := retry.UntilSucceeds(ctx, func() (*protocol.AssertionCreatedInfo, error) {
		return m.chain.ReadAssertionCreationInfo(ctx, creationInfo.ParentAssertionHash)
	})
	if err != nil {
		log.Error("Could not get prev assertion creation info", "err", err)
		return
	}
	exceedsMaxMempoolSizeEphemeralErrorHandler := ephemeral.NewEphemeralErrorHandler(10*time.Minute, "posting this transaction will exceed max mempool size", 0)
	gasEstimationEphemeralErrorHandler := ephemeral.NewEphemeralErrorHandler(10*time.Minute, "gas estimation errored for tx with hash", 0)
	ticker := time.NewTicker(m.times.confInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if m.enableFastConfirmation {
				var confirmed bool
				confirmed, err = m.chain.FastConfirmAssertion(ctx, creationInfo)
				if err != nil {
					log.Error("Could not fast confirm latest assertion", "err", err)
				} else if confirmed {
					assertionConfirmedCounter.Inc(1)
					log.Info("Fast Confirmed assertion", "assertionHash", creationInfo.AssertionHash)
					return
				}
			}
			opts := m.chain.GetCallOptsWithDesiredRpcHeadBlockNumber(&bind.CallOpts{Context: ctx})
			parentAssertion, err := m.chain.GetAssertion(
				ctx,
				opts,
				creationInfo.ParentAssertionHash,
			)
			if err != nil {
				log.Error("Could not get parent assertion", "err", err)
				continue
			}
			parentAssertionHasSecondChild, err := parentAssertion.HasSecondChild(ctx, opts)
			if err != nil {
				log.Error("Could not confirm if parent assertion has second child", "err", err)
				continue
			}
			// Assertions that have a rival assertion cannot be confirmed by time.
			if parentAssertionHasSecondChild {
				return
			}
			confirmed, err := solimpl.TryConfirmingAssertion(ctx, creationInfo.AssertionHash, prevCreationInfo.ConfirmPeriodBlocks+creationInfo.CreationL1Block, m.chain, m.times.avgBlockTime, option.None[protocol.EdgeId]())
			if err != nil {
				if !strings.Contains(err.Error(), "PREV_NOT_LATEST_CONFIRMED") {
					logLevel := log.Error
					logLevel = exceedsMaxMempoolSizeEphemeralErrorHandler.LogLevel(err, logLevel)
					logLevel = gasEstimationEphemeralErrorHandler.LogLevel(err, logLevel)

					logLevel("Could not confirm assertion", "err", err, "assertionHash", assertionHash.Hash)
					errorConfirmingAssertionByTimeCounter.Inc(1)
				}
				continue
			}

			exceedsMaxMempoolSizeEphemeralErrorHandler.Reset()
			gasEstimationEphemeralErrorHandler.Reset()

			if confirmed {
				assertionConfirmedCounter.Inc(1)
				log.Info("Confirmed assertion by time", "assertionHash", creationInfo.AssertionHash)
				return
			}
		}
	}
}

func (m *Manager) updateLatestConfirmedMetrics(ctx context.Context) {
	ticker := time.NewTicker(m.times.confInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			latestConfirmed, err := m.chain.LatestConfirmed(ctx, m.chain.GetCallOptsWithDesiredRpcHeadBlockNumber(&bind.CallOpts{Context: ctx}))
			if err != nil {
				log.Debug("Could not fetch latest confirmed assertion", "err", err)
				continue
			}
			info, err := m.chain.ReadAssertionCreationInfo(ctx, latestConfirmed.Id())
			if err != nil {
				log.Debug("Could not fetch latest confirmed assertion", "err", err)
				continue
			}
			afterState := protocol.GoExecutionStateFromSolidity(info.AfterState)
			log.Info("Latest confirmed assertion", "assertionAfterState", fmt.Sprintf("%+v", afterState))

			// TODO: Check if the latest assertion that was confirmed is one we agree with.
			latestConfirmedBlockNum, err := safecast.ToInt64(latestConfirmed.CreatedAtBlock())
			if err != nil {
				log.Error("Could not convert latest confirmed block number to int64", "err", err)
				continue
			}
			latestConfirmedAssertionGauge.Update(latestConfirmedBlockNum)
		case <-ctx.Done():
			return
		}
	}
}
