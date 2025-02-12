package snapshotter

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

type DatabaseSnapshotterAPI struct {
	bc          *core.BlockChain
	snapshotter *DatabaseSnapshotter
}

func NewDatabaseSnapshotterAPI(bc *core.BlockChain, snapshotter *DatabaseSnapshotter) *DatabaseSnapshotterAPI {
	return &DatabaseSnapshotterAPI{
		bc:          bc,
		snapshotter: snapshotter,
	}
}

func (a *DatabaseSnapshotterAPI) Snapshot(ctx context.Context, number rpc.BlockNumber) (SnapshotResult, error) {
	if number == rpc.PendingBlockNumber {
		number = rpc.LatestBlockNumber
	}
	var header *types.Header
	switch number {
	case rpc.LatestBlockNumber:
		header = a.bc.CurrentBlock()
	case rpc.FinalizedBlockNumber:
		header = a.bc.CurrentFinalBlock()
	case rpc.SafeBlockNumber:
		header = a.bc.CurrentSafeBlock()
	default:
		if number < 0 {
			return SnapshotResult{}, fmt.Errorf("invalid block number: %d", number)
		}
		// #nosec G115
		block := a.bc.GetBlockByNumber(uint64(number))
		if block == nil {
			return SnapshotResult{}, fmt.Errorf("block #%d not found", number)
		}
		header = block.Header()
	}
	if header == nil {
		return SnapshotResult{}, fmt.Errorf("block #%d not found", number)
	}
	promise := a.snapshotter.Trigger(header.Hash())
	result, err := promise.Await(ctx)
	if err != nil {
		return SnapshotResult{}, err
	}
	return result, nil
}
