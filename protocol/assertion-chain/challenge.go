package assertionchain

import (
	"math/big"

	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/outgen"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/common"
)

// Challenge is a developer-friendly wrapper around
// the protocol struct with the same name.
type Challenge struct {
	manager *ChallengeManager
	inner   outgen.Challenge
}

// ChallengeType defines an enum of the same name
// from the protocol.
type ChallengeType uint

const (
	BlockChallenge ChallengeType = iota
	BigStepChallenge
	SmallStepChallenge
	OneStepChallenge
)

// AddLeaf vertex to a BlockChallenge using an assertion and a history commitment.
func (c *Challenge) AddLeaf(
	assertion *Assertion,
	history util.HistoryCommitment,
	validator common.Address,
) (*ChallengeVertex, error) {
	assertionId := getAssertionId(assertion.StateCommitment, assertion.inner.PredecessorId)
	challengeId, err := c.manager.CalculateChallengeId(assertionId, BlockChallenge)
	if err != nil {
		return nil, err
	}

	// Flatten the last leaf proof for submission to the chain.
	lastLeafProof := make([]byte, 0)
	for _, h := range history.LastLeafProof {
		lastLeafProof = append(lastLeafProof, h[:]...)
	}
	leafData := outgen.AddLeafArgs{
		ChallengeId:            challengeId,
		ClaimId:                assertionId,
		Height:                 big.NewInt(int64(history.Height)),
		HistoryCommitment:      history.Merkle,
		FirstState:             history.FirstLeaf,
		FirstStatehistoryProof: make([]byte, 0), // TODO: Add in.
		LastState:              history.LastLeaf,
		LastStatehistoryProof:  lastLeafProof,
	}
	c.manager.assertionChain.txOpts.From = validator

	err := withChainCommitment(c.manager.assertionChain.backend, func() error {
		_, err := c.manager.writer.AddLeaf(
			c.manager.assertionChain.txOpts,
			leafData,
			make([]byte, 0), // TODO: Proof of inbox consumption.
			make([]byte, 0), // TODO: Proof of last state (redundant)
		)
		return err
	})
	if err != nil {
		return nil, err
	}
	vertexId, err := c.manager.caller.CalculateChallengeVertexId(
		c.manager.assertionChain.callOpts,
		challengeId,
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
