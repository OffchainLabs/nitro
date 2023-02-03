package assertionchain

import (
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
	callOpts       *bind.CallOpts
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
		callOpts:       ac.callOpts,
		txOpts:         ac.txOpts,
	}, nil
}

func (cm *ChallengeManager) GetChallenge(challengeID common.Hash) (*Challenge, error) {
	c, err := cm.caller.GetChallenge(cm.callOpts, challengeID)
	if err != nil {
		return nil, err
	}
	switch {
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

func (cm *ChallengeManager) GetVertex(vertexID common.Hash) (*ChallengeVertex, error) {
	v, err := cm.caller.GetVertex(cm.callOpts, vertexID)
	switch {
	case strings.Contains(err.Error(), "Vertex does not exist"):
		return nil, errors.Wrapf(
			ErrChallengeVertexNotFound,
			"challenge vertex id %#x",
			vertexID,
		)
	case err != nil:
		return nil, err
	}
	return &ChallengeVertex{inner: v}, nil
}

func (cm *ChallengeManager) CreateChallenge(assertionId common.Hash) error {
	tx, err := cm.writer.CreateChallenge(cm.txOpts, assertionId)
	switch {
	case strings.Contains(err.Error(), "Only assertion chain can create challenges"):
		return errors.Wrap(ErrInvalidCaller, err.Error())
	case strings.Contains(err.Error(), "Challenge already exists"):
		return errors.Wrapf(ErrChallengeExists, "assertion id %#x", assertionId)
	case err != nil:
		return err
	}
	// TODO: What to do with tx here?
	_ = tx
	return nil
}

func (cm *ChallengeManager) Bisect(vertexId common.Hash, prefixHistoryCommitment common.Hash, prefixProof []byte) error {
	tx, err := cm.writer.Bisect(cm.txOpts, vertexId, prefixHistoryCommitment, prefixProof)
	if err != nil {
		return err
	}
	// TODO: What to do with tx here?
	_ = tx
	return nil
}

func (cm *ChallengeManager) Merge(vertexId common.Hash, prefixHistoryCommitment common.Hash, prefixProof []byte) error {
	tx, err := cm.writer.Merge(cm.txOpts, vertexId, prefixHistoryCommitment, prefixProof)
	if err != nil {
		return err
	}
	// TODO: What to do with tx here?
	_ = tx
	return nil
}

func (cm *ChallengeManager) AddLeaf(addLeafArg outgen.AddLeafArgs, proof1, proof2 []byte) error {
	tx, err := cm.writer.AddLeaf(cm.txOpts, addLeafArg, proof1, proof2)
	if err != nil {
		return err
	}
	// TODO: Add better error h andling here
	// TODO: What to do with tx here?
	_ = tx
	return nil
}
