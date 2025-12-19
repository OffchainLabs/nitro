// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package l2pricing

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"

	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/storage"
)

func TestMultiGasConstraint(t *testing.T) {
	sto := storage.NewMemoryBacked(burn.NewSystemBurner(nil, false))
	c := OpenMultiGasConstraint(sto)

	require.NoError(t, c.SetTarget(123))
	require.NoError(t, c.SetAdjustmentWindow(456))
	require.NoError(t, c.SetBacklog(789))

	target, _ := c.Target()
	window, _ := c.AdjustmentWindow()
	backlog, _ := c.Backlog()
	require.Equal(t, uint64(123), target)
	require.Equal(t, uint32(456), window)
	require.Equal(t, uint64(789), backlog)

	weightedResources := map[uint8]uint64{
		uint8(multigas.ResourceKindComputation):   10,
		uint8(multigas.ResourceKindStorageAccess): 20,
	}
	require.NoError(t, c.SetResourceWeights(weightedResources))

	w1, _ := c.ResourceWeight(uint8(multigas.ResourceKindComputation))
	w2, _ := c.ResourceWeight(uint8(multigas.ResourceKindStorageAccess))
	require.Equal(t, uint64(10), w1)
	require.Equal(t, uint64(20), w2)

	res, err := c.ResourcesWithWeights()
	require.NoError(t, err)
	require.Len(t, res, 2)
	require.Equal(t, uint64(10), res[multigas.ResourceKindComputation])
	require.Equal(t, uint64(20), res[multigas.ResourceKindStorageAccess])

	used, err := c.UsedResources()
	require.NoError(t, err)
	require.Len(t, used, 2)
	require.Contains(t, used, multigas.ResourceKindComputation)
	require.Contains(t, used, multigas.ResourceKindStorageAccess)

	maxWeight, err := c.MaxWeight()
	require.NoError(t, err)
	require.Equal(t, uint64(20), maxWeight)

	require.NoError(t, c.Clear())
	target, _ = c.Target()
	backlog, _ = c.Backlog()
	require.Zero(t, target)
	require.Zero(t, backlog)
	res, _ = c.ResourcesWithWeights()
	require.Empty(t, res)

	used, err = c.UsedResources()
	require.NoError(t, err)
	require.Len(t, used, 0)

	maxWeight, err = c.MaxWeight()
	require.NoError(t, err)
	require.Equal(t, uint64(0), maxWeight)
}

func TestMultiGasConstraintResourceWeightsValidation(t *testing.T) {
	sto := storage.NewMemoryBacked(burn.NewSystemBurner(nil, false))
	c := OpenMultiGasConstraint(sto)

	// invalid kind
	weights := map[uint8]uint64{
		uint8(multigas.NumResourceKind): 10,
	}
	require.Error(t, c.SetResourceWeights(weights))

	// valid set
	valid := map[uint8]uint64{
		uint8(multigas.ResourceKindComputation):   3,
		uint8(multigas.ResourceKindStorageAccess): 7,
	}
	require.NoError(t, c.SetResourceWeights(valid))

	maxWeight, err := c.maxWeight.Get()
	require.NoError(t, err)
	require.Equal(t, uint64(7), maxWeight)
}

func TestMultiGasConstraintBacklogAggregation(t *testing.T) {
	sto := storage.NewMemoryBacked(burn.NewSystemBurner(nil, false))
	c := OpenMultiGasConstraint(sto)

	require.NoError(t, c.SetTarget(5))
	require.NoError(t, c.SetAdjustmentWindow(2))

	require.NoError(t, c.SetResourceWeights(map[uint8]uint64{
		uint8(multigas.ResourceKindComputation):   1,
		uint8(multigas.ResourceKindStorageAccess): 2,
	}))

	mg := multigas.MultiGasFromPairs(
		multigas.Pair{Kind: multigas.ResourceKindComputation, Amount: 10},
		multigas.Pair{Kind: multigas.ResourceKindStorageAccess, Amount: 10},
	)

	require.NoError(t, c.GrowBacklog(mg))
}

func TestMultiGasConstraintBacklogGrowth(t *testing.T) {
	sto := storage.NewMemoryBacked(burn.NewSystemBurner(nil, false))
	c := OpenMultiGasConstraint(sto)

	require.NoError(t, c.SetTarget(10))
	require.NoError(t, c.SetAdjustmentWindow(5))

	require.NoError(t, c.SetResourceWeights(map[uint8]uint64{
		uint8(multigas.ResourceKindComputation):   1,
		uint8(multigas.ResourceKindStorageAccess): 2,
	}))

	mg1 := multigas.MultiGasFromPairs(
		multigas.Pair{Kind: multigas.ResourceKindComputation, Amount: 10},
		multigas.Pair{Kind: multigas.ResourceKindStorageAccess, Amount: 10},
	)

	require.NoError(t, c.GrowBacklog(mg1))

	b1, err := c.Backlog()
	require.NoError(t, err)
	require.Equal(t, uint64(30), b1, "initial backlog: 1*10 + 2*10 = 30")

	mg2 := multigas.MultiGasFromPairs(
		multigas.Pair{Kind: multigas.ResourceKindComputation, Amount: 5},
		multigas.Pair{Kind: multigas.ResourceKindStorageAccess, Amount: 15},
	)

	require.NoError(t, c.GrowBacklog(mg2))

	b2, err := c.Backlog()
	require.NoError(t, err)
	// new backlog = old (30) + 1*5 + 2*15 = 30 + 35 = 65
	require.Equal(t, uint64(65), b2, "backlog should accumulate across calls")
}

func TestMultiGasConstraintBacklogDecay(t *testing.T) {
	sto := storage.NewMemoryBacked(burn.NewSystemBurner(nil, false))
	c := OpenMultiGasConstraint(sto)

	require.NoError(t, c.SetTarget(10))
	require.NoError(t, c.SetAdjustmentWindow(5))

	require.NoError(t, c.SetResourceWeights(map[uint8]uint64{
		uint8(multigas.ResourceKindComputation):   1,
		uint8(multigas.ResourceKindStorageAccess): 2,
	}))

	// Initial backlog: 1*10 + 2*10 = 30
	mgGrow := multigas.MultiGasFromPairs(
		multigas.Pair{Kind: multigas.ResourceKindComputation, Amount: 10},
		multigas.Pair{Kind: multigas.ResourceKindStorageAccess, Amount: 10},
	)
	require.NoError(t, c.GrowBacklog(mgGrow))

	b1, err := c.Backlog()
	require.NoError(t, err)
	require.Equal(t, uint64(30), b1)

	// First decay: 1*3 + 2*4 = 11 → new backlog = 30 - 11 = 19
	mgDecay1 := multigas.MultiGasFromPairs(
		multigas.Pair{Kind: multigas.ResourceKindComputation, Amount: 3},
		multigas.Pair{Kind: multigas.ResourceKindStorageAccess, Amount: 4},
	)
	require.NoError(t, c.ShrinkBacklog(mgDecay1))

	b2, err := c.Backlog()
	require.NoError(t, err)
	require.Equal(t, uint64(19), b2, "30 - (1*3 + 2*4) = 19")

	// Second decay underflows: 1*50 + 2*50 = 150 → should clamp to zero
	mgDecay2 := multigas.MultiGasFromPairs(
		multigas.Pair{Kind: multigas.ResourceKindComputation, Amount: 50},
		multigas.Pair{Kind: multigas.ResourceKindStorageAccess, Amount: 50},
	)
	require.NoError(t, c.ShrinkBacklog(mgDecay2))

	b3, err := c.Backlog()
	require.NoError(t, err)
	require.Equal(t, uint64(0), b3, "backlog must clamp to zero on underflow")
}
