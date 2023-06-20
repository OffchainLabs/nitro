package challenge_testing

import (
	"math/big"

	protocol "github.com/OffchainLabs/challenge-protocol-v2/chain-abstraction"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/rollupgen"
	"github.com/ethereum/go-ethereum/common"
)

func GenerateRollupConfig(
	prod bool,
	wasmModuleRoot common.Hash,
	rollupOwner common.Address,
	chainId *big.Int,
	loserStakeEscrow common.Address,
	miniStakeValue *big.Int,
) rollupgen.Config {
	var confirmPeriod uint64
	if prod {
		confirmPeriod = 45818
	} else {
		confirmPeriod = 25
	}

	return rollupgen.Config{
		MiniStakeValue:      miniStakeValue,
		ConfirmPeriodBlocks: confirmPeriod,
		StakeToken:          common.Address{},
		BaseStake:           big.NewInt(100),
		WasmModuleRoot:      wasmModuleRoot,
		Owner:               rollupOwner,
		LoserStakeEscrow:    loserStakeEscrow,
		ChainId:             chainId,
		SequencerInboxMaxTimeVariation: rollupgen.ISequencerInboxMaxTimeVariation{
			DelayBlocks:   big.NewInt(60 * 60 * 24 / 15),
			FutureBlocks:  big.NewInt(12),
			DelaySeconds:  big.NewInt(60 * 60 * 24),
			FutureSeconds: big.NewInt(60 * 60),
		},
		LayerZeroBlockEdgeHeight:     big.NewInt(protocol.LevelZeroBlockEdgeHeight),
		LayerZeroBigStepEdgeHeight:   big.NewInt(protocol.LevelZeroBigStepEdgeHeight),
		LayerZeroSmallStepEdgeHeight: big.NewInt(protocol.LevelZeroSmallStepEdgeHeight),
	}
}
