// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package parent

import (
	"context"
	_ "embed"
	"encoding/json"
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
		panic(fmt.Errorf("error marshalling default chainsConfig: %w", err))
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

type ParentChain struct {
	stopwaiter.StopWaiter
	ChainID  *big.Int
	L1Reader *headerreader.HeaderReader
	config   ConfigFetcher

	cachedBlobConfig atomic.Pointer[params.BlobConfig]
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
		if err := parentChain.pollEthConfig(fetchCtx); err != nil && fetchCtx.Err() == nil {
			log.Warn("Failed to poll parent chain eth_config from NewParentChainWithConfig", "err", err)
		}
	}

	return &parentChain
}

// ethConfigResponse mirrors the eth_config RPC response (EIP-7910).
// We only parse the fields we need.
type ethConfigResponse struct {
	Current *ethConfigEntry `json:"current"`
}

type ethConfigEntry struct {
	BlobSchedule *params.BlobConfig `json:"blobSchedule"`
	ChainId      *hexutil.Big       `json:"chainId"`
}

func (p *ParentChain) Start(ctxIn context.Context) {
	if err := p.StopWaiterSafe.Start(ctxIn, p); err != nil {
		log.Debug("Already started (shared between execution and consensus nodes)", "err", err)
		return
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
	if resp.Current.BlobSchedule != nil {
		if resp.Current.ChainId != nil && resp.Current.ChainId.ToInt().Cmp(p.ChainID) != 0 {
			return fmt.Errorf("chain ID mismatch: expected %s, got %s", p.ChainID, resp.Current.ChainId.ToInt())
		}

		p.cachedBlobConfig.Store(resp.Current.BlobSchedule)
		log.Info("Updated parent chain blob config from eth_config",
			"target", resp.Current.BlobSchedule.Target,
			"max", resp.Current.BlobSchedule.Max,
			"updateFraction", resp.Current.BlobSchedule.UpdateFraction,
		)
	}
	return nil
}

// CachedBlobConfig returns the cached blob config from the last successful
// eth_config poll, or nil if no config has been fetched yet.
func (p *ParentChain) CachedBlobConfig() *params.BlobConfig {
	return p.cachedBlobConfig.Load()
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

// cachedBlobConfigMaxAge is the maximum age of a header (relative to the
// current wall-clock time) for which we trust the cached blob config obtained
// from eth_config. The cached value reflects the *current* fork's blob
// schedule, so it is only valid for recent headers. For older headers (e.g.
// during historical replay) we fall through to the static chain config which
// can resolve the correct blob schedule for any timestamp.
const cachedBlobConfigMaxAge = 12 * time.Hour

// blobConfig returns the currently active blob config, preferring the
// value fetched from the parent chain's eth_config RPC.
// Falls back to the hardcoded chain config if no cached value is available.
func (p *ParentChain) blobConfig(headerTime uint64) (*params.BlobConfig, error) {
	if cachedBlobConfig := p.cachedBlobConfig.Load(); cachedBlobConfig != nil {
		headerAge := time.Since(time.Unix(int64(headerTime), 0)) // #nosec G115
		if headerAge < cachedBlobConfigMaxAge {
			return cachedBlobConfig, nil
		}
	}
	staticBlobConfig, err := p.chainConfig()
	if err != nil {
		return nil, err
	}
	// We bring staticBlobConfig.LatestFork() to an earlier spot which is the equivalent of latestBlobConfig
	// from eip4844 which is called from MaxBlobGasPerBlock and CalcBlobFee.
	// currentArbosVersion as 0 matches latestBlobConfig from eip4844
	blobConfig := staticBlobConfig.BlobConfig(staticBlobConfig.LatestFork(headerTime, 0))
	// We return an error if staticBlobConfig.IsArbitrum() since blobConfig == nil. From the
	// hardcoded chainConfigs none of them sets ArbitrumChainParams.EnableArbOS so we're safe
	// with current configuration
	if blobConfig == nil {
		return nil, fmt.Errorf("no blob config for parent chain %s at time %d", p.ChainID, headerTime)
	}
	return blobConfig, nil
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
	// TODO: do we want to return 0 if config.IsArbitrum()? Again, none of the hardcoded
	// chainConfig sets ArbitrumChainParams.EnableArbOS. But then we need to get that info
	// from somewhere
	return eip4844.CalcBlobFeeWithConfig(bc, header.ExcessBlobGas), nil
}
