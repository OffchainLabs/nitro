package solimpl

import (
	"context"
	"math/big"

	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/challengeV2gen"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/core/types"
)

// AddBlockChallengeLeaf vertex to a BlockChallenge using an assertion and a history commitment.
func (c *Challenge) AddBlockChallengeLeaf(
	ctx context.Context,
	assertion *Assertion,
	history util.HistoryCommitment,
) (*ChallengeVertex, error) {
	// Flatten the last leaf proof for submission to the chain.
	lastLeafProof := make([]byte, 0)
	for _, h := range history.LastLeafProof {
		lastLeafProof = append(lastLeafProof, h[:]...)
	}
	callOpts := c.manager.assertionChain.callOpts
	assertionId, err := c.manager.assertionChain.rollup.GetAssertionId(callOpts, assertion.id)
	if err != nil {
		return nil, err
	}
	prevAssertion, err := c.manager.assertionChain.AssertionByID(assertion.inner.PrevNum)
	if err != nil {
		return nil, err
	}
	leafData := challengeV2gen.AddLeafArgs{
		ChallengeId:            c.id,
		ClaimId:                assertionId,
		Height:                 big.NewInt(int64(history.Height)),
		HistoryRoot:            history.Merkle,
		FirstState:             prevAssertion.inner.StateHash,
		FirstStatehistoryProof: make([]byte, 0), // TODO: Add in.
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

	_, err = transact(ctx, c.manager.assertionChain.backend, func() (*types.Transaction, error) {
		return c.manager.writer.AddLeaf(
			opts,
			leafData,
			make([]byte, 0), // TODO: Proof of inbox consumption.
			make([]byte, 0), // TODO: Proof of last state (redundant)
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

// AddBigStepChallengeLeaf vertex to a BigStepChallenge using a vertex and a history commitment.
func (c *Challenge) AddBigStepChallengeLeaf(
	ctx context.Context,
	vertex *ChallengeVertex,
	history util.HistoryCommitment,
) (*ChallengeVertex, error) {
	// Flatten the last leaf proof for submission to the chain.
	lastLeafProof := make([]byte, 0)
	for _, h := range history.LastLeafProof {
		lastLeafProof = append(lastLeafProof, h[:]...)
	}

	leafData := challengeV2gen.AddLeafArgs{
		ChallengeId:            c.id,
		ClaimId:                vertex.id,
		Height:                 big.NewInt(int64(history.Height)),
		HistoryRoot:            history.Merkle,
		FirstState:             [32]byte{},      // TODO: What to do with this
		FirstStatehistoryProof: make([]byte, 0), // TODO: Add in.
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

	_, err = transact(ctx, c.manager.assertionChain.backend, func() (*types.Transaction, error) {
		return c.manager.writer.AddLeaf(
			opts,
			leafData,
			make([]byte, 0), // TODO: Proof of inbox consumption.
			make([]byte, 0), // TODO: Proof of last state (redundant)
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
