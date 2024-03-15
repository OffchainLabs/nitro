package gethexec

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/offchainlabs/nitro/validator"
	flag "github.com/spf13/pflag"
)

type NitroSyncHelperConfig struct {
	Enabled          bool   `koanf:"enabled"`
	CheckpointPeriod uint64 `koanf:"checkpoint-period"`
	CheckpointCache  uint   `koanf:"checkpoint-cache"`
}

func NitroSyncHelperConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Uint64(prefix+".checkpoint-period", NitroSyncHelperConfigDefault.CheckpointPeriod, "number of blocks between sync checkpoints")
	f.Uint(prefix+".checkpoint-cache", NitroSyncHelperConfigDefault.CheckpointCache, "number of recently confirmed checkpoints to keep in cache")
}

var NitroSyncHelperConfigDefault = NitroSyncHelperConfig{
	Enabled:          true,      // TODO
	CheckpointPeriod: 10 * 1000, // TODO
	CheckpointCache:  16,        // TODO
}

// implements arbitrum.SyncHelper
// implements staker.LatestConfirmedNotifier
type NitroSyncHelper struct {
	stopwaiter.StopWaiter
	config          NitroSyncHelperConfigFetcher
	bc              *core.BlockChain
	checkpointCache *CheckpointCache
	newConfirmed    chan Confirmed

	lastConfirmedLock sync.RWMutex
	lastConfirmed     *Confirmed

	confirmedNodeHelper execution.ConfirmedNodeHelper
}

type NitroSyncHelperConfigFetcher func() *NitroSyncHelperConfig

func NewNitroSyncHelper(config NitroSyncHelperConfigFetcher, bc *core.BlockChain) *NitroSyncHelper {
	return &NitroSyncHelper{
		config:          config,
		bc:              bc,
		checkpointCache: NewCheckpointCache(int(config().CheckpointCache)),
		newConfirmed:    make(chan Confirmed), // TODO
	}
}

func (h *NitroSyncHelper) SetConfirmedNodeHelper(confirmedHelper execution.ConfirmedNodeHelper) {
	if h.Started() {
		panic("trying to set confirmed node validator after nitro sync helper start")
	}
	if h.confirmedNodeHelper != nil {
		panic("trying to set confirmed node validator when already set")
	}
	h.confirmedNodeHelper = confirmedHelper
}

func (h *NitroSyncHelper) Start(ctx context.Context) error {
	if err := h.StopWaiter.StopWaiterSafe.Start(ctx, h); err != nil {
		return err
	}
	if h.confirmedNodeHelper != nil {
		err := h.confirmedNodeHelper.SubscribeLatest(h)
		if err != nil {
			return fmt.Errorf("Failed to subscribe for latest confirmed notifications: %w", err)
		}
	}
	return h.StopWaiterSafe.LaunchThreadSafe(func(ctx context.Context) {
		for {
			select {
			// TODO refactor the newConfirmed channel (might not be needed as confirmedNodeHelper should handle non blocking update propagation)
			case c := <-h.newConfirmed:
				if err := h.updateLastConfirmed(ctx, &c); err != nil {
					log.Error("Sync helper failed to update last confirmed", "err", err)
				}
			case <-ctx.Done():
				return
			}
		}
	})
}

// returns true and previous value if last confirmed was updated
// otherwise returns false and nil
func (h *NitroSyncHelper) updateLastConfirmed(ctx context.Context, newConfirmed *Confirmed) error {
	// validate block hash
	header := h.bc.GetHeaderByNumber(uint64(newConfirmed.BlockNumber))
	newConfirmed.Header = header
	if hash := header.Hash(); hash.Cmp(newConfirmed.BlockHash) != 0 {
		return fmt.Errorf("confirmed BlockHash doesn't match header hash from blockchain, block #%d %s, confirmed %s", newConfirmed.BlockNumber, hash, newConfirmed.BlockHash)
	}

	h.lastConfirmedLock.Lock()
	defer h.lastConfirmedLock.Unlock()
	previousConfirmed := h.lastConfirmed
	if previousConfirmed != nil {
		if newConfirmed.BlockNumber == previousConfirmed.BlockNumber {
			if newConfirmed.BlockHash != previousConfirmed.BlockHash || newConfirmed.Node != previousConfirmed.Node {
				return fmt.Errorf("New confirmed block number same as previous confirmed, but block hash and/or node number doesn't match, block #%d, previous %s node %d, new %s node %d", previousConfirmed.BlockNumber, previousConfirmed.BlockHash, previousConfirmed.Node, newConfirmed.BlockHash, newConfirmed.Node)
			}
			return nil
		}
		if newConfirmed.BlockNumber < previousConfirmed.BlockNumber {
			// TODO do we want to continue either way?
			return fmt.Errorf("New confirmed block number lower then previous confirmed, previous block #%d %s node %d, new block #%d %s node %d", previousConfirmed.BlockNumber, previousConfirmed.BlockHash, previousConfirmed.Node, newConfirmed.BlockNumber, newConfirmed.BlockHash, newConfirmed.Node)
		}
	}
	h.lastConfirmed = newConfirmed
	return h.scanNewConfirmedCheckpoints(ctx, newConfirmed, previousConfirmed)
}

// scan for new confirmed and available checkpoints and add them to cache
func (h *NitroSyncHelper) scanNewConfirmedCheckpoints(ctx context.Context, newConfirmed *Confirmed, previousConfirmed *Confirmed) error {
	period := int64(h.config().CheckpointPeriod)
	var nextCheckpoint int64
	if previousConfirmed == nil {
		genesis := int64(h.bc.Config().ArbitrumChainParams.GenesisBlockNum)
		nextCheckpoint = (genesis/period + 1) * period // TODO add option to start the scan from n blocks before nextCheckpoint.BlockNumber
	} else {
		nextCheckpoint = (previousConfirmed.BlockNumber/period + 1) * period
	}
	for nextCheckpoint <= newConfirmed.BlockNumber && ctx.Err() == nil {
		header := h.bc.GetHeaderByNumber(uint64(nextCheckpoint))
		if header == nil {
			// TODO should we continue and just skip this checkpoint?
			return fmt.Errorf("missing header for block #%d", nextCheckpoint)
		}
		// TODO can we just use h.bc.StateAt?
		if _, err := state.New(header.Root, h.bc.StateCache(), nil); err == nil {
			h.checkpointCache.Add(header)
		}
		nextCheckpoint += period
	}
	return nil
}

func GetForceTriedbCommitHookForConfig(config NitroSyncHelperConfigFetcher) core.ForceTriedbCommitHook {
	if !config().Enabled {
		// TODO do we want to support hot-reloading of Enabled?
		return nil
	}
	return func(block *types.Block, processing time.Duration, gas uint64) bool {
		if block.NumberU64() == 0 {
			return false
		}
		commit := block.NumberU64()%config().CheckpointPeriod == 0
		// TODO add condition for minimal processing since last checkpoint
		// TODO add condition for minimal gas used since last checkpoint
		_ = processing
		_ = gas
		return commit
	}
}

// implements staker.LatestConfirmedNotifier
func (h *NitroSyncHelper) UpdateLatestConfirmed(count arbutil.MessageIndex, globalState validator.GoGlobalState, node uint64) {
	genesis := h.bc.Config().ArbitrumChainParams.GenesisBlockNum
	h.newConfirmed <- Confirmed{
		BlockNumber: arbutil.MessageCountToBlockNumber(count, genesis),
		BlockHash:   globalState.BlockHash,
		// TODO do we want to also use SendRoot?
		Node:   node,
		Header: nil,
	}
}

func (h *NitroSyncHelper) LastCheckpoint() (*types.Header, error) {
	if last := h.checkpointCache.Last(); last != nil {
		return last, nil
	}
	return nil, errors.New("unavailable")
}

func (h *NitroSyncHelper) CheckpointSupported(header *types.Header) (bool, error) {
	if header == nil {
		return false, errors.New("header is nil")
	}
	return h.checkpointCache.Has(header), nil
}

func (h *NitroSyncHelper) LastConfirmed() (*types.Header, uint64, error) {
	h.lastConfirmedLock.RLock()
	defer h.lastConfirmedLock.RUnlock()
	if h.lastConfirmed == nil {
		return nil, 0, errors.New("unavailable")
	}
	return h.lastConfirmed.Header, h.lastConfirmed.Node, nil
}

func (h *NitroSyncHelper) ValidateConfirmed(header *types.Header, node uint64) (bool, error) {
	if !h.Started() {
		return false, errors.New("not started")
	}
	if header == nil {
		return false, errors.New("header is nil")
	}
	if h.confirmedNodeHelper == nil {
		return false, errors.New("confirmed node validator unavailable")
	}
	hash := header.Hash()
	return h.confirmedNodeHelper.Validate(node, hash)
}

type Confirmed struct {
	BlockNumber int64
	BlockHash   common.Hash
	Node        uint64
	Header      *types.Header // filled out later in updateLastConfirmed
}

type CheckpointCache struct {
	capacity int

	lock           sync.RWMutex
	checkpointsMap map[uint64]*types.Header
	checkpoints    []*types.Header
}

// capacity has to be greater then 0
func NewCheckpointCache(capacity int) *CheckpointCache {
	if capacity <= 0 {
		capacity = 1
	}
	cache := &CheckpointCache{
		capacity:       capacity,
		checkpointsMap: make(map[uint64]*types.Header, capacity),
		checkpoints:    make([]*types.Header, 0, capacity),
	}
	return cache
}

func (c *CheckpointCache) Add(header *types.Header) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if len(c.checkpoints) >= c.capacity {
		var dropped *types.Header
		dropped, c.checkpoints = c.checkpoints[0], c.checkpoints[1:]
		delete(c.checkpointsMap, dropped.Number.Uint64())
	}
	number := header.Number.Uint64()
	if previous, has := c.checkpointsMap[number]; has {
		// TODO do we expect this to happen in normal operations?
		log.Warn("CheckpointCache: duplicate checkpoint header added, replacing previous", "number", number)
		var i int
		for i := 0; i < len(c.checkpoints); i++ {
			if c.checkpoints[i] == previous {
				break
			}
		}
		if i == len(c.checkpoints) {
			// shouldn't ever happen
			log.Error("CheckpointCache: duplicate not found in checkpoints slice", "number", number)
		} else {
			c.checkpoints = append(c.checkpoints[:i], c.checkpoints[i+1:]...)
		}
	}
	c.checkpoints = append(c.checkpoints, header)
	c.checkpointsMap[number] = header
}

func (c *CheckpointCache) Has(header *types.Header) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	cached, has := c.checkpointsMap[header.Number.Uint64()]
	if !has {
		return false
	}
	// TODO won't comparing fields be more efficient?
	return header.Hash().Cmp(cached.Hash()) == 0
}

func (c *CheckpointCache) Last() *types.Header {
	c.lock.RLock()
	defer c.lock.RUnlock()
	if len(c.checkpoints) > 0 {
		return c.checkpoints[len(c.checkpoints)-1]
	}
	return nil
}
