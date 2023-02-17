package solimpl

import (
	"math/big"

	"context"
	"errors"
	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/challengeV2gen"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/core/types"
	"time"
)

func (c *Challenge) RootAssertion(
	ctx context.Context, tx protocol.ActiveTx,
) (protocol.Assertion, error) {
	return nil, errors.New("unimplemented")
}

func (c *Challenge) RootVertex(
	ctx context.Context, tx protocol.ActiveTx,
) (protocol.ChallengeVertex, error) {
	return nil, errors.New("unimplemented")
}

func (c *Challenge) GetType(
	ctx context.Context, tx protocol.ActiveTx,
) (protocol.ChallengeType, error) {
	return nil, errors.New("unimplemented")
}

func (c *Challenge) GetCreationTime(
	ctx context.Context, tx protocol.ActiveTx,
) (time.Time, error) {
	return time.Time{}, errors.New("unimplemented")
}

func (c *Challenge) ParentStateCommitment(
	ctx context.Context, tx protocol.ActiveTx,
) (util.StateCommitment, error) {
	return util.StateCommitment{}, errors.New("unimplemented")
}

func (c *Challenge) WinnerVertex(
	ctx context.Context, tx protocol.ActiveTx,
) (util.Option[protocol.ChallengeVertex], error) {
	return util.None[protocol.ChallengeVertex](), errors.New("unimplemented")
}

// AddLeaf vertex to a BlockChallenge using an assertion and a history commitment.
func (c *Challenge) AddLeaf(
	ctx context.Context,
	tx protocol.ActiveTx,
	assertion protocol.Assertion,
	history util.HistoryCommitment,
) (protocol.ChallengeVertex, error) {
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

func (c *Challenge) AddSubchallengeLeaf(
	ctx context.Context,
	tx protocol.ActiveTx,
	vertex protocol.ChallengeVertex,
	history util.HistoryCommitment,
) (protocol.ChallengeVertex, error) {
	return nil, errors.New("unimplemented")
}
