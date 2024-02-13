package endtoend

import (
	"context"
	"testing"
	"time"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	retry "github.com/OffchainLabs/bold/runtime"
	"github.com/OffchainLabs/bold/solgen/go/rollupgen"
	"github.com/OffchainLabs/bold/testing/setup"
	"github.com/OffchainLabs/bold/util"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/stretchr/testify/require"
)

// expect is a function that will be called asynchronously to verify some success criteria
// for the given scenario.
type expect func(t *testing.T, ctx context.Context, addresses *setup.RollupAddresses, be protocol.ChainBackend) error

// Expects that an assertion is confirmed by challenge win.
func expectAssertionConfirmedByChallengeWin(
	t *testing.T,
	ctx context.Context,
	addresses *setup.RollupAddresses,
	backend protocol.ChainBackend,
) error {
	t.Run("assertion confirmed by challenge win", func(t *testing.T) {
		rc, err := rollupgen.NewRollupCore(addresses.Rollup, backend)
		require.NoError(t, err)

		var confirmed bool
		for ctx.Err() == nil && !confirmed {
			i, err := retry.UntilSucceeds(ctx, func() (*rollupgen.RollupCoreAssertionConfirmedIterator, error) {
				return rc.FilterAssertionConfirmed(nil, nil)
			})
			require.NoError(t, err)
			for i.Next() {
				assertionNode, err := retry.UntilSucceeds(ctx, func() (rollupgen.AssertionNode, error) {
					return rc.GetAssertion(util.GetFinalizedCallOpts(&bind.CallOpts{Context: ctx}), i.Event.AssertionHash)
				})
				require.NoError(t, err)
				if assertionNode.Status != uint8(protocol.AssertionConfirmed) {
					t.Fatal("Confirmed assertion with unfinished state")
				}
				confirmed = true
				break
			}
			time.Sleep(500 * time.Millisecond) // Don't spam the backend.
		}

		if !confirmed {
			t.Fatal("assertion was not confirmed")
		}
	})
	return nil
}
