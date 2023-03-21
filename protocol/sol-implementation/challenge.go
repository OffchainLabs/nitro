package solimpl

import (
	"context"
	"math/big"
	"time"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/challengeV2gen"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
)

func (c *Challenge) Id() protocol.ChallengeHash {
	return c.id
}

func (c *Challenge) Challenger() common.Address {
	return c.inner.Challenger
}

func (c *Challenge) RootAssertion(
	ctx context.Context, tx protocol.ActiveTx,
) (protocol.Assertion, error) {
	rootVertex, err := c.manager.GetVertex(ctx, tx, c.inner.RootId)
	if err != nil {
		return nil, err
	}
	if rootVertex.IsNone() {
		return nil, errors.New("root vertex not found")
	}
	root := rootVertex.Unwrap().(*ChallengeVertex)
	assertionNum, err := c.manager.assertionChain.GetAssertionNum(ctx, tx, root.inner.ClaimId)
	if err != nil {
		return nil, errors.Wrap(err, "HERE IN ROOT")
	}
	assertion, err := c.manager.assertionChain.AssertionBySequenceNum(ctx, tx, assertionNum)
	if err != nil {
		return nil, err
	}
	return assertion, nil
}

// TopLevelClaimVertex gets the vertex at the BlockChallenge level that originated a subchallenge.
// For example, if two validators open a subchallenge S at vertex A in a BlockChallenge, the TopLevelClaimVertex
// of S is A. If two validators open a subchallenge S' at vertex B in BigStepChallenge, the TopLevelClaimVertex
// is vertex A.
func (c *Challenge) TopLevelClaimVertex(ctx context.Context, tx protocol.ActiveTx) (protocol.ChallengeVertex, error) {
	if c.GetType() == protocol.BlockChallenge {
		return nil, errors.New("not a subchallenge")
	}
	rootV, err := c.manager.GetVertex(ctx, tx, c.inner.RootId)
	if err != nil {
		return nil, err
	}
	if rootV.IsNone() {
		return nil, errors.New("no root vertex for challenge found")
	}
	root := rootV.Unwrap().(*ChallengeVertex)
	claimVertexV, err := c.manager.GetVertex(ctx, tx, root.inner.ClaimId)
	if err != nil {
		return nil, err
	}
	if claimVertexV.IsNone() {
		return nil, errors.New("no root vertex for challenge found")
	}
	claimVertex := claimVertexV

	// If we are in a big step challenge, the claim vertex is the top-level vertex of the
	// corresponding BlockChallenge, so we are done.
	if c.GetType() == protocol.BigStepChallenge {
		return claimVertex.Unwrap(), nil
	}

	// Otherwise, a bit more work is required.
	// Get the root vertex of the BigStepChallenge claimVertex belongs to.
	bigStepChallengeId := claimVertex.Unwrap().(*ChallengeVertex).inner.ChallengeId
	bigStepC, err := c.manager.GetChallenge(ctx, tx, bigStepChallengeId)
	if err != nil {
		return nil, err
	}
	bigStepChallenge := bigStepC.Unwrap().(*Challenge)
	bigStepRootV, err := c.manager.GetVertex(ctx, tx, bigStepChallenge.inner.RootId)
	if err != nil {
		return nil, err
	}
	if bigStepRootV.IsNone() {
		return nil, errors.New("no root vertex for challenge found")
	}
	bigStepRoot := bigStepRootV.Unwrap().(*ChallengeVertex)

	// Get the claim vertex of the BigStepChallenge's root vertex.
	claimVertexV, err = c.manager.GetVertex(ctx, tx, bigStepRoot.inner.ClaimId)
	if err != nil {
		return nil, err
	}
	if claimVertexV.IsNone() {
		return nil, errors.New("no claim vertex for BigStepChallenge found")
	}
	return claimVertexV.Unwrap(), nil
}

func (c *Challenge) RootVertex(
	ctx context.Context, tx protocol.ActiveTx,
) (protocol.ChallengeVertex, error) {
	rootId := c.inner.RootId
	v, err := c.manager.GetVertex(ctx, tx, rootId)
	if err != nil {
		return nil, err
	}
	return v.Unwrap(), nil
}

func (c *Challenge) WinningClaim() util.Option[protocol.AssertionHash] {
	if c.inner.WinningClaim == [32]byte{} {
		return util.None[protocol.AssertionHash]()
	}
	return util.Some[protocol.AssertionHash](c.inner.WinningClaim)
}

func (c *Challenge) GetType() protocol.ChallengeType {
	return protocol.ChallengeType(c.inner.ChallengeType)
}

func (c *Challenge) GetCreationTime(
	ctx context.Context, tx protocol.ActiveTx,
) (time.Time, error) {
	return time.Time{}, errors.New("unimplemented")
}

func (c *Challenge) ParentStateCommitment(
	ctx context.Context, tx protocol.ActiveTx,
) (util.StateCommitment, error) {
	v, err := c.manager.GetVertex(ctx, tx, c.inner.RootId)
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
	assertionSeqNum, err := c.manager.assertionChain.rollup.GetAssertionNum(
		c.manager.assertionChain.callOpts, concreteV.inner.ClaimId,
	)
	if err != nil {
		return util.StateCommitment{}, err
	}
	assertion, err := c.manager.assertionChain.AssertionBySequenceNum(ctx, tx, protocol.AssertionSequenceNumber(assertionSeqNum))
	if err != nil {
		return util.StateCommitment{}, err
	}
	return util.StateCommitment{
		Height:    assertion.Height(),
		StateRoot: assertion.StateHash(),
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
	callOpts := c.manager.assertionChain.callOpts
	assertionId, err := c.manager.assertionChain.rollup.GetAssertionId(callOpts, uint64(assertion.SeqNum()))
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
	miniStake, err := c.manager.miniStakeAmount()
	if err != nil {
		return nil, err
	}
	opts := copyTxOpts(c.manager.assertionChain.txOpts)
	opts.Value = miniStake

	_, err = transact(ctx, c.manager.assertionChain.backend, c.manager.assertionChain.headerReader, func() (*types.Transaction, error) {
		return c.manager.writer.AddLeaf(
			opts,
			leafData,
			flatLastLeafProof,
			make([]byte, 0), // Inbox proof
		)
	})
	if err != nil {
		return nil, err
	}

	vertexId, err := c.manager.caller.CalculateChallengeVertexId(
		c.manager.assertionChain.callOpts,
		c.id,
		history.Merkle,
		big.NewInt(int64(history.Height)),
	)
	if err != nil {
		return nil, err
	}
	vertex, err := c.manager.caller.GetVertex(
		c.manager.assertionChain.callOpts,
		vertexId,
	)
	if err != nil {
		return nil, err
	}
	return &ChallengeVertex{
		id:      vertexId,
		inner:   vertex,
		manager: c.manager,
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
	miniStake, err := c.manager.miniStakeAmount()
	if err != nil {
		return nil, err
	}
	opts := copyTxOpts(c.manager.assertionChain.txOpts)
	opts.Value = miniStake

	_, err = transact(ctx, c.manager.assertionChain.backend, c.manager.assertionChain.headerReader, func() (*types.Transaction, error) {
		return c.manager.writer.AddLeaf(
			opts,
			leafData,
			flatLastLeafProof,
			flatLastLeafProof, // TODO(RJ): Should be different for big and small step.
		)
	})
	if err != nil {
		return nil, err
	}

	vertexId, err := c.manager.caller.CalculateChallengeVertexId(
		c.manager.assertionChain.callOpts,
		c.id,
		history.Merkle,
		big.NewInt(int64(history.Height)),
	)
	if err != nil {
		return nil, err
	}
	bsVertex, err := c.manager.caller.GetVertex(
		c.manager.assertionChain.callOpts,
		vertexId,
	)
	if err != nil {
		return nil, err
	}
	return &ChallengeVertex{
		id:      vertexId,
		inner:   bsVertex,
		manager: c.manager,
	}, nil
}
