package solimpl

import (
	"context"
	"math/big"
	"strings"

	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
)

// Bisect a challenge vertex by providing a history commitment.
func (v *ChallengeVertex) Bisect(
	ctx context.Context,
	history util.HistoryCommitment,
	proof []common.Hash,
) (*ChallengeVertex, error) {
	// Flatten the last leaf proof for submission to the chain.
	flatProof := make([]byte, 0)
	for _, h := range proof {
		flatProof = append(flatProof, h[:]...)
	}
	_, err := transact(ctx, v.manager.assertionChain.backend, func() (*types.Transaction, error) {
		return v.manager.writer.Bisect(
			v.manager.assertionChain.txOpts,
			v.id,
			history.Merkle,
			flatProof,
		)
	})
	if err != nil {
		return nil, err
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
		id:      bisectedToId,
		inner:   bisectedTo,
		manager: v.manager,
	}, nil
}

func (v *ChallengeVertex) ConfirmPsTimer(ctx context.Context) error {
	_, err := transact(ctx, v.manager.assertionChain.backend, func() (*types.Transaction, error) {
		return v.manager.writer.ConfirmForPsTimer(
			v.manager.assertionChain.txOpts,
			v.id,
		)
	})
	if err == nil {
		return nil
	}
	switch {
	case strings.Contains(err.Error(), "PsTimer not greater than challenge period"):
		return errors.Wrapf(ErrPsTimerNotYet, "vertex id %#v", v.id)
	default:
		return err
	}
}
