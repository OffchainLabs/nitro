package assertionchain

import (
	"math/big"

	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/outgen"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
)

// AddLeaf vertex to a BlockChallenge using an assertion and a history commitment.
func (c *Challenge) AddLeaf(
	assertion *Assertion,
	history util.HistoryCommitment,
) (*ChallengeVertex, error) {
	// Refresh the inner fields of our before making on-chain calls.
	if err := assertion.invalidate(); err != nil {
		return nil, err
	}
	if err := c.invalidate(); err != nil {
		return nil, err
	}

	// Flatten the last leaf proof for submission to the chain.
	lastLeafProof := make([]byte, 0)
	for _, h := range history.LastLeafProof {
		lastLeafProof = append(lastLeafProof, h[:]...)
	}
	leafData := outgen.AddLeafArgs{
		ChallengeId:            c.id,
		ClaimId:                assertion.id,
		Height:                 big.NewInt(int64(history.Height)),
		HistoryCommitment:      history.Merkle,
		FirstState:             history.FirstLeaf,
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

	if err2 := withChainCommitment(c.manager.assertionChain.backend, func() error {
		_, err3 := c.manager.writer.AddLeaf(
			opts,
			leafData,
			make([]byte, 0), // TODO: Proof of inbox consumption.
			make([]byte, 0), // TODO: Proof of last state (redundant)
		)
		return err3
	}); err2 != nil {
		return nil, err2
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
		inner:   vertex,
		manager: c.manager,
	}, nil
}
