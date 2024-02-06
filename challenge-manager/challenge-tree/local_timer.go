// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

package challengetree

import (
	"fmt"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	"github.com/OffchainLabs/bold/containers/option"
	"github.com/OffchainLabs/bold/containers/threadsafe"
)

// Gets the local timer of an edge at a block number, T. If T is earlier than the edge's creation,
// this function will return 0.
func (ht *RoyalChallengeTree) LocalTimer(e protocol.ReadOnlyEdge, blockNum uint64) (uint64, error) {
	createdAtBlock, err := e.CreatedAtBlock()
	if err != nil {
		return 0, err
	}
	if blockNum <= createdAtBlock {
		return 0, nil
	}
	// If no rival at a block num, then the local timer is defined
	// as t - t_creation(e).
	unrivaled, err := ht.UnrivaledAtBlockNum(e, blockNum)
	if err != nil {
		return 0, err
	}
	if unrivaled {
		return blockNum - createdAtBlock, nil
	}
	// Else we return the earliest created rival's block number: t_rival - t_creation(e).
	// This unwrap is safe because the edge has rivals at this point due to the check above.
	earliest := ht.EarliestCreatedRivalBlockNumber(e)
	tRival := earliest.Unwrap()
	if createdAtBlock >= tRival {
		return 0, nil
	}
	return tRival - createdAtBlock, nil
}

// Gets the minimum creation block number across all of an edge's rivals. If an edge
// has no rivals, this minimum is undefined.
func (ht *RoyalChallengeTree) EarliestCreatedRivalBlockNumber(e protocol.ReadOnlyEdge) option.Option[uint64] {
	rivals := ht.rivalsWithCreationTimes(e)
	creationBlocks := make([]uint64, len(rivals))
	earliestCreatedRivalBlock := option.None[uint64]()
	for i, r := range rivals {
		creationBlocks[i] = uint64(r.createdAtBlock)
		if earliestCreatedRivalBlock.IsNone() {
			earliestCreatedRivalBlock = option.Some(uint64(r.createdAtBlock))
		} else if uint64(r.createdAtBlock) < earliestCreatedRivalBlock.Unwrap() {
			earliestCreatedRivalBlock = option.Some(uint64(r.createdAtBlock))
		}
	}
	return earliestCreatedRivalBlock
}

// Determines if an edge was unrivaled at a block num T. If any rival existed
// for the edge at T, this function will return false.
func (ht *RoyalChallengeTree) UnrivaledAtBlockNum(e protocol.ReadOnlyEdge, blockNum uint64) (bool, error) {
	createdAtBlock, err := e.CreatedAtBlock()
	if err != nil {
		return false, err
	}
	if blockNum < createdAtBlock {
		return false, fmt.Errorf(
			"edge creation block %d less than specified %d",
			createdAtBlock,
			blockNum,
		)
	}
	rivals := ht.rivalsWithCreationTimes(e)
	if len(rivals) == 0 {
		return true, nil
	}
	for _, r := range rivals {
		// If a rival existed before or at the time of the edge's
		// creation, we then return false.
		if uint64(r.createdAtBlock) <= blockNum {
			return false, nil
		}
	}
	return true, nil
}

// Contains a rival edge's id and its creation block number.
type rival struct {
	id             protocol.EdgeId
	createdAtBlock creationTime
}

// Computes the set of rivals with their creation block number for an edge being tracked
// by the challenge tree. We do this by computing the mutual id of the edge and fetching
// all edge ids that share the same one from a set the challenge tree keeps track of.
// We exclude the specified edge from the returned list of rivals.
func (ht *RoyalChallengeTree) rivalsWithCreationTimes(eg protocol.ReadOnlyEdge) []*rival {
	rivals := make([]*rival, 0)
	key := buildEdgeCreationTimeKey(eg.OriginId(), eg.MutualId())
	mutuals := ht.edgeCreationTimes.Get(key)
	if mutuals == nil {
		ht.edgeCreationTimes.Put(key, threadsafe.NewMap[protocol.EdgeId, creationTime]())
		return rivals
	}
	_ = mutuals.ForEach(func(rivalId protocol.EdgeId, t creationTime) error {
		if rivalId == eg.Id() {
			return nil
		}
		rivals = append(rivals, &rival{
			id:             rivalId,
			createdAtBlock: t,
		})
		return nil
	})
	return rivals
}
