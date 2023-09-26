// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

package challenge_testing

import (
	"math/big"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	"github.com/OffchainLabs/bold/solgen/go/rollupgen"
	"github.com/ethereum/go-ethereum/common"
)

const (
	LevelZeroBlockEdgeHeight     = 1 << 5
	LevelZeroBigStepEdgeHeight   = 1 << 5
	LevelZeroSmallStepEdgeHeight = 1 << 5
	MaxDataSize                  = 117964
)

type Opt func(c *rollupgen.Config)

func WithNumBigStepLevels(num uint64) Opt {
	return func(c *rollupgen.Config) {
		c.NumBigStepLevel = new(big.Int).SetUint64(num)
	}
}

func WithLayerZeroHeights(h *protocol.LayerZeroHeights) Opt {
	return func(c *rollupgen.Config) {
		c.LayerZeroBlockEdgeHeight = new(big.Int).SetUint64(h.BlockChallengeHeight)
		c.LayerZeroBigStepEdgeHeight = new(big.Int).SetUint64(h.BigStepChallengeHeight)
		c.LayerZeroSmallStepEdgeHeight = new(big.Int).SetUint64(h.SmallStepChallengeHeight)
	}
}

func GenerateRollupConfig(
	prod bool,
	wasmModuleRoot common.Hash,
	rollupOwner common.Address,
	chainId *big.Int,
	loserStakeEscrow common.Address,
	miniStakeValue *big.Int,
	stakeToken common.Address,
	genesisExecutionState rollupgen.ExecutionState,
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
		MiniStakeValue:      miniStakeValue,
		ConfirmPeriodBlocks: confirmPeriod,
		StakeToken:          stakeToken,
		BaseStake:           big.NewInt(100),
		WasmModuleRoot:      wasmModuleRoot,
		Owner:               rollupOwner,
		LoserStakeEscrow:    loserStakeEscrow,
		ChainId:             chainId,
		ChainConfig:         "{ 'config': 'Test config'}",
		SequencerInboxMaxTimeVariation: rollupgen.ISequencerInboxMaxTimeVariation{
			DelayBlocks:   big.NewInt(60 * 60 * 24 / 15),
			FutureBlocks:  big.NewInt(12),
			DelaySeconds:  big.NewInt(60 * 60 * 24),
			FutureSeconds: big.NewInt(60 * 60),
		},
		LayerZeroBlockEdgeHeight:     big.NewInt(LevelZeroBlockEdgeHeight),
		LayerZeroBigStepEdgeHeight:   big.NewInt(LevelZeroBigStepEdgeHeight),
		LayerZeroSmallStepEdgeHeight: big.NewInt(LevelZeroSmallStepEdgeHeight),
		GenesisExecutionState:        genesisExecutionState,
		GenesisInboxCount:            genesisInboxCount,
		AnyTrustFastConfirmer:        anyTrustFastConfirmer,
		NumBigStepLevel:              new(big.Int).SetUint64(1),
		ChallengeGracePeriodBlocks:   gracePeriod,
	}
	for _, o := range opts {
		o(&cfg)
	}
	return cfg
}
