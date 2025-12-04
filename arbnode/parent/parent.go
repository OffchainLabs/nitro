// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package parent

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/util/headerreader"
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

type ParentChain struct {
	ChainID  *big.Int
	L1Reader *headerreader.HeaderReader
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

// MaxBlobGasPerBlock returns the maximum blob gas per block according to
// according to the configuration of the parent chain.
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
	pCfg, err := p.chainConfig()
	if err != nil {
		return 0, err
	}
	return eip4844.MaxBlobGasPerBlock(pCfg, header.Time), nil
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
	pCfg, err := p.chainConfig()
	if err != nil {
		return big.NewInt(0), err
	}
	return eip4844.CalcBlobFee(pCfg, header), nil
}
