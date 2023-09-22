package api_test

import (
	"github.com/OffchainLabs/bold/api"
	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	"github.com/OffchainLabs/bold/challenge-manager/challenge-tree/mock"
)

func edgesToMockEdges(e []*api.Edge) []*mock.Edge {
	me := make([]*mock.Edge, len(e))
	for i, ee := range e {
		me[i] = edgeToMockEdge(ee)
	}
	return me
}

func edgeToMockEdge(e *api.Edge) *mock.Edge {
	return &mock.Edge{
		ID: mock.EdgeId(e.ID.Bytes()),
		EdgeType: func() protocol.ChallengeLevel {
			et, err := protocol.ChallengeLevelFromString(e.Type)
			if err != nil {
				panic(err)
			}
			return et
		}(),
		StartHeight:          e.StartCommitment.Height,
		StartCommit:          mock.Commit(e.StartCommitment.Hash.Bytes()),
		EndHeight:            e.EndCommitment.Height,
		EndCommit:            mock.Commit(e.EndCommitment.Hash.Bytes()),
		OriginID:             mock.OriginId(e.OriginID.Bytes()),
		ClaimID:              string(e.ClaimID.Bytes()),
		LowerChildID:         mock.EdgeId(e.LowerChildID.Bytes()),
		UpperChildID:         mock.EdgeId(e.UpperChildID.Bytes()),
		CreationBlock:        e.CreatedAtBlock,
		TotalChallengeLevels: 3,
	}
}
