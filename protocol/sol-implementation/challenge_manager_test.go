package solimpl

import (
	"context"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetChallengeByID(t *testing.T) {
	ctx := context.Background()
	height1 := uint64(6)
	height2 := uint64(7)
	_, _, challenge, chain, _ := setupTopLevelFork(t, ctx, height1, height2)

	cm, err := chain.ChallengeManager()
	require.NoError(t, err)

	t.Run("challenge does not exists", func(t *testing.T) {
		_, err = cm.ChallengeByID(ctx, common.Hash{})
		require.ErrorContains(t, err, "challenge not found")
	})

	t.Run("challenge exists", func(t *testing.T) {
		fetched, err := cm.ChallengeByID(ctx, challenge.id)
		require.NoError(t, err)
		require.Equal(t, uint8(0), fetched.inner.ChallengeType)
		require.Equal(t, [32]byte{}, fetched.inner.WinningClaim)
	})
}
