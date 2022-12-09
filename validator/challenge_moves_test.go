package validator

import (
	"context"
	"testing"

	"github.com/OffchainLabs/new-rollup-exploration/protocol"
	"github.com/OffchainLabs/new-rollup-exploration/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func Test_bisect(t *testing.T) {
	ctx := context.Background()
	_, _, validator := createTwoValidatorFork(t, ctx)
	vertex := &protocol.ChallengeVertex{
		Prev: &protocol.ChallengeVertex{
			Commitment: util.HistoryCommitment{
				Height: 0,
				Merkle: common.BytesToHash([]byte{0}),
			},
		},
	}
	bisectedVertex, err := validator.bisect(ctx, vertex)
	require.NoError(t, err)
	_ = bisectedVertex
}
