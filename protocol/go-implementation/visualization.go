package goimpl

import (
	"context"
	"fmt"

	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/emicklei/dot"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// Visualize returns a graphviz string for the current assertion chain tree.
type Visualization struct {
	AssertionChain string                    `json:"assertion_chain"`
	Challenges     []*ChallengeVisualization `json:"challenges"`
}

func (chain *AssertionChain) Visualize(ctx context.Context, tx *ActiveTx) *Visualization {
	return &Visualization{
		AssertionChain: chain.visualizeAssertionChain(),
		Challenges:     chain.visualizeChallenges(ctx, tx),
	}
}

type vizNode struct {
	parent    util.Option[*Assertion]
	assertion *Assertion
	dotNode   dot.Node
}

func (chain *AssertionChain) visualizeAssertionChain() string {
	graph := dot.NewGraph(dot.Directed)
	graph.Attr("rankdir", "RL")
	graph.Attr("labeljust", "l")

	assertions := chain.assertions
	// Construct nodes
	m := make(map[[32]byte]*vizNode)
	for i := 0; i < len(assertions); i++ {
		a := assertions[i]
		commit := a.StateCommitment
		commitHash := commit.Hash()
		// Construct label of each node.
		rStr := hexutil.Encode(commit.Hash().Bytes())
		staker := common.Address{}
		if !a.Staker.IsNone() {
			staker = a.Staker.Unwrap()
		}
		label := fmt.Sprintf(
			"height: %d\n commitment: %#x\n staker: %x",
			commit.Height,
			commitHash[:4],
			staker[len(staker)-1:],
		)

		dotN := graph.Node(rStr).Box().Attr("label", label)
		m[commit.Hash()] = &vizNode{
			parent:    a.Prev,
			assertion: a,
			dotNode:   dotN,
		}
	}

	// Construct an edge only if block's parent exist in the tree.
	for _, n := range m {
		if !n.parent.IsNone() {
			parentHash := n.parent.Unwrap().StateCommitment.Hash()
			if _, ok := m[parentHash]; ok {
				graph.Edge(n.dotNode, m[parentHash].dotNode)
			}
		}
	}
	return graph.String()
}

type challengeVertexNode struct {
	parent  util.Option[ChallengeVertexInterface]
	vertex  *ChallengeVertex
	dotNode dot.Node
}

type ChallengeVisualization struct {
	RootAssertionCommit util.StateCommitment `json:"root_assertion_commit"`
	Graph               string               `json:"graph"`
}

func (chain *AssertionChain) visualizeChallenges(ctx context.Context, tx *ActiveTx) []*ChallengeVisualization {
	res := make([]*ChallengeVisualization, 0, len(chain.challengeVerticesByCommitHash))
	for cHash, challenge := range chain.challengesByCommitHash {
		// Ignore challenges with no root assertion or completed status.
		if challenge.(*Challenge).rootAssertion.IsNone() {
			continue
		}
		completed, _ := challenge.Completed(ctx, tx)
		if completed {
			continue
		}

		graph := dot.NewGraph(dot.Directed)
		graph.Attr("rankdir", "RL")
		graph.Attr("labeljust", "l")

		// Construct nodes.
		m := make(map[[32]byte]*challengeVertexNode)
		vertices := chain.challengeVerticesByCommitHash[cHash]

		childCount := make(map[VertexCommitHash]uint64)
		for _, v := range vertices {
			commit := v.(*ChallengeVertex).Commitment
			// Construct label of each node.
			rStr := hexutil.Encode(commit.Hash().Bytes())
			commitHash := commit.Hash()
			label := fmt.Sprintf(
				"height: %d\n merkle: %#x\n staker: %x",
				commit.Height,
				commitHash[:4],
				v.(*ChallengeVertex).Validator[len(v.(*ChallengeVertex).Validator)-1:],
			)

			if !v.(*ChallengeVertex).Prev.IsNone() {
				prevCommitment, _ := v.(*ChallengeVertex).Prev.Unwrap().GetCommitment(ctx, tx)
				childCount[VertexCommitHash(prevCommitment.Hash())]++
			}

			dotN := graph.Node(rStr).Box().Attr("label", label)

			prev, _ := v.GetPrev(ctx, tx)
			m[commit.Hash()] = &challengeVertexNode{
				parent:  prev,
				vertex:  v.(*ChallengeVertex),
				dotNode: dotN,
			}
		}

		// Construct an edge only if block's parent exist in the tree.
		for _, n := range m {
			if !n.parent.IsNone() {
				parentCommitment, _ := n.parent.Unwrap().GetCommitment(ctx, tx)
				parentHash := parentCommitment.Hash()
				if _, ok := m[parentHash]; ok {

					vertexIsPresumptiveSuccessor, _ := n.vertex.IsPresumptiveSuccessor(ctx, tx)
					if childCount[VertexCommitHash(parentHash)] > 1 && vertexIsPresumptiveSuccessor {
						graph.Edge(n.dotNode, m[parentHash].dotNode).
							Bold().
							Label("ps").
							Attr("color", "red")
					} else {
						graph.Edge(n.dotNode, m[parentHash].dotNode)
					}
				}
			}
		}
		var rootAssertionCommit util.StateCommitment
		if !challenge.(*Challenge).rootAssertion.IsNone() {
			rootAssertionCommit = challenge.(*Challenge).rootAssertion.Unwrap().StateCommitment
		}
		res = append(res, &ChallengeVisualization{
			RootAssertionCommit: rootAssertionCommit,
			Graph:               graph.String(),
		})
	}
	return res
}
