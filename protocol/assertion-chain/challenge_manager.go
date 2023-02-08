package assertionchain

import (
	"bytes"
	"strings"

	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/outgen"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
)

var (
	ErrChallengeNotFound = errors.New("challenge not found")
)

// ChallengeManager --
type ChallengeManager struct {
	assertionChain *AssertionChain
	caller         *outgen.ChallengeManagerImplCaller
	writer         *outgen.ChallengeManagerImplTransactor
}

// ChallengeManager returns an instance of the current challenge manager
// used by the assertion chain.
func (ac *AssertionChain) ChallengeManager() (*ChallengeManager, error) {
	addr, err := ac.caller.ChallengeManagerAddr(ac.callOpts)
	if err != nil {
		return nil, err
	}
	managerBinding, err := outgen.NewChallengeManagerImpl(addr, ac.backend)
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
func (cm *ChallengeManager) CalculateChallengeId(assertionId common.Hash, cType ChallengeType) (common.Hash, error) {
	c, err := cm.caller.CalculateChallengeId(cm.assertionChain.callOpts, assertionId, uint8(cType))
	if err != nil {
		return common.Hash{}, err
	}
	return c, nil
}

// ChallengeByID returns a challenge by its challenge ID.
func (cm *ChallengeManager) ChallengeByID(challengeID common.Hash) (*Challenge, error) {
	c, err := cm.caller.GetChallenge(cm.assertionChain.callOpts, challengeID)
	switch {
	case bytes.Equal(c.RootId[:], make([]byte, 32)):
		return nil, errors.Wrapf(
			ErrChallengeNotFound,
			"challenge with id %#x",
			challengeID,
		)
	case err == nil:
		return &Challenge{inner: c}, nil
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
