package assertionchain

import (
	"context"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/outgen"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"time"
)

// type AssertionManager interface {
// 	Inbox() *Inbox
// 	NumAssertions(tx *ActiveTx) uint64
// 	AssertionBySequenceNum(tx *ActiveTx, seqNum AssertionSequenceNumber) (*Assertion, error)
// 	ChallengeByCommitHash(tx *ActiveTx, commitHash ChallengeCommitHash) (*Challenge, error)
// 	ChallengeVertexByCommitHash(tx *ActiveTx, challenge ChallengeCommitHash, vertex VertexCommitHash) (*ChallengeVertex, error)
// 	IsAtOneStepFork(
// 		tx *ActiveTx,
// 		challengeCommitHash ChallengeCommitHash,
// 		vertexCommit util.HistoryCommitment,
// 		vertexParentCommit util.HistoryCommitment,
// 	) (bool, error)
// 	ChallengePeriodLength(tx *ActiveTx) time.Duration
// 	LatestConfirmed(*ActiveTx) *Assertion
// 	CreateLeaf(tx *ActiveTx, prev *Assertion, commitment util.StateCommitment, staker common.Address) (*Assertion, error)
// 	TimeReference() util.TimeReference
// }

type Opt func(chain *AssertionChain) error

func WithBackend(b bind.ContractBackend) Opt {
	return func(chain *AssertionChain) error {
		chain.backend = b
		return nil
	}
}

type AssertionChain struct {
	backend bind.ContractBackend
	binding *outgen.IAssertionChain
}

func (ac *AssertionChain) ChalengePeriodLength() time.Duration {
	return time.Second
}

func NewAssertionChain(
	ctx context.Context,
	contractAddr common.Address,
	txOpts *bind.TransactOpts,
	stakerAddr common.Address,
	opts ...Opt,
) (*AssertionChain, error) {
	chain := &AssertionChain{}
	for _, o := range opts {
		if err := o(chain); err != nil {
			return nil, err
		}
	}
	assertionChainBinding, err := outgen.NewIAssertionChain(contractAddr, chain.backend)

	if err != nil {
		return nil, err
	}
	chain.binding = assertionChainBinding
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
