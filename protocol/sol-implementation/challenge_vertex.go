package solimpl

import (
	"math/big"

	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

// Merge a challenge vertex to another by providing its history
// commitment and a prefix proof.
func (v *ChallengeVertex) Merge(
	mergingToHistory util.HistoryCommitment,
	proof []common.Hash,
) error {
	// Refresh the inner fields of vertices before making calls.
	if err := v.invalidate(); err != nil {
		return err
	}
	// Flatten the last leaf proof for submission to the chain.
	flatProof := make([]byte, 0)
	for _, h := range proof {
		flatProof = append(flatProof, h[:]...)
	}
	if err := withChainCommitment(v.manager.assertionChain.backend, func() error {
		_, mergeErr := v.manager.writer.Merge(
			v.manager.assertionChain.txOpts,
			v.id,
			mergingToHistory.Merkle,
			flatProof,
		)
		return mergeErr
	}); err != nil {
		return err
	}
	return getBisectedToVertex(
		v.manager,
		v.manager.assertionChain.callOpts,
		v.inner.ChallengeId,
		mergingToHistory,
	)
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
	return getBisectedToVertex(
		v.manager,
		v.manager.assertionChain.callOpts,
		v.inner.ChallengeId,
		history,
	)
}

func getBisectedToVertex(
	manager *ChallengeManager,
	opts *bind.CallOpts,
	challengeId [32]byte,
	history util.HistoryCommitment,
) (*ChallengeVertex, error) {
	bisectedToId, err := manager.caller.CalculateChallengeVertexId(
		opts,
		challengeId,
		history.Merkle,
		big.NewInt(int64(history.Height)),
	)
	if err != nil {
		return nil, err
	}
	bisectedTo, err := manager.caller.GetVertex(
		opts,
		bisectedToId,
	)
	if err != nil {
		return nil, err
	}
	return &ChallengeVertex{
		id:      bisectedToId,
		inner:   bisectedTo,
		manager: manager,
	}, nil
}
