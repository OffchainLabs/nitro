package snapshotter

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
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

func (a *DatabaseSnapshotterAPI) Snapshot(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (SnapshotResult, error) {
	var blockHash common.Hash
	if number, ok := blockNrOrHash.Number(); ok {
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
	} else if hash, ok := blockNrOrHash.Hash(); ok {
		block := a.bc.GetBlockByHash(hash)
		if block == nil {
			return SnapshotResult{}, fmt.Errorf("block %s not found", hash.Hex())
		}
	} else {
		return SnapshotResult{}, errors.New("either block number or block hash must be specified")
	}
	promise := a.snapshotter.Trigger(blockHash)
	result, err := promise.Await(ctx)
	if err != nil {
		return SnapshotResult{}, err
	}
	return result, nil
}
