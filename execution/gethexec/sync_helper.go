package gethexec

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbutil"
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
	Enabled:          true, // TODO
	CheckpointPeriod: 10 * 1000,
	CheckpointCache:  16,
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
}

type NitroSyncHelperConfigFetcher func() *NitroSyncHelperConfig

func NewNitroSyncHelper(config NitroSyncHelperConfigFetcher, bc *core.BlockChain) *NitroSyncHelper {
	return &NitroSyncHelper{
		config:          config,
		bc:              bc,
		checkpointCache: NewCheckpointCache(int(config().CheckpointCache)),
	}
}

func (h *NitroSyncHelper) Start(ctx context.Context) {
	h.StopWaiter.Start(ctx, h)
	h.StopWaiter.LaunchThread(func(ctx context.Context) {
		for {
			select {
			case c := <-h.newConfirmed:
				if updated, previous := h.updateLastConfirmed(&c); updated {
					h.scanNewConfirmedCheckpoints(ctx, &c, previous)
				}
			case <-ctx.Done():
				return
			}
		}
	})
}

// returns true and previous value if last confirmed was updated
// otherwise returns false and nil
func (h *NitroSyncHelper) updateLastConfirmed(newConfirmed *Confirmed) (bool, *Confirmed) {
	// validate block hash
	header := h.bc.GetHeaderByNumber(uint64(newConfirmed.BlockNumber))
	newConfirmed.Header = header
	if hash := header.Hash(); hash.Cmp(newConfirmed.BlockHash) != 0 {
		log.Error("Confirmed BlockHash doesn't match header hash from blockchain", "blockNumber", newConfirmed.BlockNumber, "headerHash", hash, "confirmedBlockHash", newConfirmed.BlockHash)
		return false, nil
	}

	h.lastConfirmedLock.Lock()
	defer h.lastConfirmedLock.Unlock()
	previousConfirmed := h.lastConfirmed
	if previousConfirmed != nil {
		if previousConfirmed.BlockNumber == newConfirmed.BlockNumber {
			if previousConfirmed.BlockHash != newConfirmed.BlockHash || previousConfirmed.Node != newConfirmed.Node {
				log.Error("New confirmed block number same as previous confirmed, but block hash and/or node number doesn't match", "blockNumber", newConfirmed.BlockNumber, "newConfirmedBlockHash", newConfirmed.BlockHash, "previousConfirmedBlockHash", previousConfirmed.BlockHash, "newConfirmedNode", newConfirmed.Node, "previousConfirmedNode", previousConfirmed.Node)
			}
			return false, nil
		}
		if previousConfirmed.BlockNumber > newConfirmed.BlockNumber {
			log.Warn("New confirmed block number lower then previous confirmed block ", "newBlockNumber", newConfirmed.BlockNumber, "previousBlockNumber", previousConfirmed.BlockNumber, "newBlockHash", newConfirmed.BlockHash, "previousBlockHash", previousConfirmed.BlockHash, "newNode", newConfirmed.Node, "previousNode", previousConfirmed.Node)
			// TODO do we want to continue either way?
			return false, nil
		}
	}
	h.lastConfirmed = newConfirmed
	return true, previousConfirmed
}

// scan for new confirmed and available checkpoints and add them to cache
func (h *NitroSyncHelper) scanNewConfirmedCheckpoints(ctx context.Context, newConfirmed *Confirmed, previousConfirmed *Confirmed) {
	period := int64(h.config().CheckpointPeriod)
	var nextCheckpoint int64
	if previousConfirmed == nil {
		genesis := int64(h.bc.Config().ArbitrumChainParams.GenesisBlockNum)
		nextCheckpoint = (genesis/period + 1) * period
	} else {
		nextCheckpoint = (previousConfirmed.BlockNumber/period + 1) * period
	}
	for nextCheckpoint <= newConfirmed.BlockNumber && ctx.Err() == nil {
		header := h.bc.GetHeaderByNumber(uint64(nextCheckpoint))
		if header == nil {
			log.Error("missing block header", "blockNumber", nextCheckpoint)
			// TODO should we continue and just skip this checkpoint?
			return
		}
		// TODO can we just use h.bc.StateAt?
		if _, err := state.New(header.Root, h.bc.StateCache(), nil); err != nil {
			h.checkpointCache.Add(header)
		}
		nextCheckpoint += period
	}
}

func GetForceTriedbCommitHookForConfig(config NitroSyncHelperConfigFetcher) core.ForceTriedbCommitHook {
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
		Node:        node,
		Header:      nil,
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
	if header == nil {
		return false, errors.New("header is nil")
	}
	hash := header.Hash()
	// TODO
	// call to consensus node, block hash + uint64 (node number)
	_ = hash
	return false, nil
}

type Confirmed struct {
	BlockNumber int64
	BlockHash   common.Hash
	Node        uint64
	Header      *types.Header // filled out later in scanNewConfirmedCheckpoints
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
