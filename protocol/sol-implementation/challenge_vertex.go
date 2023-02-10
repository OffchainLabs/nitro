package solimpl

import (
	"context"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

// HasConfirmedSibling checks if the vertex has a confirmed sibling in the protocol.
func (v *ChallengeVertex) HasConfirmedSibling(ctx context.Context) (bool, error) {
	return v.manager.caller.HasConfirmedSibling(v.manager.assertionChain.callOpts, v.id)
}

// IsPresumptiveSuccessor checks if a vertex is the presumptive successor
// within its challenge.
func (v *ChallengeVertex) IsPresumptiveSuccessor(ctx context.Context) (bool, error) {
	return v.manager.caller.IsPresumptiveSuccessor(v.manager.assertionChain.callOpts, v.id)
}

// IsAtOneStepFork checks if a vertex is at a one-step-fork in the challenge
// it is contained in.
func (v *ChallengeVertex) IsAtOneStepFork(ctx context.Context) (bool, error) {
	return v.manager.caller.IsAtOneStepFork(v.manager.assertionChain.callOpts, v.id)
}

// Bisect a challenge vertex by providing a history commitment.
func (v *ChallengeVertex) Bisect(
	history util.HistoryCommitment,
	proof []common.Hash,
) (*ChallengeVertex, error) {
	// Refresh the inner fields of our before making on-chain calls.
	if err := v.invalidate(); err != nil {
		return nil, err
	}

	// Flatten the last leaf proof for submission to the chain.
	flatProof := make([]byte, 0)
	for _, h := range proof {
		flatProof = append(flatProof, h[:]...)
	}
	if err2 := withChainCommitment(v.manager.assertionChain.backend, func() error {
		_, err3 := v.manager.writer.Bisect(
			v.manager.assertionChain.txOpts,
			v.id,
			history.Merkle,
			flatProof,
		)
		return err3
	}); err2 != nil {
		return nil, err2
	}
	bisectedToId, err := v.manager.caller.CalculateChallengeVertexId(
		v.manager.assertionChain.callOpts,
		v.inner.ChallengeId,
		history.Merkle,
		big.NewInt(int64(history.Height)),
	)
	if err != nil {
		return nil, err
	}
	bisectedTo, err := v.manager.caller.GetVertex(
		v.manager.assertionChain.callOpts,
		bisectedToId,
	)
	if err != nil {
		return nil, err
	}
	return &ChallengeVertex{
		inner:   bisectedTo,
		manager: v.manager,
	}, nil
}
