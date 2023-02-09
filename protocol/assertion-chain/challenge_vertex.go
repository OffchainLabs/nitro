package assertionchain

import (
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

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

	vertexId, err := v.manager.caller.CalculateChallengeVertexId(
		v.manager.assertionChain.callOpts,
		v.id,
		history.Merkle,
		big.NewInt(int64(history.Height)),
	)
	if err != nil {
		return nil, err
	}
	vertex, err := v.manager.caller.GetVertex(
		v.manager.assertionChain.callOpts,
		vertexId,
	)
	if err != nil {
		return nil, err
	}
	return &ChallengeVertex{
		inner:   vertex,
		manager: v.manager,
	}, nil
}
