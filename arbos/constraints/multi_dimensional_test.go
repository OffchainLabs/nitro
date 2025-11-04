package constraints

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
	require.Equal(t, uint64(456), window)
	require.Equal(t, uint64(789), backlog)

	require.NoError(t, c.SetResourceWeight(uint8(multigas.ResourceKindComputation), 10))
	require.NoError(t, c.SetResourceWeight(uint8(multigas.ResourceKindStorageAccess), 20))
	require.NoError(t, c.SetResourceWeight(uint8(multigas.ResourceKindL1Calldata), 0)) // unused

	w1, _ := c.ResourceWeight(uint8(multigas.ResourceKindComputation))
	w2, _ := c.ResourceWeight(uint8(multigas.ResourceKindStorageAccess))
	require.Equal(t, uint64(10), w1)
	require.Equal(t, uint64(20), w2)

	res, err := c.ResourcesWithWeights()
	require.NoError(t, err)
	require.Len(t, res, 2)
	require.Equal(t, uint64(10), res[multigas.ResourceKindComputation])
	require.Equal(t, uint64(20), res[multigas.ResourceKindStorageAccess])

	require.NoError(t, c.Clear())
	target, _ = c.Target()
	backlog, _ = c.Backlog()
	require.Zero(t, target)
	require.Zero(t, backlog)
	res, _ = c.ResourcesWithWeights()
	require.Empty(t, res)
}

func TestMultiGasConstraintResourceKindEdgeCases(t *testing.T) {
	sto := storage.NewMemoryBacked(burn.NewSystemBurner(nil, false))
	c := OpenMultiGasConstraint(sto)

	require.Error(t, c.SetResourceWeight(0, 111))
	_, err := c.ResourceWeight(0)
	require.Error(t, err)

	lastKind := uint8(multigas.NumResourceKind - 1)
	require.NoError(t, c.SetResourceWeight(lastKind, 222))
	v, err := c.ResourceWeight(lastKind)
	require.NoError(t, err)
	require.Equal(t, uint64(222), v)

	outOfRange := uint8(multigas.NumResourceKind)
	err = c.SetResourceWeight(outOfRange, 333)
	require.Error(t, err)
	_, err = c.ResourceWeight(outOfRange)
	require.Error(t, err)

	res, err := c.ResourcesWithWeights()
	require.NoError(t, err)
	require.Len(t, res, 1)
	require.Equal(t, uint64(222), res[multigas.ResourceKindWasmComputation])
}

func TestMultiGasConstraintBacklogAggregationAndDecomposition(t *testing.T) {
	sto := storage.NewMemoryBacked(burn.NewSystemBurner(nil, false))
	c := OpenMultiGasConstraint(sto)

	require.NoError(t, c.SetResourceWeight(uint8(multigas.ResourceKindComputation), 2))
	require.NoError(t, c.SetResourceWeight(uint8(multigas.ResourceKindStorageAccess), 3))
	require.NoError(t, c.SetResourceWeight(uint8(multigas.ResourceKindL1Calldata), 5))
	require.NoError(t, c.SetResourceWeight(uint8(multigas.ResourceKindL2Calldata), 0)) // unused

	mg := multigas.MultiGasFromPairs(
		multigas.Pair{Kind: multigas.ResourceKindComputation, Amount: 10},
		multigas.Pair{Kind: multigas.ResourceKindHistoryGrowth, Amount: 11},
		multigas.Pair{Kind: multigas.ResourceKindStorageAccess, Amount: 12},
		multigas.Pair{Kind: multigas.ResourceKindStorageGrowth, Amount: 13},
		multigas.Pair{Kind: multigas.ResourceKindL1Calldata, Amount: 14},
		multigas.Pair{Kind: multigas.ResourceKindL2Calldata, Amount: 15},
		multigas.Pair{Kind: multigas.ResourceKindWasmComputation, Amount: 16},
	)

	require.NoError(t, c.SetBacklogWithMultigas(mg))

	backlog, err := c.Backlog()
	require.NoError(t, err)
	require.Equal(t, uint64(126), backlog)

	v1, err := c.ResourceWeightedBacklog(uint8(multigas.ResourceKindComputation))
	require.NoError(t, err)
	require.Equal(t, uint64(25), v1) // 126 * 2 / 10 = 25.2 → 25

	v2, err := c.ResourceWeightedBacklog(uint8(multigas.ResourceKindStorageAccess))
	require.NoError(t, err)
	require.Equal(t, uint64(37), v2) // 126 * 3 / 10 = 37.8 → 37

	v3, err := c.ResourceWeightedBacklog(uint8(multigas.ResourceKindL1Calldata))
	require.NoError(t, err)
	require.Equal(t, uint64(63), v3) // 126 * 5 / 10 = 63

	v4, err := c.ResourceWeightedBacklog(uint8(multigas.ResourceKindL2Calldata))
	require.NoError(t, err)
	require.Zero(t, v4)
}
