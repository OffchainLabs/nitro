package solimpl

import (
	"context"
	"testing"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

var _ = protocol.ChallengeManager(&ChallengeManager{})

func TestGetChallengeByID(t *testing.T) {
	ctx := context.Background()
	height1 := uint64(6)
	height2 := uint64(7)
	_, _, challenge, chain, _ := setupTopLevelFork(t, ctx, height1, height2)

	cm, err := chain.CurrentChallengeManager(ctx)
	require.NoError(t, err)

	t.Run("challenge does not exist", func(t *testing.T) {
		_, err = cm.GetChallenge(ctx, protocol.ChallengeHash(common.Hash{}))
		require.ErrorContains(t, err, "does not exist")
	})

	t.Run("challenge exists", func(t *testing.T) {
		fetched, err := cm.GetChallenge(ctx, challenge.id)
		require.NoError(t, err)
		require.Equal(t, false, fetched.IsNone())
		fChal := fetched.Unwrap()

		fChalType := fChal.GetType()
		fChalWinningClaim, err := fChal.WinningClaim(ctx)
		require.NoError(t, err)
		require.Equal(t, protocol.BlockChallenge, fChalType)
		require.Equal(t, true, fChalWinningClaim.IsNone())
	})
}
