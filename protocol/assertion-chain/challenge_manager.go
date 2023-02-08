package assertionchain

import (
	"bytes"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/outgen"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
)

// ChallengeManager --
type ChallengeManager struct {
	assertionChain *AssertionChain
	caller         *outgen.ChallengeManagerImplCaller
	writer         *outgen.ChallengeManagerImplTransactor
}

func newChallengeManager() (*ChallengeManager, error) {
	return nil, nil
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

// ChallengeByID --
func (cm *ChallengeManager) ChallengeByID(challengeId common.Hash) (*Challenge, error) {
	res, err := cm.caller.GetChallenge(cm.assertionChain.callOpts, challengeId)
	if err != nil {
		return nil, err
	}
	if bytes.Equal(res.RootId[:], make([]byte, 32)) {
		return nil, errors.Wrapf(
			ErrNotFound,
			"challenge with id %#x",
			challengeId,
		)
	}
	return &Challenge{
		inner:   res,
		manager: cm,
	}, nil
}
