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
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/solgen/go/bridge_legacy_gen"
	"github.com/offchainlabs/nitro/solgen/go/challenge_legacy_gen"
	"github.com/offchainlabs/nitro/solgen/go/osp_legacy_gen"
	"github.com/offchainlabs/nitro/solgen/go/rollup_legacy_gen"
	"github.com/offchainlabs/nitro/solgen/go/upgrade_executorgen"
	"github.com/offchainlabs/nitro/solgen/go/yulgen"
	"github.com/offchainlabs/nitro/util/headerreader"
)

func GenerateLegacyRollupConfig(prod bool, wasmModuleRoot common.Hash, rollupOwner common.Address, chainConfig *params.ChainConfig, serializedChainConfig []byte, loserStakeEscrow common.Address) rollup_legacy_gen.Config {
	var confirmPeriod uint64
	if prod {
		confirmPeriod = 45818
	} else {
		confirmPeriod = 20
	}
	return rollup_legacy_gen.Config{
		ConfirmPeriodBlocks:      confirmPeriod,
		ExtraChallengeTimeBlocks: 200,
		StakeToken:               common.Address{},
		BaseStake:                big.NewInt(params.Ether),
		WasmModuleRoot:           wasmModuleRoot,
		Owner:                    rollupOwner,
		LoserStakeEscrow:         loserStakeEscrow,
		ChainId:                  chainConfig.ChainID,
		// TODO could the ChainConfig be just []byte?
		ChainConfig: string(serializedChainConfig),
		SequencerInboxMaxTimeVariation: rollup_legacy_gen.ISequencerInboxMaxTimeVariation{
			DelayBlocks:   big.NewInt(60 * 60 * 24 / 15),
			FutureBlocks:  big.NewInt(12),
			DelaySeconds:  big.NewInt(60 * 60 * 24),
			FutureSeconds: big.NewInt(60 * 60),
		},
	}
}

func andTxSucceeded(ctx context.Context, parentChainReader *headerreader.HeaderReader, tx *types.Transaction, err error) error {
	if err != nil {
		return fmt.Errorf("error submitting tx: %w", err)
	}
	_, err = parentChainReader.WaitForTxApproval(ctx, tx)
	if err != nil {
		return fmt.Errorf("error executing tx: %w", err)
	}
	return nil
}

func deployBridgeCreator(ctx context.Context, parentChainReader *headerreader.HeaderReader, auth *bind.TransactOpts, maxDataSize *big.Int, chainSupportsBlobs bool) (common.Address, error) {
	client := parentChainReader.Client()

	/// deploy eth based templates
	bridgeTemplate, tx, _, err := bridge_legacy_gen.DeployBridge(auth, client)
	err = andTxSucceeded(ctx, parentChainReader, tx, err)
	if err != nil {
		return common.Address{}, fmt.Errorf("bridge deploy error: %w", err)
	}

	var reader4844 common.Address
	if chainSupportsBlobs {
		reader4844, tx, _, err = yulgen.DeployReader4844(auth, client)
		err = andTxSucceeded(ctx, parentChainReader, tx, err)
		if err != nil {
			return common.Address{}, fmt.Errorf("blob basefee reader deploy error: %w", err)
		}
	}
	seqInboxTemplateEthBased, tx, _, err := bridge_legacy_gen.DeploySequencerInbox(auth, client, maxDataSize, reader4844, false)
	err = andTxSucceeded(ctx, parentChainReader, tx, err)
	if err != nil {
		return common.Address{}, fmt.Errorf("sequencer inbox eth based deploy error: %w", err)
	}
	seqInboxTemplateERC20Based, tx, _, err := bridge_legacy_gen.DeploySequencerInbox(auth, client, maxDataSize, reader4844, true)
	err = andTxSucceeded(ctx, parentChainReader, tx, err)
	if err != nil {
		return common.Address{}, fmt.Errorf("sequencer inbox erc20 based deploy error: %w", err)
	}

	inboxTemplate, tx, _, err := bridge_legacy_gen.DeployInbox(auth, client, maxDataSize)
	err = andTxSucceeded(ctx, parentChainReader, tx, err)
	if err != nil {
		return common.Address{}, fmt.Errorf("inbox deploy error: %w", err)
	}

	rollupEventBridgeTemplate, tx, _, err := rollup_legacy_gen.DeployRollupEventInbox(auth, client)
	err = andTxSucceeded(ctx, parentChainReader, tx, err)
	if err != nil {
		return common.Address{}, fmt.Errorf("rollup event bridge deploy error: %w", err)
	}

	outboxTemplate, tx, _, err := bridge_legacy_gen.DeployOutbox(auth, client)
	err = andTxSucceeded(ctx, parentChainReader, tx, err)
	if err != nil {
		return common.Address{}, fmt.Errorf("outbox deploy error: %w", err)
	}

	ethBasedTemplates := rollup_legacy_gen.BridgeCreatorBridgeContracts{
		Bridge:           bridgeTemplate,
		SequencerInbox:   seqInboxTemplateEthBased,
		Inbox:            inboxTemplate,
		RollupEventInbox: rollupEventBridgeTemplate,
		Outbox:           outboxTemplate,
	}

	/// deploy ERC20 based templates
	erc20BridgeTemplate, tx, _, err := bridge_legacy_gen.DeployERC20Bridge(auth, client)
	err = andTxSucceeded(ctx, parentChainReader, tx, err)
	if err != nil {
		return common.Address{}, fmt.Errorf("bridge deploy error: %w", err)
	}

	erc20InboxTemplate, tx, _, err := bridge_legacy_gen.DeployERC20Inbox(auth, client, maxDataSize)
	err = andTxSucceeded(ctx, parentChainReader, tx, err)
	if err != nil {
		return common.Address{}, fmt.Errorf("inbox deploy error: %w", err)
	}

	erc20RollupEventBridgeTemplate, tx, _, err := rollup_legacy_gen.DeployERC20RollupEventInbox(auth, client)
	err = andTxSucceeded(ctx, parentChainReader, tx, err)
	if err != nil {
		return common.Address{}, fmt.Errorf("rollup event bridge deploy error: %w", err)
	}

	erc20OutboxTemplate, tx, _, err := bridge_legacy_gen.DeployERC20Outbox(auth, client)
	err = andTxSucceeded(ctx, parentChainReader, tx, err)
	if err != nil {
		return common.Address{}, fmt.Errorf("outbox deploy error: %w", err)
	}

	erc20BasedTemplates := rollup_legacy_gen.BridgeCreatorBridgeContracts{
		Bridge:           erc20BridgeTemplate,
		SequencerInbox:   seqInboxTemplateERC20Based,
		Inbox:            erc20InboxTemplate,
		RollupEventInbox: erc20RollupEventBridgeTemplate,
		Outbox:           erc20OutboxTemplate,
	}

	bridgeCreatorAddr, tx, _, err := rollup_legacy_gen.DeployBridgeCreator(auth, client, ethBasedTemplates, erc20BasedTemplates)
	err = andTxSucceeded(ctx, parentChainReader, tx, err)
	if err != nil {
		return common.Address{}, fmt.Errorf("bridge creator deploy error: %w", err)
	}

	return bridgeCreatorAddr, nil
}

func deployChallengeFactory(ctx context.Context, parentChainReader *headerreader.HeaderReader, auth *bind.TransactOpts) (common.Address, common.Address, error) {
	client := parentChainReader.Client()
	osp0, tx, _, err := osp_legacy_gen.DeployOneStepProver0(auth, client)
	err = andTxSucceeded(ctx, parentChainReader, tx, err)
	if err != nil {
		return common.Address{}, common.Address{}, fmt.Errorf("osp0 deploy error: %w", err)
	}

	ospMem, tx, _, err := osp_legacy_gen.DeployOneStepProverMemory(auth, client)
	err = andTxSucceeded(ctx, parentChainReader, tx, err)
	if err != nil {
		return common.Address{}, common.Address{}, fmt.Errorf("ospMemory deploy error: %w", err)
	}

	ospMath, tx, _, err := osp_legacy_gen.DeployOneStepProverMath(auth, client)
	err = andTxSucceeded(ctx, parentChainReader, tx, err)
	if err != nil {
		return common.Address{}, common.Address{}, fmt.Errorf("ospMath deploy error: %w", err)
	}

	ospHostIo, tx, _, err := osp_legacy_gen.DeployOneStepProverHostIo(auth, client)
	err = andTxSucceeded(ctx, parentChainReader, tx, err)
	if err != nil {
		return common.Address{}, common.Address{}, fmt.Errorf("ospHostIo deploy error: %w", err)
	}

	challengeManagerAddr, tx, _, err := challenge_legacy_gen.DeployChallengeManager(auth, client)
	err = andTxSucceeded(ctx, parentChainReader, tx, err)
	if err != nil {
		return common.Address{}, common.Address{}, fmt.Errorf("challenge manager deploy error: %w", err)
	}

	ospEntryAddr, tx, _, err := osp_legacy_gen.DeployOneStepProofEntry(auth, client, osp0, ospMem, ospMath, ospHostIo)
	err = andTxSucceeded(ctx, parentChainReader, tx, err)
	if err != nil {
		return common.Address{}, common.Address{}, fmt.Errorf("ospEntry deploy error: %w", err)
	}

	return ospEntryAddr, challengeManagerAddr, nil
}

func deployRollupCreator(ctx context.Context, parentChainReader *headerreader.HeaderReader, auth *bind.TransactOpts, maxDataSize *big.Int, chainSupportsBlobs bool) (*rollup_legacy_gen.RollupCreator, common.Address, common.Address, common.Address, error) {
	bridgeCreator, err := deployBridgeCreator(ctx, parentChainReader, auth, maxDataSize, chainSupportsBlobs)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, fmt.Errorf("bridge creator deploy error: %w", err)
	}

	ospEntryAddr, challengeManagerAddr, err := deployChallengeFactory(ctx, parentChainReader, auth)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, err
	}

	rollupAdminLogic, tx, _, err := rollup_legacy_gen.DeployRollupAdminLogic(auth, parentChainReader.Client())
	err = andTxSucceeded(ctx, parentChainReader, tx, err)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, fmt.Errorf("rollup admin logic deploy error: %w", err)
	}

	rollupUserLogic, tx, _, err := rollup_legacy_gen.DeployRollupUserLogic(auth, parentChainReader.Client())
	err = andTxSucceeded(ctx, parentChainReader, tx, err)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, fmt.Errorf("rollup user logic deploy error: %w", err)
	}

	rollupCreatorAddress, tx, rollupCreator, err := rollup_legacy_gen.DeployRollupCreator(auth, parentChainReader.Client())
	err = andTxSucceeded(ctx, parentChainReader, tx, err)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, fmt.Errorf("rollup creator deploy error: %w", err)
	}

	upgradeExecutor, tx, _, err := upgrade_executorgen.DeployUpgradeExecutor(auth, parentChainReader.Client())
	err = andTxSucceeded(ctx, parentChainReader, tx, err)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, fmt.Errorf("upgrade executor deploy error: %w", err)
	}

	validatorUtils, tx, _, err := rollup_legacy_gen.DeployValidatorUtils(auth, parentChainReader.Client())
	err = andTxSucceeded(ctx, parentChainReader, tx, err)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, fmt.Errorf("validator utils deploy error: %w", err)
	}

	validatorWalletCreator, tx, _, err := rollup_legacy_gen.DeployValidatorWalletCreator(auth, parentChainReader.Client())
	err = andTxSucceeded(ctx, parentChainReader, tx, err)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, fmt.Errorf("validator wallet creator deploy error: %w", err)
	}

	l2FactoriesDeployHelper, tx, _, err := rollup_legacy_gen.DeployDeployHelper(auth, parentChainReader.Client())
	err = andTxSucceeded(ctx, parentChainReader, tx, err)
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
	err = andTxSucceeded(ctx, parentChainReader, tx, err)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, fmt.Errorf("rollup set template error: %w", err)
	}

	return rollupCreator, rollupCreatorAddress, validatorUtils, validatorWalletCreator, nil
}

func DeployLegacyOnParentChain(ctx context.Context, parentChainReader *headerreader.HeaderReader, deployAuth *bind.TransactOpts, batchPosters []common.Address, batchPosterManager common.Address, authorizeValidators uint64, config rollup_legacy_gen.Config, nativeToken common.Address, maxDataSize *big.Int, chainSupportsBlobs bool) (*chaininfo.RollupAddresses, error) {
	if config.WasmModuleRoot == (common.Hash{}) {
		return nil, errors.New("no machine specified")
	}

	rollupCreator, _, validatorUtils, validatorWalletCreator, err := deployRollupCreator(ctx, parentChainReader, deployAuth, maxDataSize, chainSupportsBlobs)
	if err != nil {
		return nil, fmt.Errorf("error deploying rollup creator: %w", err)
	}

	var validatorAddrs []common.Address
	for i := uint64(1); i <= authorizeValidators; i++ {
		validatorAddrs = append(validatorAddrs, crypto.CreateAddress(validatorWalletCreator, i))
	}

	deployParams := rollup_legacy_gen.RollupCreatorRollupDeploymentParams{
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
