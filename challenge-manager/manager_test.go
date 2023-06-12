package validator

import (
	"context"
	"testing"

	"github.com/OffchainLabs/challenge-protocol-v2/testing/mocks"
	"github.com/OffchainLabs/challenge-protocol-v2/testing/setup"
	"github.com/stretchr/testify/require"
)

var _ = ChallengeCreator(&Manager{})

func setupValidator(t *testing.T) (*Manager, *mocks.MockProtocol, *mocks.MockStateManager) {
	t.Helper()
	p := &mocks.MockProtocol{}
	ctx := context.Background()
	p.On("CurrentChallengeManager", ctx).Return(&mocks.MockChallengeManager{}, nil)
	p.On("SpecChallengeManager", ctx).Return(&mocks.MockSpecChallengeManager{}, nil)
	s := &mocks.MockStateManager{}
	cfg, err := setup.ChainsWithEdgeChallengeManager()
	require.NoError(t, err)
	v, err := New(context.Background(), p, cfg.Backend, s, cfg.Addrs.Rollup)
	require.NoError(t, err)
	return v, p, s
}
