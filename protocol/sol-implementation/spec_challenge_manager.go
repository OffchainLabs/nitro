package solimpl

import (
	"context"
	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/mocksgen"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/common"
	"time"
)

type SpecEdge struct{}

func (e *SpecEdge) Id() [32]byte {
	return [32]byte{}
}
func (e *SpecEdge) MiniStaker() (common.Address, error) {
	return common.Address{}, nil
}
func (e *SpecEdge) StartCommitment() (protocol.Height, common.Hash) {
	return 0, common.Hash{}
}
func (e *SpecEdge) TargetCommitment() (protocol.Height, common.Hash) {

	return 0, common.Hash{}
}
func (e *SpecEdge) PresumptiveTimer(ctx context.Context) (uint64, error) {
	return 0, nil
}
func (e *SpecEdge) IsPresumptive(ctx context.Context) (bool, error) {
	return false, nil
}
func (e *SpecEdge) Status(ctx context.Context) (protocol.EdgeStatus, error) {
	return protocol.EdgePending, nil
}
func (e *SpecEdge) DirectChildren(ctx context.Context) (util.Option[protocol.EdgeChildren], error) {
	return util.None[protocol.EdgeChildren](), nil

}
func (e *SpecEdge) Bisect(
	ctx context.Context,
	history util.HistoryCommitment,
	proof []byte,
) (protocol.SpecEdge, protocol.SpecEdge, error) {

	return nil, nil, nil
}
func (e *SpecEdge) CreateSubChallenge(ctx context.Context) (protocol.SpecChallenge, error) {
	return nil, nil
}
func (e *SpecEdge) ConfirmForTimer(ctx context.Context) error {
	return nil
}
func (e *SpecEdge) ConfirmForSubChallengeWin(ctx context.Context) error {
	return nil
}

type SpecChallenge struct{}

func (c *SpecChallenge) Id() protocol.ChallengeHash {
	return protocol.ChallengeHash{}
}
func (c *SpecChallenge) GetType() protocol.ChallengeType {
	return 0
}
func (c *SpecChallenge) StartTime() (uint64, error) {
	return 0, nil
}
func (c *SpecChallenge) RootCommitment() (protocol.Height, common.Hash, error) {
	return 0, common.Hash{}, nil
}
func (c *SpecChallenge) Status(ctx context.Context) (protocol.ChallengeStatus, error) {
	return 0, nil
}
func (c *SpecChallenge) RootAssertion(ctx context.Context) (protocol.Assertion, error) {
	return nil, nil
}
func (c *SpecChallenge) TopLevelClaimCommitment(ctx context.Context) (protocol.Height, common.Hash, error) {
	return 0, common.Hash{}, nil
}
func (c *SpecChallenge) WinningEdge(ctx context.Context) (util.Option[protocol.SpecEdge], error) {
	return util.None[protocol.SpecEdge](), nil
}
func (c *SpecChallenge) EdgeIsOneStepForkSource(
	ctx context.Context,
	edge protocol.SpecEdge,
) (bool, error) {
	return false, nil
}
func (c *SpecChallenge) AddBlockChallengeLevelZeroEdge(
	ctx context.Context,
	assertion protocol.Assertion,
	history util.HistoryCommitment,
) (protocol.SpecEdge, error) {
	return nil, nil
}
func (c *SpecChallenge) AddSubChallengeLevelZeroEdge(
	ctx context.Context,
	challengedEdge protocol.SpecEdge,
	history util.HistoryCommitment,
) (protocol.SpecEdge, error) {
	return nil, nil
}

// ChallengeManager --
type SpecChallengeManager struct {
	addr           common.Address
	assertionChain *AssertionChain
	caller         *mocksgen.SpecChallengeManagerCaller
	writer         *mocksgen.SpecChallengeManagerTransactor
	filterer       *mocksgen.SpecChallengeManagerFilterer
}

// CurrentChallengeManager returns an instance of the current challenge manager
// used by the assertion chain.
func NewSpecCM(ctx context.Context) (protocol.SpecChallengeManager, error) {
	managerBinding, err := mocksgen.NewSpecChallengeManager(common.Address{}, nil)
	if err != nil {
		return nil, err
	}
	return &SpecChallengeManager{
		addr:           common.Address{},
		assertionChain: &AssertionChain{},
		caller:         &managerBinding.SpecChallengeManagerCaller,
		writer:         &managerBinding.SpecChallengeManagerTransactor,
		filterer:       &managerBinding.SpecChallengeManagerFilterer,
	}, nil
}

func (cm *SpecChallengeManager) Address() common.Address {
	return cm.addr
}

// Duration of the challenge period.
func (cm *SpecChallengeManager) ChallengePeriodSeconds(
	ctx context.Context,
) (time.Duration, error) {
	return time.Second, nil
}

// Calculates the unique identifier for a challenge given an claim ID and a challenge type.
// An claim could be an assertion or a vertex that originated the challenge.
func (cm *SpecChallengeManager) CalculateChallengeHash(
	ctx context.Context,
	claimId common.Hash,
	challengeType protocol.ChallengeType,
) (protocol.ChallengeHash, error) {
	return protocol.ChallengeHash{}, nil
}

// Calculates an edge hash given its challenge id, start history, and end history.
func (cm *SpecChallengeManager) CalculateEdgeHash(
	ctx context.Context,
	challengeId protocol.ChallengeHash,
	startHistory util.HistoryCommitment,
	endHistory util.HistoryCommitment,
) (protocol.EdgeHash, error) {
	return protocol.EdgeHash{}, nil
}

// Gets an edge by its hash.
func (cm *SpecChallengeManager) GetEdge(
	ctx context.Context,
	edgeId protocol.EdgeHash,
) (util.Option[protocol.SpecEdge], error) {
	return util.None[protocol.SpecEdge](), nil
}

// Gets a challenge by its hash.
func (cm *SpecChallengeManager) GetChallenge(
	ctx context.Context, challengeId protocol.ChallengeHash,
) (util.Option[protocol.SpecChallenge], error) {
	return util.None[protocol.SpecChallenge](), nil
}
