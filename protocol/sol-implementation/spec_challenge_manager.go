package solimpl

import (
	"context"
	"time"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/challengeV2gen"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

type SpecEdge struct {
	id               [32]byte
	manager          *SpecChallengeManager
	startHeight      uint64
	startCommitment  common.Hash
	targetHeight     uint64
	targetCommitment common.Hash
	miniStaker       common.Address
}

func (e *SpecEdge) Id() [32]byte {
	return e.id
}

func (e *SpecEdge) MiniStaker() (common.Address, error) {
	return e.miniStaker, nil
}

func (e *SpecEdge) StartCommitment() (protocol.Height, common.Hash) {
	return protocol.Height(e.startHeight), e.startCommitment
}

func (e *SpecEdge) TargetCommitment() (protocol.Height, common.Hash) {
	return protocol.Height(e.targetHeight), e.targetCommitment
}

func (e *SpecEdge) PresumptiveTimer(ctx context.Context) (uint64, error) {
	timer, err := e.manager.caller.GetCurrentPsTimer(e.manager.callOpts, e.id)
	if err != nil {
		return 0, err
	}
	return timer.Uint64(), nil
}

func (e *SpecEdge) IsPresumptive(ctx context.Context) (bool, error) {
	return e.manager.caller.IsPresumptive(e.manager.callOpts, e.id)
}

func (e *SpecEdge) Status(ctx context.Context) (protocol.EdgeStatus, error) {
	edge, err := e.manager.caller.GetEdge(e.manager.callOpts, e.id)
	if err != nil {
		return 0, err
	}
	return protocol.EdgeStatus(edge.Status), nil
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

type SpecChallenge struct {
	manager *SpecChallengeManager
}

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
	addr     common.Address
	callOpts *bind.CallOpts
	caller   *challengeV2gen.EdgeChallengeManagerCaller
	writer   *challengeV2gen.EdgeChallengeManagerTransactor
	filterer *challengeV2gen.EdgeChallengeManagerFilterer
}

// CurrentChallengeManager returns an instance of the current challenge manager
// used by the assertion chain.
func NewSpecCM(ctx context.Context) (protocol.SpecChallengeManager, error) {
	managerBinding, err := challengeV2gen.NewEdgeChallengeManager(common.Address{}, nil)
	if err != nil {
		return nil, err
	}
	return &SpecChallengeManager{
		addr:     common.Address{},
		caller:   &managerBinding.EdgeChallengeManagerCaller,
		writer:   &managerBinding.EdgeChallengeManagerTransactor,
		filterer: &managerBinding.EdgeChallengeManagerFilterer,
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
	edge, err := cm.caller.GetEdge(cm.callOpts, edgeId)
	if err != nil {
		return util.None[protocol.SpecEdge](), err
	}
	return util.Some(&SpecEdge{
		id:               edge.ClaimEdgeId,
		manager:          cm,
		startHeight:      edge.StartHeight.Uint64(),
		targetHeight:     edge.EndHeight.Uint64(),
		startCommitment:  edge.StartHistoryRoot,
		targetCommitment: edge.EndHistoryRoot,
		miniStaker:       edge.Staker,
	}), nil
}

// Gets a challenge by its hash.
func (cm *SpecChallengeManager) GetChallenge(
	ctx context.Context, challengeId protocol.ChallengeHash,
) (util.Option[protocol.SpecChallenge], error) {
	return util.None[protocol.SpecChallenge](), nil
}
