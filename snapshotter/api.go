package snapshotter

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/util/containers"
)

type DatabaseSnapshotterAPI struct {
	bc          *core.BlockChain
	snapshotter *DatabaseSnapshotter

	promise     containers.PromiseInterface[SnapshotResult]
	promiseLock sync.Mutex
}

func NewDatabaseSnapshotterAPI(bc *core.BlockChain, snapshotter *DatabaseSnapshotter) *DatabaseSnapshotterAPI {
	return &DatabaseSnapshotterAPI{
		bc:          bc,
		snapshotter: snapshotter,
	}
}

func (a *DatabaseSnapshotterAPI) Snapshot(ctx context.Context, number rpc.BlockNumber) error {
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
			return fmt.Errorf("invalid block number: %d", number)
		}
		// #nosec G115
		block := a.bc.GetBlockByNumber(uint64(number))
		if block == nil {
			return fmt.Errorf("block #%d not found", number)
		}
		header = block.Header()
	}
	if header == nil {
		return fmt.Errorf("block #%d not found", number)
	}
	var promise containers.PromiseInterface[SnapshotResult]
	started := make(chan struct{}, 1)
	err := func() error {
		a.promiseLock.Lock()
		defer a.promiseLock.Unlock()
		if a.promise != nil {
			if a.promise.Ready() {
				return errors.New("needs rewind")
			}
			return errors.New("already running")
		}
		promise = a.snapshotter.Trigger(header.Hash(), started)
		a.promise = promise
		return nil
	}()
	if err != nil {
		return err
	}
	timer := time.NewTicker(5 * time.Second)
	defer timer.Stop()
	select {
	case <-promise.ReadyChan():
		_, err := promise.Current()
		return err
	case <-started:
	case <-timer.C:
	}
	return nil
}

// if rewind is set to true, the snapshotter api will be rewound if the result is ready
func (a *DatabaseSnapshotterAPI) Result(ctx context.Context, rewind bool) (SnapshotResult, error) {
	a.promiseLock.Lock()
	defer a.promiseLock.Unlock()
	if a.promise == nil {
		return SnapshotResult{}, errors.New("not started yet")
	}
	promise := a.promise
	if a.promise.Ready() && rewind {
		a.promise = nil
	}
	return promise.Current()
}
