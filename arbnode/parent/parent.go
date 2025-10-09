// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package parent

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/util/headerreader"
)

func newUint64(val uint64) *uint64 { return &val }

var (
	DevNet3ChainConfig = &params.ChainConfig{
		ChainID:                 big.NewInt(7023102237),
		HomesteadBlock:          big.NewInt(0),
		DAOForkBlock:            nil,
		DAOForkSupport:          true,
		EIP150Block:             big.NewInt(0),
		EIP155Block:             big.NewInt(0),
		EIP158Block:             big.NewInt(0),
		ByzantiumBlock:          big.NewInt(0),
		ConstantinopleBlock:     big.NewInt(0),
		PetersburgBlock:         big.NewInt(0),
		IstanbulBlock:           big.NewInt(0),
		MuirGlacierBlock:        nil,
		BerlinBlock:             big.NewInt(0),
		LondonBlock:             big.NewInt(0),
		ArrowGlacierBlock:       nil,
		GrayGlacierBlock:        nil,
		TerminalTotalDifficulty: big.NewInt(0),
		MergeNetsplitBlock:      nil,
		ShanghaiTime:            newUint64(0),
		CancunTime:              newUint64(0),
		PragueTime:              newUint64(0),
		OsakaTime:               newUint64(1753379304),
		BPO1Time:                newUint64(1753477608),
		BPO2Time:                newUint64(1753575912),
		BPO3Time:                newUint64(1753674216),
		BPO4Time:                newUint64(1753772520),
		BPO5Time:                newUint64(1753889256),
		DepositContractAddress:  common.HexToAddress("0x00000000219ab540356cBB839Cbe05303d7705Fa"),
		Ethash:                  new(params.EthashConfig),
		BlobScheduleConfig: &params.BlobScheduleConfig{
			Cancun: params.DefaultCancunBlobConfig,
			Prague: params.DefaultPragueBlobConfig,
			Osaka:  params.DefaultOsakaBlobConfig,
			BPO1:   &params.BlobConfig{Target: 9, Max: 12, UpdateFraction: 5007716},
			BPO2:   &params.BlobConfig{Target: 12, Max: 15, UpdateFraction: 5007716},
			BPO3:   &params.BlobConfig{Target: 15, Max: 18, UpdateFraction: 5007716},
			BPO4:   &params.BlobConfig{Target: 6, Max: 9, UpdateFraction: 5007716},
			BPO5:   &params.BlobConfig{Target: 15, Max: 20, UpdateFraction: 5007716},
		},
	}
	knownConfigs = map[uint64]*params.ChainConfig{
		params.MainnetChainConfig.ChainID.Uint64():         params.MainnetChainConfig,
		params.HoleskyChainConfig.ChainID.Uint64():         params.HoleskyChainConfig,
		params.SepoliaChainConfig.ChainID.Uint64():         params.SepoliaChainConfig,
		params.AllDevChainProtocolChanges.ChainID.Uint64(): params.AllDevChainProtocolChanges,
		DevNet3ChainConfig.ChainID.Uint64():                DevNet3ChainConfig,
	}
)

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

// SupportsCellProofs returns whether the parent chain has activated the Osaka fork
// (Fusaka), which introduced cell proofs for blobs.
// Passing in a nil header will use the time from the latest header.
func (p *ParentChain) SupportsCellProofs(ctx context.Context, h *types.Header) (bool, error) {
	header := h
	if h == nil {
		lh, err := p.L1Reader.LastHeader(ctx)
		if err != nil {
			return false, err
		}
		header = lh
	}
	pCfg, err := p.chainConfig()
	if err != nil {
		return false, err
	}
	if pCfg.IsArbitrum() {
		// Arbitrum does not support blob transactions, so this should not have been called.
		return false, errors.New("parent chain is Arbitrum and does not support blobs")
	}
	// arbosVersion 0 because we're checking L1 (not L2 Arbitrum)
	return pCfg.IsOsaka(pCfg.LondonBlock, header.Time, 0), nil
}
