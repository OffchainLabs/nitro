// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

package challenge_testing

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/bold/chain-abstraction"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
)

const (
	LevelZeroBlockEdgeHeight     = 1 << 5
	LevelZeroBigStepEdgeHeight   = 1 << 5
	LevelZeroSmallStepEdgeHeight = 1 << 5
	MaxDataSize                  = 117964
)

type Opt func(c *rollupgen.Config)

func WithNumBigStepLevels(num uint8) Opt {
	return func(c *rollupgen.Config) {
		c.NumBigStepLevel = num
	}
}

func WithLayerZeroHeights(h *protocol.LayerZeroHeights) Opt {
	return func(c *rollupgen.Config) {
		c.LayerZeroBlockEdgeHeight = new(big.Int).SetUint64(h.BlockChallengeHeight.Uint64())
		c.LayerZeroBigStepEdgeHeight = new(big.Int).SetUint64(h.BigStepChallengeHeight.Uint64())
		c.LayerZeroSmallStepEdgeHeight = new(big.Int).SetUint64(h.SmallStepChallengeHeight.Uint64())
	}
}

func WithConfirmPeriodBlocks(num uint64) Opt {
	return func(c *rollupgen.Config) {
		c.ConfirmPeriodBlocks = num
	}
}

func WithChallengeGracePeriodBlocks(num uint64) Opt {
	return func(c *rollupgen.Config) {
		c.ChallengeGracePeriodBlocks = num
	}
}

func WithBaseStakeValue(num *big.Int) Opt {
	return func(c *rollupgen.Config) {
		c.BaseStake = num
	}
}

func WithChainConfig(cfg string) Opt {
	return func(c *rollupgen.Config) {
		c.ChainConfig = cfg
	}
}

func GenerateRollupConfig(
	prod bool,
	wasmModuleRoot common.Hash,
	rollupOwner common.Address,
	chainId *big.Int,
	loserStakeEscrow common.Address,
	miniStakeValues []*big.Int,
	stakeToken common.Address,
	genesisExecutionState rollupgen.AssertionState,
	genesisInboxCount *big.Int,
	anyTrustFastConfirmer common.Address,
	opts ...Opt,
) rollupgen.Config {
	var confirmPeriod uint64
	if prod {
		confirmPeriod = 45818
	} else {
		confirmPeriod = 25
	}

	var gracePeriod uint64
	if prod {
		gracePeriod = 14400
	} else {
		gracePeriod = 3
	}

	cfg := rollupgen.Config{
		MiniStakeValues:        miniStakeValues,
		ConfirmPeriodBlocks:    confirmPeriod,
		StakeToken:             stakeToken,
		BaseStake:              big.NewInt(1),
		WasmModuleRoot:         wasmModuleRoot,
		Owner:                  rollupOwner,
		LoserStakeEscrow:       loserStakeEscrow,
		ChainId:                chainId,
		ChainConfig:            "{ 'config': 'Test config'}",
		MinimumAssertionPeriod: big.NewInt(75),
		ValidatorAfkBlocks:     201600,
		SequencerInboxMaxTimeVariation: rollupgen.ISequencerInboxMaxTimeVariation{
			DelayBlocks:   big.NewInt(60 * 60 * 24 / 15),
			FutureBlocks:  big.NewInt(12),
			DelaySeconds:  big.NewInt(60 * 60 * 24),
			FutureSeconds: big.NewInt(60 * 60),
		},
		LayerZeroBlockEdgeHeight:     big.NewInt(LevelZeroBlockEdgeHeight),
		LayerZeroBigStepEdgeHeight:   big.NewInt(LevelZeroBigStepEdgeHeight),
		LayerZeroSmallStepEdgeHeight: big.NewInt(LevelZeroSmallStepEdgeHeight),
		GenesisAssertionState:        genesisExecutionState,
		GenesisInboxCount:            genesisInboxCount,
		AnyTrustFastConfirmer:        anyTrustFastConfirmer,
		NumBigStepLevel:              1,
		ChallengeGracePeriodBlocks:   gracePeriod,
		DataCostEstimate:             big.NewInt(0),
	}
	for _, o := range opts {
		o(&cfg)
	}
	return cfg
}
