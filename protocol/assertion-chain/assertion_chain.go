package assertionchain

import (
	"context"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/outgen"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"time"
)

//	type AssertionManager interface {
//		Inbox() *Inbox
//		NumAssertions(tx *ActiveTx) uint64
//		AssertionBySequenceNum(tx *ActiveTx, seqNum AssertionSequenceNumber) (*Assertion, error)
//		ChallengeByCommitHash(tx *ActiveTx, commitHash ChallengeCommitHash) (*Challenge, error)
//		ChallengeVertexByCommitHash(tx *ActiveTx, challenge ChallengeCommitHash, vertex VertexCommitHash) (*ChallengeVertex, error)
//		IsAtOneStepFork(
//			tx *ActiveTx,
//			challengeCommitHash ChallengeCommitHash,
//			vertexCommit util.HistoryCommitment,
//			vertexParentCommit util.HistoryCommitment,
//		) (bool, error)
//		ChallengePeriodLength(tx *ActiveTx) time.Duration
//		LatestConfirmed(*ActiveTx) *Assertion
//		CreateLeaf(tx *ActiveTx, prev *Assertion, commitment util.StateCommitment, staker common.Address) (*Assertion, error)
//		TimeReference() util.TimeReference
//	}
type Opt func(chain *AssertionChain) error

func WithBackend(b bind.ContractBackend) Opt {
	return func(chain *AssertionChain) error {
		chain.backend = b
		return nil
	}
}

type Assertion struct {
	inner outgen.Assertion
}

type AssertionChain struct {
	backend  bind.ContractBackend
	caller   *outgen.AssertionChainV2Caller
	writer   *outgen.AssertionChainV2Transactor
	callOpts *bind.CallOpts
}

type ChallengeManager struct {
	assertionChain *AssertionChain
	caller         *outgen.ChallengeManagerCaller
	writer         *outgen.ChallengeManagerTransactor
	callOpts       *bind.CallOpts
}

type Challenge struct {
	inner outgen.Challenge
}

func (m *ChallengeManager) ChallengeByID(challengeId common.Hash) (*Challenge, error) {
	res, err := m.caller.GetChallenge(m.callOpts, challengeId)
	if err != nil {
		return nil, err
	}
	return &Challenge{
		inner: res,
	}, nil
}

// Returns a challenge manager instance.
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
	}, nil
}

func (ac *AssertionChain) ChalengePeriodLength() (time.Duration, error) {
	res, err := ac.caller.ChallengePeriod(ac.callOpts)
	if err != nil {
		return time.Second, err
	}
	return time.Second * time.Duration(res.Uint64()), nil
}

func (ac *AssertionChain) AssertionByID(assertionId common.Hash) (*Assertion, error) {
	res, err := ac.caller.GetAssertion(ac.callOpts, assertionId)
	if err != nil {
		return nil, err
	}
	return &Assertion{
		inner: res,
	}, nil
}

func NewAssertionChain(
	ctx context.Context,
	contractAddr common.Address,
	txOpts *bind.TransactOpts,
	callOpts *bind.CallOpts,
	stakerAddr common.Address,
	opts ...Opt,
) (*AssertionChain, error) {
	chain := &AssertionChain{
		callOpts: callOpts,
	}
	for _, o := range opts {
		if err := o(chain); err != nil {
			return nil, err
		}
	}
	assertionChainBinding, err := outgen.NewAssertionChainV2(
		contractAddr, chain.backend,
	)

	if err != nil {
		return nil, err
	}
	chain.caller = &assertionChainBinding.AssertionChainV2Caller
	chain.writer = &assertionChainBinding.AssertionChainV2Transactor
	return chain, nil

	// // Attach the clients to the service struct.
	// fetcher := ethclient.NewClient(client)
	// s.rpcClient = client
	// s.httpLogger = fetcher

	// depositContractCaller, err := contracts.NewDepositContractCaller(s.cfg.depositContractAddr, fetcher)
	// if err != nil {
	// 	client.Close()
	// 	return errors.Wrap(err, "could not initialize deposit contract caller")
	// }
	// s.depositContractCaller = depositContractCaller
}
