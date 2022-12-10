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
	t.Run("bad bisection points", func(t *testing.T) {
		vertex := &protocol.ChallengeVertex{
			Commitment: util.HistoryCommitment{
				Height: 0,
				Merkle: common.BytesToHash([]byte{1}),
			},
			Prev: &protocol.ChallengeVertex{
				Commitment: util.HistoryCommitment{
					Height: 3,
					Merkle: common.BytesToHash([]byte{0}),
				},
			},
		}
		_, err := validator.bisect(ctx, vertex)
		require.ErrorContains(t, err, "determining bisection point failed")
	})
}
