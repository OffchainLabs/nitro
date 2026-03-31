// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package parent

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

// knownConfigs maps known Ethereum chain IDs to their chain configurations.
// Note that this is not exhaustive; users can add more configurations via
// chain_config.json files or can even override existing known configurations by providing
// the same chain ID.
var (
	knownConfigs = map[uint64]*params.ChainConfig{
		params.MainnetChainConfig.ChainID.Uint64():         params.MainnetChainConfig,
		params.HoleskyChainConfig.ChainID.Uint64():         params.HoleskyChainConfig,
		params.SepoliaChainConfig.ChainID.Uint64():         params.SepoliaChainConfig,
		params.AllDevChainProtocolChanges.ChainID.Uint64(): params.AllDevChainProtocolChanges,
	}
)

//go:embed chain_config.json
var DefaultChainsConfigBytes []byte

func init() {
	var chainsConfig []*params.ChainConfig
	err := json.Unmarshal(DefaultChainsConfigBytes, &chainsConfig)
	if err != nil {
		panic(fmt.Errorf("error unmarshalling default chainsConfig: %w", err))
	}
	for _, chainConfig := range chainsConfig {
		knownConfigs[chainConfig.ChainID.Uint64()] = chainConfig
	}
}

type Config struct {
	ConfigPollInterval time.Duration
}

var DefaultConfig = Config{
	ConfigPollInterval: time.Hour,
}

var TestConfig = Config{
	ConfigPollInterval: time.Second,
}

type ConfigFetcher func() *Config

// nilBlobScheduleErrorThreshold is the number of consecutive eth_config polls
// returning nil BlobSchedule before the log level escalates from Debug to Error.
// With the default 1-hour poll interval, a threshold of 6 means operators will
// see an error after ~6 hours of a defective endpoint, while still tolerating
// occasional transient nil responses.
const nilBlobScheduleErrorThreshold = 6

type ParentChain struct {
	stopwaiter.StopWaiter
	ChainID  *big.Int
	L1Reader *headerreader.HeaderReader
	config   ConfigFetcher

	cachedEthConfig            atomic.Pointer[ethConfigResponse]
	consecutiveNilBlobSchedule atomic.Int32
}

func NewParentChain(ctx context.Context, chainID *big.Int, l1Reader *headerreader.HeaderReader) *ParentChain {
	return NewParentChainWithConfig(ctx, chainID, l1Reader, nil)
}

// NewParentChainWithConfig creates a new ParentChain and eagerly fetches
// config from L1. The eager fetch is needed because some ParentChain instances
// are never Start()'d (e.g. DataposterOnlyUsedToCreateValidatorWalletContract)
// and still need valid config. This means instances that are Start()'d will
// make a duplicate call on the first poll interval, which is acceptable given
// the call is lightweight and polling is infrequent (default: once per hour).
func NewParentChainWithConfig(ctx context.Context, chainID *big.Int, l1Reader *headerreader.HeaderReader, config ConfigFetcher) *ParentChain {
	if config == nil {
		config = func() *Config { return &DefaultConfig }
	}

	parentChain := ParentChain{
		ChainID:  chainID,
		L1Reader: l1Reader,
		config:   config,
	}

	fetchCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if l1Reader != nil {
		if err := parentChain.pollEthConfig(fetchCtx); err != nil {
			if fetchCtx.Err() != nil {
				log.Warn("Eager eth_config fetch timed out, will use static config until next successful poll", "timeout", "10s")
			} else {
				log.Warn("Failed to poll parent chain eth_config, will use static config until next successful poll", "err", err)
			}
		}
	}

	return &parentChain
}

// ethConfigResponse mirrors the eth_config RPC response (EIP-7910).
// We only parse the fields we need.
type ethConfigResponse struct {
	Current *ethConfigEntry `json:"current"`
	Next    *ethConfigEntry `json:"next"`
}

type ethConfigEntry struct {
	BlobSchedule   *params.BlobConfig `json:"blobSchedule"`
	ChainId        *hexutil.Big       `json:"chainId"`
	ActivationTime uint64             `json:"activationTime"`
}

// Start begins polling the parent chain's eth_config RPC.
// ParentChain is shared between execution and consensus nodes when co-located,
// so it may be Start()'d twice. We call StopWaiterSafe.Start() directly
// (bypassing the panicking StopWaiter.Start()) and check for
// ErrAlreadyStarted to handle double-start gracefully.
func (p *ParentChain) Start(ctxIn context.Context) {
	if err := p.StopWaiterSafe.Start(ctxIn, p); err != nil {
		if errors.Is(err, stopwaiter.ErrAlreadyStarted) && p.Started() {
			log.Debug("ParentChain already started (shared between execution and consensus nodes)")
			return
		}
		panic(fmt.Sprintf("Failed to start ParentChain poller: %v", err))
	}
	if p.L1Reader == nil {
		return
	}

	p.CallIteratively(func(ctx context.Context) time.Duration {
		if err := p.pollEthConfig(ctx); err != nil && ctx.Err() == nil {
			log.Warn("Failed to poll parent chain eth_config", "err", err)
		}
		return p.config().ConfigPollInterval
	})
}

func (p *ParentChain) pollEthConfig(ctx context.Context) error {
	client := p.L1Reader.Client()
	var resp ethConfigResponse
	if err := client.Client().CallContext(ctx, &resp, "eth_config"); err != nil {
		return fmt.Errorf("calling eth_config: %w", err)
	}
	if resp.Current == nil {
		return fmt.Errorf("eth_config returned nil current config")
	}
	if resp.Current.BlobSchedule == nil {
		count := p.consecutiveNilBlobSchedule.Add(1)
		if count >= nilBlobScheduleErrorThreshold {
			log.Error("eth_config has returned nil BlobSchedule for multiple consecutive polls, cache may be stale",
				"chainID", p.ChainID, "consecutiveNilPolls", count)
		} else {
			log.Debug("eth_config returned nil BlobSchedule, skipping cache update",
				"chainID", p.ChainID, "consecutiveNilPolls", count)
		}
		return nil
	}
	if resp.Current.ChainId == nil {
		return fmt.Errorf("eth_config response missing chainId, cannot validate against expected chain %s", p.ChainID)
	} else if resp.Current.ChainId.ToInt().Cmp(p.ChainID) != 0 {
		return fmt.Errorf("chain ID mismatch: expected %s, got %s", p.ChainID, resp.Current.ChainId.ToInt())
	}

	if resp.Next != nil {
		if resp.Next.ChainId == nil {
			return fmt.Errorf("eth_config next entry missing chainId, cannot validate against expected chain %s", p.ChainID)
		} else if resp.Next.ChainId.ToInt().Cmp(p.ChainID) != 0 {
			return fmt.Errorf("next config chain ID mismatch: expected %s, got %s", p.ChainID, resp.Next.ChainId.ToInt())
		}
		if resp.Next.BlobSchedule == nil {
			log.Warn("eth_config next entry has nil BlobSchedule", "chainID", p.ChainID)
		}
		if resp.Next.ActivationTime == 0 {
			log.Warn("eth_config next entry has zero ActivationTime, ignoring", "chainID", p.ChainID)
			resp.Next = nil
		}
	}

	p.consecutiveNilBlobSchedule.Store(0)
	p.cachedEthConfig.Store(&resp)
	log.Info("Updated parent chain config from eth_config",
		"currentTarget", resp.Current.BlobSchedule.Target,
		"currentMax", resp.Current.BlobSchedule.Max,
		"hasNext", resp.Next != nil,
	)
	return nil
}

// CachedBlobConfig returns the current blob config from the last successful
// eth_config poll, or nil if no config has been fetched yet.
func (p *ParentChain) CachedBlobConfig() *params.BlobConfig {
	if cached := p.cachedEthConfig.Load(); cached != nil && cached.Current != nil {
		return cached.Current.BlobSchedule
	}
	return nil
}

// ErrUnknownChain is returned when an unknown chain ID is requested.
type ErrUnknownChain struct {
	ChainID *big.Int
}

// Error implements the error interface.
func (e ErrUnknownChain) Error() string {
	return "unknown chain ID " + e.ChainID.String()
}

// chainConfig returns the parent chain's configuration.
//
// Note: This is really a hack. This method returns one of the known Ethereum
// L1 chain configurations, but should not be used by chains that need to post
// blobs to some other parent chain.
func (p *ParentChain) chainConfig() (*params.ChainConfig, error) {
	cfg, ok := knownConfigs[p.ChainID.Uint64()]
	if !ok {
		return nil, ErrUnknownChain{p.ChainID}
	}
	return cfg, nil
}

// blobConfig returns the currently active blob config, preferring the
// value fetched from the parent chain's eth_config RPC.
// Falls back to the hardcoded chain config if no cached value is available
// or if headerTime is earlier than the current cached config.
// Returns (nil, nil) when the parent chain does not support blobs (e.g. an
// Arbitrum L2 acting as parent for an L3). Callers must handle a nil config.
func (p *ParentChain) blobConfig(headerTime uint64) (*params.BlobConfig, error) {
	cached := p.cachedEthConfig.Load()
	if cached != nil {
		// If next config exists and headerTime is at or past its activation,
		// use the next config's blob schedule.
		if cached.Next != nil && cached.Next.BlobSchedule != nil &&
			headerTime >= cached.Next.ActivationTime {
			return cached.Next.BlobSchedule, nil
		}
		var currentActivationTime uint64
		if cached.Current != nil {
			currentActivationTime = cached.Current.ActivationTime
		}

		// Only use current if headerTime is at or past its activation;
		// otherwise fall through to the static chain config for older forks.
		if cached.Current != nil && cached.Current.BlobSchedule != nil &&
			headerTime >= currentActivationTime {
			return cached.Current.BlobSchedule, nil
		}
		log.Warn("Falling back to static blob config despite cached eth_config",
			"headerTime", headerTime,
			"currentActivationTime", currentActivationTime,
		)
	}
	// Fall back to the hardcoded chain config. If the parent chain is an
	// Arbitrum chain (e.g. an L2 acting as parent for an L3), it won't
	// have blob support: chainConfig().BlobConfig() returns nil because
	// IsArbitrum() is true. In that case we return nil so callers can
	// handle it gracefully (e.g. return 0 for blob fees).
	staticBlobConfig, err := p.chainConfig()
	if err != nil {
		var unknownErr ErrUnknownChain
		if errors.As(err, &unknownErr) {
			return nil, nil
		}
		return nil, err
	}
	// Replicate the logic of the unexported latestBlobConfig() from eip4844
	// by resolving the active fork for headerTime, then looking up its blob config.
	return staticBlobConfig.BlobConfig(staticBlobConfig.LatestFork(headerTime, 0)), nil
}

// MaxBlobGasPerBlock returns the maximum blob gas per block according to
// the configuration of the parent chain.
// Passing in a nil header will use the time from the latest header.
func (p *ParentChain) MaxBlobGasPerBlock(ctx context.Context, h *types.Header) (uint64, error) {
	header := h
	if h == nil {
		lh, err := p.L1Reader.LastHeader(ctx)
		if err != nil {
			return 0, err
		}
		header = lh
	}
	blobConfig, err := p.blobConfig(header.Time)
	if err != nil {
		return 0, err
	}
	// nil means the parent chain doesn't support blobs (e.g. Arbitrum L2 parent).
	if blobConfig == nil {
		return 0, nil
	}
	// #nosec G115
	return uint64(blobConfig.Max) * params.BlobTxBlobGasPerBlob, nil
}

// BlobFeePerByte returns the blob fee per byte according to the configuration
// of the parent chain.
// Passing in a nil header will use the time from the latest header.
func (p *ParentChain) BlobFeePerByte(ctx context.Context, h *types.Header) (*big.Int, error) {
	header := h
	if h == nil {
		lh, err := p.L1Reader.LastHeader(ctx)
		if err != nil {
			return big.NewInt(0), err
		}
		header = lh
	}
	bc, err := p.blobConfig(header.Time)
	if err != nil {
		return big.NewInt(0), err
	}
	// nil means the parent chain doesn't support blobs (e.g. Arbitrum L2
	// parent). Return 0, matching CalcBlobFee behavior.
	if bc == nil {
		return big.NewInt(0), nil
	}
	return eip4844.CalcBlobFeeWithConfig(bc, header.ExcessBlobGas), nil
}
