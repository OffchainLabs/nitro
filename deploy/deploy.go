package deploy

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/solgen/go/challengegen"
	"github.com/offchainlabs/nitro/solgen/go/ospgen"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
	"github.com/offchainlabs/nitro/solgen/go/upgrade_executorgen"
	"github.com/offchainlabs/nitro/solgen/go/yulgen"
	"github.com/offchainlabs/nitro/util/headerreader"
)

func andTxSucceeded(ctx context.Context, l1Reader *headerreader.HeaderReader, tx *types.Transaction, err error) error {
	if err != nil {
		return fmt.Errorf("error submitting tx: %w", err)
	}
	_, err = l1Reader.WaitForTxApproval(ctx, tx)
	if err != nil {
		return fmt.Errorf("error executing tx: %w", err)
	}
	return nil
}

func deployBridgeCreator(ctx context.Context, l1Reader *headerreader.HeaderReader, auth *bind.TransactOpts, maxDataSize *big.Int, isUsingFeeToken bool) (common.Address, error) {
	client := l1Reader.Client()

	/// deploy eth based templates
	bridgeTemplate, tx, _, err := bridgegen.DeployBridge(auth, client)
	err = andTxSucceeded(ctx, l1Reader, tx, err)
	if err != nil {
		return common.Address{}, fmt.Errorf("bridge deploy error: %w", err)
	}

	reader4844, tx, _, err := yulgen.DeployReader4844(auth, client)
	err = andTxSucceeded(ctx, l1Reader, tx, err)
	if err != nil {
		return common.Address{}, fmt.Errorf("blob basefee reader deploy error: %w", err)
	}
	seqInboxTemplate, tx, _, err := bridgegen.DeploySequencerInbox(auth, client, maxDataSize, reader4844, isUsingFeeToken)
	err = andTxSucceeded(ctx, l1Reader, tx, err)
	if err != nil {
		return common.Address{}, fmt.Errorf("sequencer inbox deploy error: %w", err)
	}

	inboxTemplate, tx, _, err := bridgegen.DeployInbox(auth, client, maxDataSize)
	err = andTxSucceeded(ctx, l1Reader, tx, err)
	if err != nil {
		return common.Address{}, fmt.Errorf("inbox deploy error: %w", err)
	}

	rollupEventBridgeTemplate, tx, _, err := rollupgen.DeployRollupEventInbox(auth, client)
	err = andTxSucceeded(ctx, l1Reader, tx, err)
	if err != nil {
		return common.Address{}, fmt.Errorf("rollup event bridge deploy error: %w", err)
	}

	outboxTemplate, tx, _, err := bridgegen.DeployOutbox(auth, client)
	err = andTxSucceeded(ctx, l1Reader, tx, err)
	if err != nil {
		return common.Address{}, fmt.Errorf("outbox deploy error: %w", err)
	}

	ethBasedTemplates := rollupgen.BridgeCreatorBridgeContracts{
		Bridge:           bridgeTemplate,
		SequencerInbox:   seqInboxTemplate,
		Inbox:            inboxTemplate,
		RollupEventInbox: rollupEventBridgeTemplate,
		Outbox:           outboxTemplate,
	}

	/// deploy ERC20 based templates
	erc20BridgeTemplate, tx, _, err := bridgegen.DeployERC20Bridge(auth, client)
	err = andTxSucceeded(ctx, l1Reader, tx, err)
	if err != nil {
		return common.Address{}, fmt.Errorf("bridge deploy error: %w", err)
	}

	erc20InboxTemplate, tx, _, err := bridgegen.DeployERC20Inbox(auth, client, maxDataSize)
	err = andTxSucceeded(ctx, l1Reader, tx, err)
	if err != nil {
		return common.Address{}, fmt.Errorf("inbox deploy error: %w", err)
	}

	erc20RollupEventBridgeTemplate, tx, _, err := rollupgen.DeployERC20RollupEventInbox(auth, client)
	err = andTxSucceeded(ctx, l1Reader, tx, err)
	if err != nil {
		return common.Address{}, fmt.Errorf("rollup event bridge deploy error: %w", err)
	}

	erc20OutboxTemplate, tx, _, err := bridgegen.DeployERC20Outbox(auth, client)
	err = andTxSucceeded(ctx, l1Reader, tx, err)
	if err != nil {
		return common.Address{}, fmt.Errorf("outbox deploy error: %w", err)
	}

	erc20BasedTemplates := rollupgen.BridgeCreatorBridgeContracts{
		Bridge:           erc20BridgeTemplate,
		SequencerInbox:   seqInboxTemplate,
		Inbox:            erc20InboxTemplate,
		RollupEventInbox: erc20RollupEventBridgeTemplate,
		Outbox:           erc20OutboxTemplate,
	}

	bridgeCreatorAddr, tx, _, err := rollupgen.DeployBridgeCreator(auth, client, ethBasedTemplates, erc20BasedTemplates)
	err = andTxSucceeded(ctx, l1Reader, tx, err)
	if err != nil {
		return common.Address{}, fmt.Errorf("bridge creator deploy error: %w", err)
	}

	return bridgeCreatorAddr, nil
}

func deployChallengeFactory(ctx context.Context, l1Reader *headerreader.HeaderReader, auth *bind.TransactOpts) (common.Address, common.Address, error) {
	client := l1Reader.Client()
	osp0, tx, _, err := ospgen.DeployOneStepProver0(auth, client)
	err = andTxSucceeded(ctx, l1Reader, tx, err)
	if err != nil {
		return common.Address{}, common.Address{}, fmt.Errorf("osp0 deploy error: %w", err)
	}

	ospMem, tx, _, err := ospgen.DeployOneStepProverMemory(auth, client)
	err = andTxSucceeded(ctx, l1Reader, tx, err)
	if err != nil {
		return common.Address{}, common.Address{}, fmt.Errorf("ospMemory deploy error: %w", err)
	}

	ospMath, tx, _, err := ospgen.DeployOneStepProverMath(auth, client)
	err = andTxSucceeded(ctx, l1Reader, tx, err)
	if err != nil {
		return common.Address{}, common.Address{}, fmt.Errorf("ospMath deploy error: %w", err)
	}

	ospHostIo, tx, _, err := ospgen.DeployOneStepProverHostIo(auth, client)
	err = andTxSucceeded(ctx, l1Reader, tx, err)
	if err != nil {
		return common.Address{}, common.Address{}, fmt.Errorf("ospHostIo deploy error: %w", err)
	}

	challengeManagerAddr, tx, _, err := challengegen.DeployChallengeManager(auth, client)
	err = andTxSucceeded(ctx, l1Reader, tx, err)
	if err != nil {
		return common.Address{}, common.Address{}, fmt.Errorf("challenge manager deploy error: %w", err)
	}

	ospEntryAddr, tx, _, err := ospgen.DeployOneStepProofEntry(auth, client, osp0, ospMem, ospMath, ospHostIo)
	err = andTxSucceeded(ctx, l1Reader, tx, err)
	if err != nil {
		return common.Address{}, common.Address{}, fmt.Errorf("ospEntry deploy error: %w", err)
	}

	return ospEntryAddr, challengeManagerAddr, nil
}

func deployRollupCreator(ctx context.Context, l1Reader *headerreader.HeaderReader, auth *bind.TransactOpts, maxDataSize *big.Int, isUsingFeeToken bool) (*rollupgen.RollupCreator, common.Address, common.Address, common.Address, error) {
	bridgeCreator, err := deployBridgeCreator(ctx, l1Reader, auth, maxDataSize, isUsingFeeToken)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, fmt.Errorf("bridge creator deploy error: %w", err)
	}

	ospEntryAddr, challengeManagerAddr, err := deployChallengeFactory(ctx, l1Reader, auth)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, err
	}

	rollupAdminLogic, tx, _, err := rollupgen.DeployRollupAdminLogic(auth, l1Reader.Client())
	err = andTxSucceeded(ctx, l1Reader, tx, err)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, fmt.Errorf("rollup admin logic deploy error: %w", err)
	}

	rollupUserLogic, tx, _, err := rollupgen.DeployRollupUserLogic(auth, l1Reader.Client())
	err = andTxSucceeded(ctx, l1Reader, tx, err)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, fmt.Errorf("rollup user logic deploy error: %w", err)
	}

	rollupCreatorAddress, tx, rollupCreator, err := rollupgen.DeployRollupCreator(auth, l1Reader.Client())
	err = andTxSucceeded(ctx, l1Reader, tx, err)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, fmt.Errorf("rollup creator deploy error: %w", err)
	}

	upgradeExecutor, tx, _, err := upgrade_executorgen.DeployUpgradeExecutor(auth, l1Reader.Client())
	err = andTxSucceeded(ctx, l1Reader, tx, err)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, fmt.Errorf("upgrade executor deploy error: %w", err)
	}

	validatorUtils, tx, _, err := rollupgen.DeployValidatorUtils(auth, l1Reader.Client())
	err = andTxSucceeded(ctx, l1Reader, tx, err)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, fmt.Errorf("validator utils deploy error: %w", err)
	}

	validatorWalletCreator, tx, _, err := rollupgen.DeployValidatorWalletCreator(auth, l1Reader.Client())
	err = andTxSucceeded(ctx, l1Reader, tx, err)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, fmt.Errorf("validator wallet creator deploy error: %w", err)
	}

	l2FactoriesDeployHelper, tx, _, err := rollupgen.DeployDeployHelper(auth, l1Reader.Client())
	err = andTxSucceeded(ctx, l1Reader, tx, err)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, fmt.Errorf("deploy helper creator deploy error: %w", err)
	}

	tx, err = rollupCreator.SetTemplates(
		auth,
		bridgeCreator,
		ospEntryAddr,
		challengeManagerAddr,
		rollupAdminLogic,
		rollupUserLogic,
		upgradeExecutor,
		validatorUtils,
		validatorWalletCreator,
		l2FactoriesDeployHelper,
	)
	err = andTxSucceeded(ctx, l1Reader, tx, err)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, fmt.Errorf("rollup set template error: %w", err)
	}

	return rollupCreator, rollupCreatorAddress, validatorUtils, validatorWalletCreator, nil
}

func DeployOnL1(ctx context.Context, parentChainReader *headerreader.HeaderReader, deployAuth *bind.TransactOpts, batchPosters []common.Address, batchPosterManager common.Address, authorizeValidators uint64, config rollupgen.Config, nativeToken common.Address, maxDataSize *big.Int, isUsingFeeToken bool) (*chaininfo.RollupAddresses, error) {
	if config.WasmModuleRoot == (common.Hash{}) {
		return nil, errors.New("no machine specified")
	}

	rollupCreator, _, validatorUtils, validatorWalletCreator, err := deployRollupCreator(ctx, parentChainReader, deployAuth, maxDataSize, isUsingFeeToken)
	if err != nil {
		return nil, fmt.Errorf("error deploying rollup creator: %w", err)
	}

	var validatorAddrs []common.Address
	for i := uint64(1); i <= authorizeValidators; i++ {
		validatorAddrs = append(validatorAddrs, crypto.CreateAddress(validatorWalletCreator, i))
	}

	deployParams := rollupgen.RollupCreatorRollupDeploymentParams{
		Config:                    config,
		Validators:                validatorAddrs,
		MaxDataSize:               maxDataSize,
		NativeToken:               nativeToken,
		DeployFactoriesToL2:       false,
		MaxFeePerGasForRetryables: big.NewInt(0), // needed when utility factories are deployed
		BatchPosters:              batchPosters,
		BatchPosterManager:        batchPosterManager,
	}

	tx, err := rollupCreator.CreateRollup(
		deployAuth,
		deployParams,
	)
	if err != nil {
		return nil, fmt.Errorf("error submitting create rollup tx: %w", err)
	}
	receipt, err := parentChainReader.WaitForTxApproval(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("error executing create rollup tx: %w", err)
	}
	info, err := rollupCreator.ParseRollupCreated(*receipt.Logs[len(receipt.Logs)-1])
	if err != nil {
		return nil, fmt.Errorf("error parsing rollup created log: %w", err)
	}

	return &chaininfo.RollupAddresses{
		Bridge:                 info.Bridge,
		Inbox:                  info.InboxAddress,
		SequencerInbox:         info.SequencerInbox,
		DeployedAt:             receipt.BlockNumber.Uint64(),
		Rollup:                 info.RollupAddress,
		NativeToken:            nativeToken,
		UpgradeExecutor:        info.UpgradeExecutor,
		ValidatorUtils:         validatorUtils,
		ValidatorWalletCreator: validatorWalletCreator,
	}, nil
}
