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

func (c *Challenge) Challenger(ctx context.Context, tx protocol.ActiveTx) common.Address {
	return c.inner(ctx, tx).Challenger
}

func (c *Challenge) RootAssertion(
	ctx context.Context, tx protocol.ActiveTx,
) (protocol.Assertion, error) {
	rootVertex, err := c.manager(ctx, tx).GetVertex(ctx, tx, c.inner(ctx, tx).RootId)
	if err != nil {
		return nil, err
	}
	if rootVertex.IsNone() {
		return nil, errors.New("root vertex not found")
	}
	root := rootVertex.Unwrap().(*ChallengeVertex)
	assertionNum, err := c.chain.GetAssertionNum(ctx, tx, root.inner(ctx, tx).ClaimId)
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
	rootId := c.inner(ctx, tx).RootId
	v, err := c.manager(ctx, tx).GetVertex(ctx, tx, rootId)
	if err != nil {
		return nil, err
	}
	return v.Unwrap(), nil
}

func (c *Challenge) WinningClaim(ctx context.Context, tx protocol.ActiveTx) util.Option[protocol.AssertionHash] {
	if c.inner(ctx, tx).WinningClaim == [32]byte{} {
		return util.None[protocol.AssertionHash]()
	}
	return util.Some[protocol.AssertionHash](c.inner(ctx, tx).WinningClaim)
}

func (c *Challenge) GetType(ctx context.Context, tx protocol.ActiveTx) protocol.ChallengeType {
	return protocol.ChallengeType(c.inner(ctx, tx).ChallengeType)
}

func (c *Challenge) GetCreationTime(
	ctx context.Context, tx protocol.ActiveTx,
) (time.Time, error) {
	return time.Time{}, errors.New("unimplemented")
}

func (c *Challenge) ParentStateCommitment(
	ctx context.Context, tx protocol.ActiveTx,
) (util.StateCommitment, error) {
	v, err := c.manager(ctx, tx).GetVertex(ctx, tx, c.inner(ctx, tx).RootId)
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
	assertionSeqNum, err := c.chain.rollup.GetAssertionNum(
		c.chain.callOpts, concreteV.inner(ctx, tx).ClaimId,
	)
	if err != nil {
		return util.StateCommitment{}, err
	}
	assertion, err := c.chain.AssertionBySequenceNum(ctx, tx, protocol.AssertionSequenceNumber(assertionSeqNum))
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
	miniStake, err := c.manager(ctx, tx).miniStakeAmount()
	if err != nil {
		return nil, err
	}
	opts := copyTxOpts(c.chain.txOpts)
	opts.Value = miniStake

	_, err = transact(ctx, c.chain.backend, c.chain.headerReader, func() (*types.Transaction, error) {
		return c.manager(ctx, tx).writer.AddLeaf(
			opts,
			leafData,
			flatLastLeafProof,
			make([]byte, 0), // Inbox proof
		)
	})
	if err != nil {
		return nil, err
	}

	vertexId, err := c.manager(ctx, tx).caller.CalculateChallengeVertexId(
		c.chain.callOpts,
		c.id,
		history.Merkle,
		big.NewInt(int64(history.Height)),
	)
	if err != nil {
		return nil, err
	}
	_, err = c.manager(ctx, tx).caller.GetVertex(
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
	miniStake, err := c.manager(ctx, tx).miniStakeAmount()
	if err != nil {
		return nil, err
	}
	opts := copyTxOpts(c.chain.txOpts)
	opts.Value = miniStake

	_, err = transact(ctx, c.chain.backend, c.chain.headerReader, func() (*types.Transaction, error) {
		return c.manager(ctx, tx).writer.AddLeaf(
			opts,
			leafData,
			flatLastLeafProof,
			flatLastLeafProof, // TODO(RJ): Should be different for big and small step.
		)
	})
	if err != nil {
		return nil, err
	}

	vertexId, err := c.manager(ctx, tx).caller.CalculateChallengeVertexId(
		c.chain.callOpts,
		c.id,
		history.Merkle,
		big.NewInt(int64(history.Height)),
	)
	if err != nil {
		return nil, err
	}
	_, err = c.manager(ctx, tx).caller.GetVertex(
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

func (c *Challenge) inner(ctx context.Context, tx protocol.ActiveTx) *challengeV2gen.Challenge {
	return nil
}

func (c *Challenge) manager(ctx context.Context, tx protocol.ActiveTx) *ChallengeManager {
	return nil
}
