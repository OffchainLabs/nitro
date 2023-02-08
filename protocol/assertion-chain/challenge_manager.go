package assertionchain

import (
	"bytes"
	"strings"

	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/outgen"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
)

var (
	ErrChallengeNotFound       = errors.New("challenge not found")
	ErrChallengeExists         = errors.New("challenge already exists")
	ErrInvalidCaller           = errors.New("invalid caller")
	ErrChallengeVertexNotFound = errors.New("challenge vertex not found")
)

// ChallengeManager --
type ChallengeManager struct {
	assertionChain *AssertionChain
	caller         *outgen.ChallengeManagerCaller
	writer         *outgen.ChallengeManagerTransactor
	txOpts         *bind.TransactOpts
}

// Challenge is a wrapper around solgen bindings.
type Challenge struct {
	inner outgen.Challenge
}

// ChallengeVertex is a wrapper around solgen bindings.
type ChallengeVertex struct {
	inner outgen.ChallengeVertex
}

// ChallengeManager returns an instance of the current challenge manager
// used by the assertion chain.
func (ac *AssertionChain) ChallengeManager() (*ChallengeManager, error) {
	addr, err := ac.caller.ChallengeManagerAddr(ac.callOpts)
	if err != nil {
		return nil, err
	}
	managerBinding, err := outgen.NewChallengeManager(addr, ac.backend)
	if err != nil {
		return nil, err
	}
	return &ChallengeManager{
		assertionChain: ac,
		caller:         &managerBinding.ChallengeManagerCaller,
		writer:         &managerBinding.ChallengeManagerTransactor,
		txOpts:         ac.txOpts,
	}, nil
}

// ChallengeByID returns a challenge by its challenge ID.
func (cm *ChallengeManager) ChallengeByID(challengeID common.Hash) (*Challenge, error) {
	c, err := cm.caller.GetChallenge(cm.assertionChain.callOpts, challengeID)
	if err != nil {
		return nil, err
	}
	switch {
	case bytes.Equal(c.RootId[:], make([]byte, 32)):
		return nil, errors.Wrapf(
			ErrNotFound,
			"challenge with id %#x",
			challengeID,
		)
	case strings.Contains(err.Error(), "Vertex does not exist"):
		return nil, errors.Wrapf(
			ErrChallengeNotFound,
			"challenge id %#x",
			challengeID,
		)
	case err != nil:
		return nil, err
	}
	return &Challenge{inner: c}, nil
}
