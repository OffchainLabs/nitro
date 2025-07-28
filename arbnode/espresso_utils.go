package arbnode

import (
	"context"
)

var (
	binarySearch_LessThanTarget    = -1
	binarySearch_GreaterThanTarget = 1
	binarySearch_EqualToTarget     = 0
)

// Looks for the first block number that is equal to or greater than the target
func binarySearchForBlockNumber(
	ctx context.Context,
	start, end uint64,
	f func(context.Context, uint64) (int, error),
) (uint64, error) {
	for start < end {
		mid := (start + end) / 2
		result, err := f(ctx, mid)
		if err != nil {
			return 0, err
		}
		if result == binarySearch_GreaterThanTarget {
			end = mid
		} else if result == binarySearch_LessThanTarget {
			start = mid + 1
		} else {
			// We are looking for the first block number.
			// So the loop should continue until start == end
			end = mid
		}
	}
	return start, nil
}
