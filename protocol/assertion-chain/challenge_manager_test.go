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

	err := chain.createAssertion(commit1, genesisId)
	require.NoError(t, err)
	acc.backend.Commit()

	commit2 := util.StateCommitment{
		Height:    1,
		StateRoot: common.BytesToHash([]byte{2}),
	}

	err = chain.createAssertion(commit2, genesisId)
	require.NoError(t, err)
	acc.backend.Commit()

	err = chain.CreateSuccessionChallenge(genesisId)
	require.NoError(t, err)
	acc.backend.Commit()

	cm, err := chain.ChallengeManager()
	require.NoError(t, err)

	_, err = cm.ChallengeByID(genesisId)
	require.NoError(t, err)
}
