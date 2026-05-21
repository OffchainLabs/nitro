// Copyright 2023-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/bold/protocol"
	"github.com/offchainlabs/nitro/bold/retry"
	"github.com/offchainlabs/nitro/bold/testing/setup"
	"github.com/offchainlabs/nitro/solgen/go/challengeV2gen"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
)

// expect is a function that will be called asynchronously to verify some success criteria
// for the given scenario.
type expect func(t *testing.T, ctx context.Context, addresses *setup.RollupAddresses, be protocol.ChainBackend, honestValidatorAddress common.Address) error

// Expects that an assertion is confirmed by challenge win.
func expectChallengeWinWithAllHonestEssentialEdgesConfirmed(
	t *testing.T,
	ctx context.Context,
	addresses *setup.RollupAddresses,
	backend protocol.ChainBackend,
	honestValidatorAddress common.Address,
) error {
	t.Run("honest essential edges confirmed by challenge win", func(t *testing.T) {
		rc, err := rollupgen.NewRollupCore(addresses.Rollup, backend)
		require.NoError(t, err)
		cmAddr, err := rc.ChallengeManager(&bind.CallOpts{})
		require.NoError(t, err)

		// Wait until a challenged assertion is confirmed by time, with a per-test
		// deadline hardcoded to 20m for the same reason the second loop's 30m is
		// hardcoded: a tight cap below the CI job timeout so a stall fails fast
		// with a structured message instead of silently consuming the CI budget.
		firstWaitCtx, firstCancel := context.WithTimeout(ctx, 20*time.Minute)
		defer firstCancel()
		firstStart := time.Now()
		var confirmed bool
		for firstWaitCtx.Err() == nil && !confirmed {
			var i *rollupgen.RollupCoreAssertionConfirmedIterator
			i, err = retry.UntilSucceeds(firstWaitCtx, func() (*rollupgen.RollupCoreAssertionConfirmedIterator, error) {
				return rc.FilterAssertionConfirmed(nil, nil)
			})
			if err != nil {
				if firstWaitCtx.Err() == nil {
					t.Fatalf("retry.UntilSucceeds returned err with live ctx, contract violated: %v", err)
				}
				break
			}
			for i.Next() {
				var assertionNode rollupgen.AssertionNode
				assertionNode, err = retry.UntilSucceeds(firstWaitCtx, func() (rollupgen.AssertionNode, error) {
					return rc.GetAssertion(&bind.CallOpts{Context: firstWaitCtx}, i.Event.AssertionHash)
				})
				if err != nil {
					if firstWaitCtx.Err() == nil {
						t.Fatalf("retry.UntilSucceeds returned err with live ctx, contract violated: %v", err)
					}
					break
				}
				isChallengeParent := assertionNode.FirstChildBlock > 0 && assertionNode.SecondChildBlock > 0
				if isChallengeParent && assertionNode.Status != uint8(protocol.AssertionConfirmed) {
					t.Fatal("Confirmed assertion with unfinished state")
				}
				confirmed = true
				break
			}
			if confirmed {
				break
			}
			select {
			case <-firstWaitCtx.Done():
			case <-time.After(500 * time.Millisecond): // Don't spam the backend.
			}
		}

		if !confirmed {
			t.Fatalf("timed out after %v waiting for challenged assertion to be confirmed", time.Since(firstStart))
		}

		// The challenge has confirmed by this point, so no further edges will
		// be added. Scrape the edges added so far, then wait until all of the
		// essential root edges among them are confirmed.
		cm, err := challengeV2gen.NewEdgeChallengeManager(cmAddr, backend)
		require.NoError(t, err)

		// Scrape all the honest edges onchain (the ones made by the honest address).
		// Check if the edges that have claim id != None are confirmed (those are essential root edges)
		// and also check one step edges from honest party are confirmed.
		honestEssentialRootIds := make(map[common.Hash]bool, 0)
		chainId, err := backend.ChainID(ctx)
		require.NoError(t, err)
		it, err := cm.FilterEdgeAdded(nil, nil, nil, nil)
		require.NoError(t, err)
		defer func() {
			if cerr := it.Close(); cerr != nil {
				t.Logf("could not close edge-added iterator: %v", cerr)
			}
		}()
		totalEvents := 0
		for it.Next() {
			require.NoError(t, it.Error(), "iterator error during edge-added scan")
			totalEvents++
			txHash := it.Event.Raw.TxHash
			tx, _, err := backend.TransactionByHash(ctx, txHash)
			require.NoError(t, err)
			sender, err := types.Sender(types.NewCancunSigner(chainId), tx)
			require.NoError(t, err)
			if sender != honestValidatorAddress {
				continue
			}
			// Skip edges that are not essential roots or the top-level challenge root.
			if it.Event.ClaimId == (common.Hash{}) || it.Event.Level == 0 {
				continue
			}
			honestEssentialRootIds[it.Event.EdgeId] = false
		}
		require.NoError(t, it.Error(), "iterator error after edge-added scan")
		require.NotEmpty(t, honestEssentialRootIds,
			"no honest essential root edges discovered for honest validator %s; FilterEdgeAdded matched %d total events",
			honestValidatorAddress.Hex(), totalEvents)

		// Wait until all of the honest essential root ids are confirmed, with a
		// per-test deadline (30m) well below the CI job timeout (--timeout 90m
		// in .github/workflows/_go-tests.yml). On stall this fails fast with
		// the unconfirmed edge IDs instead of letting the parent ctx expire
		// silently and burning the entire CI budget.
		waitCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
		defer cancel()
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		waitStart := time.Now()
		edgeStates := make(map[common.Hash]edgeState, len(honestEssentialRootIds))
		confirmedCount := 0
		for confirmedCount < len(honestEssentialRootIds) {
			for k, markedConfirmed := range honestEssentialRootIds {
				if markedConfirmed {
					continue
				}
				// retry.UntilSucceeds only returns an error when its context is
				// done, and we pass waitCtx here, so an error necessarily means
				// waitCtx is expired. Break out of the inner loop and let the
				// outer select hit its <-waitCtx.Done() case, which emits the
				// structured diagnostic with unconfirmed edge IDs. Do NOT pass
				// a shorter-lived context to retry.UntilSucceeds without
				// revisiting this; the assertion below pins the contract at
				// the call site so a future refactor of retry.UntilSucceeds
				// surfaces immediately instead of silently spinning.
				edge, err := retry.UntilSucceeds(waitCtx, func() (challengeV2gen.ChallengeEdge, error) {
					return cm.GetEdge(&bind.CallOpts{Context: waitCtx}, k)
				})
				if err != nil {
					if waitCtx.Err() == nil {
						t.Fatalf("retry.UntilSucceeds returned err with live ctx, contract violated: %v", err)
					}
					break
				}
				edgeStates[k] = edgeState{level: edge.Level, status: edge.Status}
				if edge.Status == uint8(protocol.EdgeConfirmed) {
					confirmedCount += 1
					honestEssentialRootIds[k] = true
					t.Logf("Confirmed %d/%d honest essential edges, got edge at level %d", confirmedCount, len(honestEssentialRootIds), edge.Level)
				}
			}
			if confirmedCount >= len(honestEssentialRootIds) {
				break
			}
			select {
			case <-waitCtx.Done():
				unconfirmed := make([]string, 0, len(honestEssentialRootIds)-confirmedCount)
				for k, markedConfirmed := range honestEssentialRootIds {
					if markedConfirmed {
						continue
					}
					if s, ok := edgeStates[k]; ok {
						unconfirmed = append(unconfirmed, fmt.Sprintf("%s level=%d status=%d", k.Hex(), s.level, s.status))
					} else {
						unconfirmed = append(unconfirmed, fmt.Sprintf("%s (never observed)", k.Hex()))
					}
				}
				t.Fatalf("timed out after %v waiting for honest essential edges: %d/%d confirmed, unconfirmed=%v", time.Since(waitStart), confirmedCount, len(honestEssentialRootIds), unconfirmed)
			case <-ticker.C:
			}
		}
	})
	return nil
}

type edgeState struct {
	level  uint8
	status uint8
}
