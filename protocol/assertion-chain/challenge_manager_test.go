package assertionchain

import (
	"testing"

	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestGetChallengeByID(t *testing.T) {
	genesisId := common.Hash{}
	chain, acc := setupAssertionChainWithChallengeManager(t)

	commit1 := util.StateCommitment{
		Height:    1,
		StateRoot: common.BytesToHash([]byte{1}),
	}
	_, err := chain.CreateAssertion(commit1, genesisId)
	require.NoError(t, err)
	acc.backend.Commit()
	commit2 := util.StateCommitment{
		Height:    1,
		StateRoot: common.BytesToHash([]byte{2}),
	}

	_, err = chain.CreateAssertion(commit2, genesisId)
	require.NoError(t, err)
	acc.backend.Commit()

	_, err = chain.CreateSuccessionChallenge(genesisId)
	require.NoError(t, err)
	acc.backend.Commit()

	cm, err := chain.ChallengeManager()
	require.NoError(t, err)

	t.Run("challenge does not exists", func(t *testing.T) {
		_, err = cm.ChallengeByID(genesisId)
		require.ErrorContains(t, err, "challenge not found")
	})

	t.Run("challenge exists", func(t *testing.T) {
		cid, err := cm.CalculateChallengeId(genesisId, 0)
		require.NoError(t, err)
		challenge, err := cm.ChallengeByID(cid)
		require.NoError(t, err)
		require.Equal(t, uint8(0), challenge.inner.ChallengeType)
		require.Equal(t, [32]byte{}, challenge.inner.WinningClaim)
	})
}
