package solimpl

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/challengeV2gen"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func (c *Challenge) Id() protocol.ChallengeHash {
	return c.id
}

func (c *Challenge) Challenger(ctx context.Context, tx protocol.ActiveTx) (common.Address, error) {
	inner, err := c.inner(ctx, tx)
	if err != nil {
		return common.Address{}, err
	}
	return inner.Challenger, nil
}

func (c *Challenge) RootAssertion(
	ctx context.Context, tx protocol.ActiveTx,
) (protocol.Assertion, error) {
	cManager, err := c.manager(ctx, tx)
	if err != nil {
		return nil, err
	}
	cInner, err := c.inner(ctx, tx)
	if err != nil {
		return nil, err
	}
	rootVertex, err := cManager.GetVertex(ctx, tx, cInner.RootId)
	if err != nil {
		return nil, err
	}
	if rootVertex.IsNone() {
		return nil, errors.New("root vertex not found")
	}
	root := rootVertex.Unwrap().(*ChallengeVertex)
	rootInner, err := root.inner(ctx, tx)
	if err != nil {
		return nil, err
	}
	assertionNum, err := c.chain.GetAssertionNum(ctx, tx, rootInner.ClaimId)
	if err != nil {
		return nil, err
	}
	assertion, err := c.chain.AssertionBySequenceNum(ctx, tx, assertionNum)
	if err != nil {
		return nil, err
	}
	return assertion, nil
}

func (c *Challenge) RootVertex(
	ctx context.Context, tx protocol.ActiveTx,
) (protocol.ChallengeVertex, error) {
	cInner, err := c.inner(ctx, tx)
	if err != nil {
		return nil, err
	}
	cManager, err := c.manager(ctx, tx)
	if err != nil {
		return nil, err
	}
	rootId := cInner.RootId
	v, err := cManager.GetVertex(ctx, tx, rootId)
	if err != nil {
		return nil, err
	}
	return v.Unwrap(), nil
}

func (c *Challenge) WinningClaim(ctx context.Context, tx protocol.ActiveTx) (util.Option[protocol.AssertionHash], error) {
	cInner, err := c.inner(ctx, tx)
	if err != nil {
		return util.None[protocol.AssertionHash](), err
	}
	if cInner.WinningClaim == [32]byte{} {
		return util.None[protocol.AssertionHash](), nil
	}
	return util.Some[protocol.AssertionHash](cInner.WinningClaim), nil
}

func (c *Challenge) GetType(ctx context.Context, tx protocol.ActiveTx) (protocol.ChallengeType, error) {
	cInner, err := c.inner(ctx, tx)
	if err != nil {
		return 0, err
	}
	return protocol.ChallengeType(cInner.ChallengeType), nil
}

func (c *Challenge) GetCreationTime(
	ctx context.Context, tx protocol.ActiveTx,
) (time.Time, error) {
	return time.Time{}, errors.New("unimplemented")
}

func (c *Challenge) ParentStateCommitment(
	ctx context.Context, tx protocol.ActiveTx,
) (util.StateCommitment, error) {
	cManager, err := c.manager(ctx, tx)
	if err != nil {
		return util.StateCommitment{}, err
	}
	cInner, err := c.inner(ctx, tx)
	if err != nil {
		return util.StateCommitment{}, err
	}
	v, err := cManager.GetVertex(ctx, tx, cInner.RootId)
	if err != nil {
		return util.StateCommitment{}, err
	}
	if v.IsNone() {
		return util.StateCommitment{}, errors.New("no root vertex for challenge")
	}
	concreteV, ok := v.Unwrap().(*ChallengeVertex)
	if !ok {
		return util.StateCommitment{}, errors.New("vertex is not expected concrete type")
	}
	concreteVInner, err := concreteV.inner(ctx, tx)
	if err != nil {
		return util.StateCommitment{}, err
	}
	assertionSeqNum, err := c.chain.rollup.GetAssertionNum(
		c.chain.callOpts, concreteVInner.ClaimId,
	)
	if err != nil {
		return util.StateCommitment{}, err
	}
	assertion, err := c.chain.AssertionBySequenceNum(ctx, tx, protocol.AssertionSequenceNumber(assertionSeqNum))
	if err != nil {
		return util.StateCommitment{}, err
	}
	height, err := assertion.Height()
	if err != nil {
		return util.StateCommitment{}, err
	}
	stateHash, err := assertion.StateHash()
	if err != nil {
		return util.StateCommitment{}, err
	}
	return util.StateCommitment{
		Height:    height,
		StateRoot: stateHash,
	}, nil
}

func (c *Challenge) WinnerVertex(
	ctx context.Context, tx protocol.ActiveTx,
) (util.Option[protocol.ChallengeVertex], error) {
	return util.None[protocol.ChallengeVertex](), errors.New("unimplemented")
}

func (c *Challenge) Completed(
	ctx context.Context, tx protocol.ActiveTx,
) (bool, error) {
	return false, errors.New("unimplemented")
}

// AddBlockChallengeLeaf vertex to a BlockChallenge using an assertion and a history commitment.
func (c *Challenge) AddBlockChallengeLeaf(
	ctx context.Context,
	tx protocol.ActiveTx,
	assertion protocol.Assertion,
	history util.HistoryCommitment,
) (protocol.ChallengeVertex, error) {
	// Flatten the last leaf proof for submission to the chain.
	flatLastLeafProof := make([]byte, 0, len(history.LastLeafProof)*32)
	lastLeafProof := make([][32]byte, len(history.LastLeafProof))
	for i, h := range history.LastLeafProof {
		var r [32]byte
		copy(r[:], h[:])
		flatLastLeafProof = append(flatLastLeafProof, r[:]...)
		lastLeafProof[i] = r
	}
	firstLeafProof := make([][32]byte, len(history.FirstLeafProof))
	for i, h := range history.FirstLeafProof {
		var r [32]byte
		copy(r[:], h[:])
		firstLeafProof[i] = r
	}
	callOpts := c.chain.callOpts
	assertionId, err := c.chain.rollup.GetAssertionId(callOpts, uint64(assertion.SeqNum()))
	if err != nil {
		return nil, err
	}
	leafData := challengeV2gen.AddLeafArgs{
		ChallengeId:            c.id,
		ClaimId:                assertionId,
		Height:                 big.NewInt(int64(history.Height)),
		HistoryRoot:            history.Merkle,
		FirstState:             history.FirstLeaf,
		FirstStatehistoryProof: firstLeafProof,
		LastState:              history.LastLeaf,
		LastStatehistoryProof:  lastLeafProof,
	}

	// Check the current mini-stake amount and transact using that as the value.
	cManager, err := c.manager(ctx, tx)
	if err != nil {
		return nil, err
	}
	miniStake, err := cManager.miniStakeAmount()
	if err != nil {
		return nil, err
	}
	opts := copyTxOpts(c.chain.txOpts)
	opts.Value = miniStake

	_, err = transact(ctx, c.chain.backend, c.chain.headerReader, func() (*types.Transaction, error) {
		return cManager.writer.AddLeaf(
			opts,
			leafData,
			flatLastLeafProof,
			make([]byte, 0), // Inbox proof
		)
	})
	if err != nil {
		return nil, err
	}

	vertexId, err := cManager.caller.CalculateChallengeVertexId(
		c.chain.callOpts,
		c.id,
		history.Merkle,
		big.NewInt(int64(history.Height)),
	)
	if err != nil {
		return nil, err
	}
	_, err = cManager.caller.GetVertex(
		c.chain.callOpts,
		vertexId,
	)
	if err != nil {
		return nil, err
	}
	return &ChallengeVertex{
		id:    vertexId,
		chain: c.chain,
	}, nil
}

// AddSubChallengeLeaf adds the appropriate leaf to the challenge based on a vertex and history commitment.
func (c *Challenge) AddSubChallengeLeaf(
	ctx context.Context,
	tx protocol.ActiveTx,
	vertex protocol.ChallengeVertex,
	history util.HistoryCommitment,
) (protocol.ChallengeVertex, error) {
	// Flatten the last leaf proof for submission to the chain.
	flatLastLeafProof := make([]byte, 0, len(history.LastLeafProof)*32)
	lastLeafProof := make([][32]byte, len(history.LastLeafProof))
	for i, h := range history.LastLeafProof {
		var r [32]byte
		copy(r[:], h[:])
		flatLastLeafProof = append(flatLastLeafProof, r[:]...)
		lastLeafProof[i] = r
	}

	firstLeafProof := make([][32]byte, len(history.FirstLeafProof))
	for i, h := range history.FirstLeafProof {
		var r [32]byte
		copy(r[:], h[:])
		firstLeafProof[i] = r
	}
	leafData := challengeV2gen.AddLeafArgs{
		ChallengeId:            c.id,
		ClaimId:                vertex.Id(),
		Height:                 big.NewInt(int64(history.Height)),
		HistoryRoot:            history.Merkle,
		FirstState:             history.FirstLeaf,
		FirstStatehistoryProof: firstLeafProof,
		LastState:              history.LastLeaf,
		LastStatehistoryProof:  lastLeafProof,
	}

	// Check the current mini-stake amount and transact using that as the value.
	cManager, err := c.manager(ctx, tx)
	if err != nil {
		return nil, err
	}
	miniStake, err := cManager.miniStakeAmount()
	if err != nil {
		return nil, err
	}
	opts := copyTxOpts(c.chain.txOpts)
	opts.Value = miniStake

	_, err = transact(ctx, c.chain.backend, c.chain.headerReader, func() (*types.Transaction, error) {
		return cManager.writer.AddLeaf(
			opts,
			leafData,
			flatLastLeafProof,
			flatLastLeafProof, // TODO(RJ): Should be different for big and small step.
		)
	})
	if err != nil {
		return nil, err
	}

	vertexId, err := cManager.caller.CalculateChallengeVertexId(
		c.chain.callOpts,
		c.id,
		history.Merkle,
		big.NewInt(int64(history.Height)),
	)
	if err != nil {
		return nil, err
	}
	_, err = cManager.caller.GetVertex(
		c.chain.callOpts,
		vertexId,
	)
	if err != nil {
		return nil, err
	}
	return &ChallengeVertex{
		id:    vertexId,
		chain: c.chain,
	}, nil
}

func (c *Challenge) inner(ctx context.Context, tx protocol.ActiveTx) (challengeV2gen.Challenge, error) {
	manager, err := c.manager(ctx, tx)
	if err != nil {
		return challengeV2gen.Challenge{}, err
	}

	challengeInner, err := manager.caller.GetChallenge(c.chain.callOpts, c.id)
	if err != nil {
		return challengeV2gen.Challenge{}, err
	}
	return challengeInner, nil
}

func (c *Challenge) manager(ctx context.Context, tx protocol.ActiveTx) (*ChallengeManager, error) {
	manager, err := c.chain.CurrentChallengeManager(ctx, tx)
	if err != nil {
		return nil, err
	}
	challengeManager, ok := manager.(*ChallengeManager)
	if !ok {
		return nil, errors.New("challengemanager is not expected concrete type")
	}
	return challengeManager, nil
}
