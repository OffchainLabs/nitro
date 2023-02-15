package solimpl

import (
	"bytes"
	"math/big"
	"strings"

	"context"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/challengeV2gen"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
)

var (
	ErrChallengeNotFound = errors.New("challenge not found")
	ErrPsTimerNotYet     = errors.New("ps timer has not exceeded challenge period")
)

// ChallengeManager --
type ChallengeManager struct {
	assertionChain *AssertionChain
	caller         *challengeV2gen.ChallengeManagerImplCaller
	writer         *challengeV2gen.ChallengeManagerImplTransactor
}

// ChallengeManager returns an instance of the current challenge manager
// used by the assertion chain.
func (ac *AssertionChain) ChallengeManager() (*ChallengeManager, error) {
	addr, err := ac.userLogic.ChallengeManager(ac.callOpts)
	if err != nil {
		return nil, err
	}
	managerBinding, err := challengeV2gen.NewChallengeManagerImpl(addr, ac.backend)
	if err != nil {
		return nil, err
	}
	return &ChallengeManager{
		assertionChain: ac,
		caller:         &managerBinding.ChallengeManagerImplCaller,
		writer:         &managerBinding.ChallengeManagerImplTransactor,
	}, nil
}

// CalculateChallengeId calculates the challenge ID for a given assertion and challenge type.
func (cm *ChallengeManager) CalculateChallengeId(ctx context.Context, assertionId common.Hash, cType ChallengeType) (common.Hash, error) {
	c, err := cm.caller.CalculateChallengeId(cm.assertionChain.callOpts, assertionId, uint8(cType))
	if err != nil {
		return common.Hash{}, err
	}
	return c, nil
}

// ChallengeByID returns a challenge by its challenge ID.
func (cm *ChallengeManager) ChallengeByID(ctx context.Context, challengeID common.Hash) (*Challenge, error) {
	c, err := cm.caller.GetChallenge(cm.assertionChain.callOpts, challengeID)
	switch {
	case bytes.Equal(c.RootId[:], make([]byte, 32)):
		return nil, errors.Wrapf(
			ErrChallengeNotFound,
			"challenge with id %#x",
			challengeID,
		)
	case err == nil:
		return &Challenge{
			id:      challengeID,
			inner:   c,
			manager: cm,
		}, nil
	case strings.Contains(err.Error(), "Vertex does not exist"):
		return nil, errors.Wrapf(
			ErrChallengeNotFound,
			"challenge id %#x",
			challengeID,
		)
	default:
		return nil, err
	}
}

//nolint:unused
func (cm *ChallengeManager) miniStakeAmount() (*big.Int, error) {
	return cm.caller.MiniStakeValue(cm.assertionChain.callOpts)
}
