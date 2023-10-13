package arbnode

import (
	"context"
	"errors"

	"github.com/OffchainLabs/bold/solgen/go/rollupgen"
	"github.com/OffchainLabs/bold/testing/setup"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/util/headerreader"
)

func DeployBOLDOnL1(ctx context.Context, parentChainReader *headerreader.HeaderReader, deployAuth *bind.TransactOpts, batchPoster common.Address, authorizeValidators uint64, config rollupgen.Config) (*setup.RollupAddresses, error) {
	if config.WasmModuleRoot == (common.Hash{}) {
		return nil, errors.New("no machine specified")
	}

	// prod := false
	// loserStakeEscrow := common.Address{}
	// miniStake := big.NewInt(1)
	// genesisExecutionState := rollupgen.ExecutionState{
	// 	GlobalState:   rollupgen.GlobalState{},
	// 	MachineStatus: 1,
	// }
	// genesisInboxCount := big.NewInt(0)
	// anyTrustFastConfirmer := common.Address{}
	// cfg := challenge_testing.GenerateRollupConfig(
	// 	prod,
	// 	wasmModuleRoot,
	// 	l1TransactionOpts.From,
	// 	chainId,
	// 	loserStakeEscrow,
	// 	miniStake,
	// 	stakeToken,
	// 	genesisExecutionState,
	// 	genesisInboxCount,
	// 	anyTrustFastConfirmer,
	// 	challenge_testing.WithLayerZeroHeights(&protocol.LayerZeroHeights{
	// 		BlockChallengeHeight:     blockChallengeLeafHeight,
	// 		BigStepChallengeHeight:   bigStepChallengeLeafHeight,
	// 		SmallStepChallengeHeight: smallStepChallengeLeafHeight,
	// 	}),
	// 	challenge_testing.WithNumBigStepLevels(uint8(5)),       // TODO: Hardcoded.
	// 	challenge_testing.WithConfirmPeriodBlocks(uint64(150)), // TODO: Hardcoded.
	// )

	addresses, err := setup.DeployFullRollupStack(
		ctx,
		parentChainReader.Client(),
		deployAuth,
		deployAuth.From,
		config,
		false, // do not use mock bridge.
		false, // do not use a mock one step prover
	)
	if err != nil {
		return nil, err
	}

	// rollupCreator, _, validatorUtils, validatorWalletCreator, err := deployRollupCreator(ctx, parentChainReader, deployAuth)
	// if err != nil {
	// 	return nil, fmt.Errorf("error deploying rollup creator: %w", err)
	// }

	// var validatorAddrs []common.Address
	// for i := uint64(1); i <= authorizeValidators; i++ {
	// 	validatorAddrs = append(validatorAddrs, crypto.CreateAddress(validatorWalletCreator, i))
	// }

	// tx, err := rollupCreator.CreateRollup(
	// 	deployAuth,
	// 	config,
	// 	batchPoster,
	// 	validatorAddrs,
	// )
	// if err != nil {
	// 	return nil, fmt.Errorf("error submitting create rollup tx: %w", err)
	// }
	// receipt, err := parentChainReader.WaitForTxApproval(ctx, tx)
	// if err != nil {
	// 	return nil, fmt.Errorf("error executing create rollup tx: %w", err)
	// }
	// info, err := rollupCreator.ParseRollupCreated(*receipt.Logs[len(receipt.Logs)-1])
	// if err != nil {
	// 	return nil, fmt.Errorf("error parsing rollup created log: %w", err)
	// }

	// return &chaininfo.RollupAddresses{
	// 	Bridge:                 info.Bridge,
	// 	Inbox:                  info.InboxAddress,
	// 	SequencerInbox:         info.SequencerInbox,
	// 	DeployedAt:             receipt.BlockNumber.Uint64(),
	// 	Rollup:                 info.RollupAddress,
	// 	ValidatorUtils:         validatorUtils,
	// 	ValidatorWalletCreator: validatorWalletCreator,
	// }, nil
	return addresses, nil
}
