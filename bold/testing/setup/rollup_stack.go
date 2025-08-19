// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// Package setup prepares a simulated backend for testing.
package setup

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient/simulated"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	protocol "github.com/offchainlabs/bold/chain-abstraction"
	solimpl "github.com/offchainlabs/bold/chain-abstraction/sol-implementation"
	l2stateprovider "github.com/offchainlabs/bold/layer2-state-provider"
	retry "github.com/offchainlabs/bold/runtime"
	challenge_testing "github.com/offchainlabs/bold/testing"
	statemanager "github.com/offchainlabs/bold/testing/mocks/state-provider"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/solgen/go/challengeV2gen"
	"github.com/offchainlabs/nitro/solgen/go/contractsgen"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/solgen/go/ospgen"
	"github.com/offchainlabs/nitro/solgen/go/proxiesgen"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
	"github.com/offchainlabs/nitro/solgen/go/yulgen"
)

type Committer interface {
	Commit() common.Hash
}

type CreatedValidatorFork struct {
	Leaf1              protocol.Assertion
	Leaf2              protocol.Assertion
	Chains             []*solimpl.AssertionChain
	Accounts           []*TestAccount
	Backend            *SimulatedBackendWrapper
	HonestStateManager l2stateprovider.Provider
	EvilStateManager   l2stateprovider.Provider
	Addrs              *RollupAddresses
}

type CreateForkConfig struct {
	DivergeBlockHeight    uint64
	BlockHeightDifference int64
	DivergeMachineHeight  uint64
	BlockChallengeHeight  uint64
}

func CreateTwoValidatorFork(
	ctx context.Context,
	t testing.TB,
	cfg *CreateForkConfig,
	opts ...Opt,
) (*CreatedValidatorFork, error) {
	t.Helper()
	setup, err := ChainsWithEdgeChallengeManager(opts...)
	if err != nil {
		return nil, err
	}

	// Advance the backend by some blocks to get over time delta errors when
	// using the assertion chain.
	for i := 0; i < 100; i++ {
		setup.Backend.Commit()
	}

	genesisHash, err := setup.Chains[1].GenesisAssertionHash(ctx)
	if err != nil {
		return nil, err
	}
	genesisCreationInfo, err := setup.Chains[1].ReadAssertionCreationInfo(ctx, protocol.AssertionHash{Hash: genesisHash})
	if err != nil {
		return nil, err
	}

	honestStateManager, err := statemanager.NewForSimpleMachine(t, setup.StateManagerOpts...)
	if err != nil {
		return nil, err
	}

	// Set defaults (zeroes are not valid here)
	if cfg.DivergeBlockHeight == 0 {
		cfg.DivergeBlockHeight = 1
	}
	if cfg.DivergeMachineHeight == 0 {
		cfg.DivergeMachineHeight = 1
	}
	if cfg.BlockChallengeHeight == 0 {
		cfg.BlockChallengeHeight = 1 << 5
	}

	stateManagerOpts := setup.StateManagerOpts
	stateManagerOpts = append(
		stateManagerOpts,
		statemanager.WithBlockDivergenceHeight(cfg.DivergeBlockHeight),
		statemanager.WithDivergentBlockHeightOffset(cfg.BlockHeightDifference),
		statemanager.WithMachineDivergenceStep(cfg.DivergeMachineHeight),
	)
	evilStateManager, err := statemanager.NewForSimpleMachine(t, stateManagerOpts...)
	if err != nil {
		return nil, err
	}
	genesis, err := honestStateManager.ExecutionStateAfterPreviousState(ctx, 0, protocol.GoGlobalState{})
	if err != nil {
		return nil, err
	}
	honestPostState, err := honestStateManager.ExecutionStateAfterPreviousState(ctx, 1, genesis.GlobalState)
	if err != nil {
		return nil, err
	}
	assertion, err := setup.Chains[0].NewStakeOnNewAssertion(
		ctx,
		genesisCreationInfo,
		honestPostState,
	)
	if err != nil {
		return nil, err
	}

	genesis, err = evilStateManager.ExecutionStateAfterPreviousState(ctx, 0, protocol.GoGlobalState{})
	if err != nil {
		return nil, err
	}
	evilPostState, err := evilStateManager.ExecutionStateAfterPreviousState(ctx, 1, genesis.GlobalState)
	if err != nil {
		return nil, err
	}
	forkedAssertion, err := setup.Chains[1].NewStakeOnNewAssertion(
		ctx,
		genesisCreationInfo,
		evilPostState,
	)
	if err != nil {
		return nil, err
	}

	return &CreatedValidatorFork{
		Leaf1:              assertion,
		Leaf2:              forkedAssertion,
		Chains:             setup.Chains,
		Accounts:           setup.Accounts,
		Backend:            setup.Backend,
		Addrs:              setup.Addrs,
		HonestStateManager: honestStateManager,
		EvilStateManager:   evilStateManager,
	}, nil
}

type ChainSetup struct {
	Chains                     []*solimpl.AssertionChain
	Accounts                   []*TestAccount
	Addrs                      *RollupAddresses
	Backend                    *SimulatedBackendWrapper
	RollupConfig               rollupgen.Config
	useMockBridge              bool
	useMockOneStepProver       bool
	numAccountsToGen           uint64
	numFundedAccounts          uint64
	minimumAssertionPeriod     int64
	autoDeposit                bool
	challengeTestingOpts       []challenge_testing.Opt
	StateManagerOpts           []statemanager.Opt
	StakeTokenAddress          common.Address
	EnableFastConfirmation     bool
	EnableSafeFastConfirmation bool
}

type Opt func(setup *ChainSetup)

func WithMockOneStepProver() Opt {
	return func(setup *ChainSetup) {
		setup.useMockOneStepProver = true
	}
}

func WithFastConfirmation() Opt {
	return func(setup *ChainSetup) {
		setup.EnableFastConfirmation = true
	}
}

func WithSafeFastConfirmation() Opt {
	return func(setup *ChainSetup) {
		setup.EnableSafeFastConfirmation = true
	}
}

func WithMockBridge() Opt {
	return func(setup *ChainSetup) {
		setup.useMockBridge = false
	}
}

func WithMinimumAssertionPeriod(period int64) Opt {
	return func(setup *ChainSetup) {
		setup.minimumAssertionPeriod = period
	}
}

func WithChallengeTestingOpts(opts ...challenge_testing.Opt) Opt {
	return func(setup *ChainSetup) {
		setup.challengeTestingOpts = opts
	}
}

func WithStateManagerOpts(opts ...statemanager.Opt) Opt {
	return func(setup *ChainSetup) {
		setup.StateManagerOpts = opts
	}
}

func WithNumAccounts(n uint64) Opt {
	return func(setup *ChainSetup) {
		setup.numAccountsToGen = n
	}
}

func WithNumFundedAccounts(n uint64) Opt {
	return func(setup *ChainSetup) {
		setup.numFundedAccounts = n
	}
}

func WithAutoDeposit() Opt {
	return func(setup *ChainSetup) {
		setup.autoDeposit = true
	}
}

func ChainsWithEdgeChallengeManager(opts ...Opt) (*ChainSetup, error) {
	ctx := context.Background()
	setp := &ChainSetup{
		numAccountsToGen: 4,
		autoDeposit:      false,
	}
	for _, o := range opts {
		o(setp)
	}
	if setp.numAccountsToGen < 3 {
		setp.numAccountsToGen = 3
	}
	accs, backend, err := Accounts(setp.numAccountsToGen)
	if err != nil {
		return nil, err
	}
	stakeToken, tx, tokenBindings, err := mocksgen.DeployTestWETH9(
		accs[0].TxOpts,
		backend,
		"Weth",
		"WETH",
	)
	if err != nil {
		return nil, err
	}
	if waitErr := challenge_testing.WaitForTx(ctx, backend, tx); waitErr != nil {
		return nil, errors.Wrap(waitErr, "errored waiting for transaction")
	}
	receipt, err := backend.TransactionReceipt(ctx, tx.Hash())
	if err != nil {
		return nil, err
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return nil, errors.New("receipt not successful")
	}
	value, ok := new(big.Int).SetString("10000000000000000000000", 10)
	if !ok {
		return nil, errors.New("could not set value")
	}
	if !setp.autoDeposit {
		accs[0].TxOpts.Value = value
		mintTx, err3 := tokenBindings.Deposit(accs[0].TxOpts)
		if err3 != nil {
			return nil, err3
		}
		if waitErr := challenge_testing.WaitForTx(ctx, backend, mintTx); waitErr != nil {
			return nil, errors.Wrap(waitErr, "errored waiting for transaction")
		}
		receipt, err = backend.TransactionReceipt(ctx, mintTx.Hash())
		if err != nil {
			return nil, err
		}
		if receipt.Status != types.ReceiptStatusSuccessful {
			return nil, errors.New("receipt not successful")
		}
		accs[0].TxOpts.Value = big.NewInt(0)
	}

	prod := false
	wasmModuleRoot := common.Hash{}
	rollupOwner := accs[0].AccountAddr
	chainId := big.NewInt(1337)
	loserStakeEscrow := rollupOwner
	cfgOpts := &rollupgen.Config{}
	for _, o := range setp.challengeTestingOpts {
		o(cfgOpts)
	}
	if setp.EnableFastConfirmation {
		cfgOpts.AnyTrustFastConfirmer = accs[1].AccountAddr
	}
	var safeProxyAddress common.Address
	if setp.EnableSafeFastConfirmation {
		var safeAddress common.Address
		safeAddress, err = retry.UntilSucceeds(ctx, func() (common.Address, error) {
			safeAddress, tx, _, err = contractsgen.DeploySafeL2(accs[0].TxOpts, backend)
			if err != nil {
				return common.Address{}, err
			}
			err = challenge_testing.TxSucceeded(ctx, tx, safeAddress, backend, err)
			if err != nil {
				return common.Address{}, err
			}
			return safeAddress, nil
		})
		if err != nil {
			return nil, err
		}

		safeProxyAddress, err = retry.UntilSucceeds(ctx, func() (common.Address, error) {
			safeProxyAddress, tx, _, err = proxiesgen.DeploySafeProxy(accs[0].TxOpts, backend, safeAddress)
			if err != nil {
				return common.Address{}, err
			}
			err = challenge_testing.TxSucceeded(ctx, tx, safeProxyAddress, backend, err)
			if err != nil {
				return common.Address{}, err
			}
			return safeProxyAddress, nil
		})
		if err != nil {
			return nil, err
		}
		var safe *contractsgen.Safe
		safe, err = contractsgen.NewSafe(safeProxyAddress, backend)
		if err != nil {
			return nil, err
		}
		tx, err = safe.Setup(
			accs[0].TxOpts,
			[]common.Address{accs[1].AccountAddr, accs[2].AccountAddr, accs[3].AccountAddr},
			big.NewInt(2),
			common.Address{},
			nil,
			common.Address{},
			common.Address{},
			big.NewInt(0),
			common.Address{},
		)
		if err != nil {
			return nil, err
		}
		if waitErr := challenge_testing.WaitForTx(ctx, backend, tx); waitErr != nil {
			return nil, errors.Wrap(waitErr, "errored waiting for transaction")
		}
		cfgOpts.AnyTrustFastConfirmer = safeProxyAddress
	}
	numLevels := cfgOpts.NumBigStepLevel + 2
	if numLevels == 2 {
		numLevels = 3
	}
	miniStakeValues := make([]*big.Int, numLevels)
	for i := 1; i <= int(numLevels); i++ {
		miniStakeValues[i-1] = big.NewInt(int64(i))
	}
	genesisExecutionState := rollupgen.AssertionState{
		GlobalState:   rollupgen.GlobalState{},
		MachineStatus: 1,
	}
	genesisInboxCount := big.NewInt(0)
	anyTrustFastConfirmer := cfgOpts.AnyTrustFastConfirmer
	cfg := challenge_testing.GenerateRollupConfig(
		prod,
		wasmModuleRoot,
		rollupOwner,
		chainId,
		loserStakeEscrow,
		miniStakeValues,
		stakeToken,
		genesisExecutionState,
		genesisInboxCount,
		anyTrustFastConfirmer,
		setp.challengeTestingOpts...,
	)
	addresses, err := DeployFullRollupStack(
		ctx,
		backend,
		accs[0].TxOpts,
		accs[0].TxOpts.From, // Sequencer addr.
		cfg,
		RollupStackConfig{
			UseMockBridge:          setp.useMockBridge,
			UseMockOneStepProver:   setp.useMockOneStepProver,
			UseBlobs:               true,
			MinimumAssertionPeriod: setp.minimumAssertionPeriod,
		},
	)
	if err != nil {
		return nil, err
	}

	chains := make([]*solimpl.AssertionChain, 0)
	for _, acc := range accs[1:] {
		var assertionChainBinding *rollupgen.RollupUserLogic
		assertionChainBinding, err = rollupgen.NewRollupUserLogic(
			addresses.Rollup, backend,
		)
		if err != nil {
			return nil, err
		}
		var challengeManagerAddr common.Address
		challengeManagerAddr, err = assertionChainBinding.ChallengeManager(
			&bind.CallOpts{Context: ctx},
		)
		if err != nil {
			return nil, err
		}
		assertionChainOpts := []solimpl.Opt{
			solimpl.WithRpcHeadBlockNumber(rpc.LatestBlockNumber),
		}
		if setp.EnableSafeFastConfirmation || (setp.EnableFastConfirmation && acc.AccountAddr == cfgOpts.AnyTrustFastConfirmer) {
			assertionChainOpts = append(assertionChainOpts, solimpl.WithFastConfirmation())
		}
		chain, chainErr := solimpl.NewAssertionChain(
			ctx,
			addresses.Rollup,
			challengeManagerAddr,
			acc.TxOpts,
			backend,
			solimpl.NewChainBackendTransactor(backend),
			assertionChainOpts...,
		)
		if chainErr != nil {
			return nil, chainErr
		}
		chains = append(chains, chain)
	}
	chalManager := chains[1].SpecChallengeManager()
	chalManagerAddr := chalManager.Address()
	seed, ok := new(big.Int).SetString("10000", 10)
	if !ok {
		return nil, errors.New("could not set big int")
	}
	if !setp.autoDeposit {
		for i := 0; i < len(accs); i++ {
			acc := accs[i]
			transferTx, err := tokenBindings.Transfer(accs[0].TxOpts, acc.TxOpts.From, seed)
			if err != nil {
				return nil, errors.Wrap(err, "could not approve account")
			}
			if waitErr := challenge_testing.WaitForTx(ctx, backend, transferTx); waitErr != nil {
				return nil, errors.Wrap(waitErr, "errored waiting for transfer transaction")
			}
			receipt, err := backend.TransactionReceipt(ctx, transferTx.Hash())
			if err != nil {
				return nil, errors.Wrap(err, "could not get tx receipt")
			}
			if receipt.Status != types.ReceiptStatusSuccessful {
				return nil, errors.New("receipt not successful")
			}
			approveTx, err := tokenBindings.Approve(acc.TxOpts, addresses.Rollup, value)
			if err != nil {
				return nil, errors.Wrap(err, "could not approve account")
			}
			if waitErr := challenge_testing.WaitForTx(ctx, backend, approveTx); waitErr != nil {
				return nil, errors.Wrap(waitErr, "errored waiting for approval transaction")
			}
			receipt, err = backend.TransactionReceipt(ctx, approveTx.Hash())
			if err != nil {
				return nil, errors.Wrap(err, "could not get tx receipt")
			}
			if receipt.Status != types.ReceiptStatusSuccessful {
				return nil, errors.New("receipt not successful")
			}
			approveTx, err = tokenBindings.Approve(acc.TxOpts, chalManagerAddr, value)
			if err != nil {
				return nil, errors.Wrap(err, "could not approve account")
			}
			if waitErr := challenge_testing.WaitForTx(ctx, backend, approveTx); waitErr != nil {
				return nil, errors.Wrap(waitErr, "errored waiting for approval transaction")
			}
			receipt, err = backend.TransactionReceipt(ctx, approveTx.Hash())
			if err != nil {
				return nil, errors.Wrap(err, "could not get tx receipt")
			}
			if receipt.Status != types.ReceiptStatusSuccessful {
				return nil, errors.New("receipt not successful")
			}
		}
	}

	setp.Chains = chains
	setp.Accounts = accs
	setp.Addrs = addresses
	setp.Backend = backend
	setp.RollupConfig = cfg
	setp.StakeTokenAddress = stakeToken
	return setp, nil
}

type RollupAddresses struct {
	Bridge                 common.Address `json:"bridge"`
	Inbox                  common.Address `json:"inbox"`
	SequencerInbox         common.Address `json:"sequencer-inbox"`
	Rollup                 common.Address `json:"rollup"`
	RollupUserLogic        common.Address `json:"rollup-user-logic"`
	ValidatorUtils         common.Address `json:"validator-utils"`
	ValidatorWalletCreator common.Address `json:"validator-wallet-creator"`
	UpgradeExecutor        common.Address `json:"upgrade-executor"`
	DeployedAt             uint64         `json:"deployed-at"`
}

type RollupStackConfig struct {
	UseMockBridge          bool
	UseMockOneStepProver   bool
	UseBlobs               bool
	MinimumAssertionPeriod int64
}

func DeployFullRollupStack(
	ctx context.Context,
	backend protocol.ChainBackend,
	deployAuth *bind.TransactOpts,
	sequencer common.Address,
	config rollupgen.Config,
	stackConf RollupStackConfig,
) (*RollupAddresses, error) {
	log.Info("Deploying rollup creator")
	rollupCreator, rollupUserAddr, rollupCreatorAddress, validatorUtils, validatorWalletCreator, err := deployRollupCreator(ctx, backend, deployAuth, stackConf.UseBlobs, stackConf.UseMockBridge, stackConf.UseMockOneStepProver)
	if err != nil {
		return nil, err
	}

	log.Info("Creating rollup")
	tx, err := retry.UntilSucceeds(ctx, func() (*types.Transaction, error) {
		creationTx, creationErr := rollupCreator.CreateRollup(
			deployAuth,
			rollupgen.RollupCreatorRollupDeploymentParams{
				Config:                    config,
				Validators:                []common.Address{},
				MaxDataSize:               big.NewInt(challenge_testing.MaxDataSize),
				NativeToken:               common.Address{},
				DeployFactoriesToL2:       false,
				MaxFeePerGasForRetryables: big.NewInt(0),
				BatchPosters:              []common.Address{},
				BatchPosterManager:        common.Address{},
			},
		)
		if creationErr != nil {
			fmt.Println(creationErr)
			return nil, creationErr
		}
		err = challenge_testing.TxSucceeded(ctx, creationTx, rollupCreatorAddress, backend, err)
		if err != nil {
			return nil, err
		}
		return creationTx, nil
	})
	if err != nil {
		return nil, err
	}

	creationReceipt, err := backend.TransactionReceipt(ctx, tx.Hash())
	if err != nil {
		return nil, err
	}
	info, err := rollupCreator.ParseRollupCreated(*creationReceipt.Logs[len(creationReceipt.Logs)-1])
	if err != nil {
		return nil, err
	}

	upgradeExecBindings, err := mocksgen.NewUpgradeExecutorMock(info.UpgradeExecutor, backend)
	if err != nil {
		return nil, err
	}

	rollupABI, err := abi.JSON(strings.NewReader(rollupgen.RollupAdminLogicABI))
	if err != nil {
		return nil, err
	}
	setWhitelistDisabled, err := rollupABI.Pack("setValidatorWhitelistDisabled", true)
	if err != nil {
		return nil, err
	}
	setMinimumAssertionPeriod, err := rollupABI.Pack("setMinimumAssertionPeriod", big.NewInt(stackConf.MinimumAssertionPeriod))
	if err != nil {
		return nil, err
	}
	seqInboxABI, err := abi.JSON(strings.NewReader(bridgegen.SequencerInboxABI))
	if err != nil {
		return nil, err
	}
	setBatchPosterManager, err := seqInboxABI.Pack("setBatchPosterManager", deployAuth.From)
	if err != nil {
		return nil, err
	}
	txs := map[common.Address][][]byte{
		info.RollupAddress:  {setWhitelistDisabled, setMinimumAssertionPeriod},
		info.SequencerInbox: {setBatchPosterManager},
	}
	// if a zero sequencer address is specified, don't authorize any sequencers
	if sequencer != (common.Address{}) {
		setIsBatchPoster, err2 := seqInboxABI.Pack("setIsBatchPoster", sequencer, true)
		if err2 != nil {
			return nil, err2
		}
		txs[info.SequencerInbox] = append(txs[info.SequencerInbox], setIsBatchPoster)
	}
	for addr, items := range txs {
		for _, item := range items {
			_, err = retry.UntilSucceeds(ctx, func() (*types.Transaction, error) {
				innerTx, err2 := upgradeExecBindings.ExecuteCall(deployAuth, addr, item)
				if err2 != nil {
					return nil, err2
				}
				if waitErr := challenge_testing.WaitForTx(ctx, backend, innerTx); waitErr != nil {
					return nil, errors.Wrap(waitErr, "errored waiting for UpgradeExecutor transaction")
				}
				return innerTx, nil
			})
			if err != nil {
				return nil, err
			}
		}
	}
	if committer, ok := backend.(Committer); ok {
		committer.Commit()
	}
	if !creationReceipt.BlockNumber.IsUint64() {
		return nil, errors.New("block number was not a uint64")
	}
	log.Info("Done deploying")

	return &RollupAddresses{
		Bridge:                 info.Bridge,
		Inbox:                  info.InboxAddress,
		SequencerInbox:         info.SequencerInbox,
		DeployedAt:             creationReceipt.BlockNumber.Uint64(),
		Rollup:                 info.RollupAddress,
		RollupUserLogic:        rollupUserAddr,
		ValidatorUtils:         validatorUtils,
		ValidatorWalletCreator: validatorWalletCreator,
		UpgradeExecutor:        info.UpgradeExecutor,
	}, nil
}

func deployBridgeCreator(
	ctx context.Context,
	auth *bind.TransactOpts,
	useBlobs bool,
	backend protocol.ChainBackend,
	useMockBridge bool,
) (common.Address, error) {
	var bridgeTemplate common.Address
	var err error
	if useMockBridge {
		bridgeTemplate, _, _, err = mocksgen.DeployBridgeStub(auth, backend)
		if err != nil {
			return common.Address{}, err
		}
	} else {
		log.Info("Deploying bridge template")
		bridgeTemplate, err = retry.UntilSucceeds(ctx, func() (common.Address, error) {
			bridgeTemplateAddr, tx, _, err2 := bridgegen.DeployBridge(auth, backend)
			if err2 != nil {
				return common.Address{}, err2
			}
			err2 = challenge_testing.TxSucceeded(ctx, tx, bridgeTemplateAddr, backend, err2)
			if err2 != nil {
				return common.Address{}, errors.Wrap(err2, "bridgegen.DeployBridge")
			}
			return bridgeTemplateAddr, nil
		})
		if err != nil {
			return common.Address{}, err
		}
	}

	var dataHashesReader common.Address
	if useBlobs {
		reader, err2 := retry.UntilSucceeds(ctx, func() (common.Address, error) {
			readerAddr, tx, _, err3 := yulgen.DeployReader4844(auth, backend)
			if err3 != nil {
				return common.Address{}, err3
			}
			err3 = challenge_testing.TxSucceeded(ctx, tx, readerAddr, backend, err3)
			if err3 != nil {
				return common.Address{}, errors.Wrap(err3, "yulgen.DeployReader4844")
			}
			return readerAddr, nil
		})
		if err2 != nil {
			return common.Address{}, err2
		}
		dataHashesReader = reader
	}

	maxDataSize := big.NewInt(challenge_testing.MaxDataSize)
	log.Info("Deploying seq inbox")
	seqInboxTemplate, err := retry.UntilSucceeds(ctx, func() (common.Address, error) {
		seqInboxTemplateAddr, tx, _, err2 := bridgegen.DeploySequencerInbox(auth, backend, maxDataSize, dataHashesReader, false /* no fee token */, false /* disable delay buffer */)
		if err2 != nil {
			return common.Address{}, err2
		}
		err2 = challenge_testing.TxSucceeded(ctx, tx, seqInboxTemplateAddr, backend, err2)
		if err2 != nil {
			return common.Address{}, errors.Wrap(err2, "bridgegen.DeploySequencerInbox")
		}
		return seqInboxTemplateAddr, nil
	})
	if err != nil {
		return common.Address{}, err
	}

	log.Info("Deploying seq inbox bufferable")
	seqInboxBufferableTemplate, err := retry.UntilSucceeds(ctx, func() (common.Address, error) {
		seqInboxTemplateAddr, tx, _, err2 := bridgegen.DeploySequencerInbox(auth, backend, maxDataSize, dataHashesReader, false /* no fee token */, true /* enable delay buffer */)
		if err2 != nil {
			return common.Address{}, err2
		}
		err2 = challenge_testing.TxSucceeded(ctx, tx, seqInboxTemplateAddr, backend, err2)
		if err2 != nil {
			return common.Address{}, errors.Wrap(err2, "bridgegen.DeploySequencerInbox")
		}
		return seqInboxTemplateAddr, nil
	})
	if err != nil {
		return common.Address{}, err
	}

	log.Info("Deploying inbox")
	inboxTemplate, err := retry.UntilSucceeds(ctx, func() (common.Address, error) {
		inboxTemplateAddr, tx, _, err2 := bridgegen.DeployInbox(auth, backend, maxDataSize)
		if err2 != nil {
			return common.Address{}, err2
		}
		err2 = challenge_testing.TxSucceeded(ctx, tx, inboxTemplateAddr, backend, err2)
		if err2 != nil {
			return common.Address{}, errors.Wrap(err2, "bridgegen.DeployInbox")
		}
		return inboxTemplateAddr, nil
	})
	if err != nil {
		return common.Address{}, err
	}

	log.Info("Deploying event bridge")
	rollupEventBridgeTemplate, err := retry.UntilSucceeds(ctx, func() (common.Address, error) {
		rollupEventBridgeTemplateAddr, tx, _, err2 := mocksgen.DeployMockRollupEventInbox(auth, backend)
		if err2 != nil {
			return common.Address{}, err2
		}
		err2 = challenge_testing.TxSucceeded(ctx, tx, rollupEventBridgeTemplateAddr, backend, err2)
		if err2 != nil {
			return common.Address{}, errors.Wrap(err2, "rollupgen.DeployRollupEventInbox")
		}
		return rollupEventBridgeTemplateAddr, nil
	})
	if err != nil {
		return common.Address{}, err
	}

	log.Info("Deploying outbox")
	outboxTemplate, err := retry.UntilSucceeds(ctx, func() (common.Address, error) {
		outboxTemplateAddr, tx, _, err2 := bridgegen.DeployOutbox(auth, backend)
		if err2 != nil {
			return common.Address{}, err
		}
		err2 = challenge_testing.TxSucceeded(ctx, tx, outboxTemplateAddr, backend, err2)
		if err2 != nil {
			return common.Address{}, errors.Wrap(err2, "bridgegen.DeployOutbox")
		}
		return outboxTemplateAddr, nil
	})
	if err != nil {
		return common.Address{}, err
	}

	ethTemplates := rollupgen.BridgeCreatorBridgeTemplates{
		SequencerInbox:                seqInboxTemplate,
		Bridge:                        bridgeTemplate,
		Inbox:                         inboxTemplate,
		RollupEventInbox:              rollupEventBridgeTemplate,
		Outbox:                        outboxTemplate,
		DelayBufferableSequencerInbox: seqInboxBufferableTemplate,
	}

	/// deploy ERC20 based templates
	erc20BridgeTemplate, err := retry.UntilSucceeds(ctx, func() (common.Address, error) {
		addr, _, _, innerErr := bridgegen.DeployERC20Bridge(auth, backend)
		return addr, innerErr
	})
	if err != nil {
		return common.Address{}, err
	}

	erc20InboxTemplate, err := retry.UntilSucceeds(ctx, func() (common.Address, error) {
		addr, _, _, innerErr := bridgegen.DeployERC20Inbox(auth, backend, maxDataSize)
		return addr, innerErr
	})
	if err != nil {
		return common.Address{}, err
	}

	erc20RollupEventBridgeTemplate, err := retry.UntilSucceeds(ctx, func() (common.Address, error) {
		addr, _, _, innerErr := rollupgen.DeployERC20RollupEventInbox(auth, backend)
		return addr, innerErr
	})
	if err != nil {
		return common.Address{}, err
	}

	erc20OutboxTemplate, err := retry.UntilSucceeds(ctx, func() (common.Address, error) {
		addr, _, _, innerErr := bridgegen.DeployERC20Outbox(auth, backend)
		return addr, innerErr
	})
	if err != nil {
		return common.Address{}, err
	}

	erc20Templates := rollupgen.BridgeCreatorBridgeTemplates{
		Bridge:                        erc20BridgeTemplate,
		SequencerInbox:                seqInboxTemplate,
		Inbox:                         erc20InboxTemplate,
		RollupEventInbox:              erc20RollupEventBridgeTemplate,
		Outbox:                        erc20OutboxTemplate,
		DelayBufferableSequencerInbox: seqInboxBufferableTemplate,
	}

	type bridgeCreationResult struct {
		bridgeCreatorAddr common.Address
		bridgeCreator     *rollupgen.BridgeCreator
	}
	log.Info("Deploying bridge creator itself")
	result, err := retry.UntilSucceeds(ctx, func() (*bridgeCreationResult, error) {
		bridgeCreatorAddr, tx, bridgeCreator, err2 := rollupgen.DeployBridgeCreator(auth, backend, ethTemplates, erc20Templates)
		if err2 != nil {
			return nil, err2
		}
		err2 = challenge_testing.TxSucceeded(ctx, tx, bridgeCreatorAddr, backend, err2)
		if err2 != nil {
			return nil, err2
		}
		return &bridgeCreationResult{
			bridgeCreatorAddr: bridgeCreatorAddr,
			bridgeCreator:     bridgeCreator,
		}, nil
	})
	if err != nil {
		return common.Address{}, err
	}

	log.Info("Updating bridge creator templates")
	_, err = retry.UntilSucceeds(ctx, func() (*types.Transaction, error) {
		tx, err2 := result.bridgeCreator.UpdateTemplates(auth, ethTemplates)
		if err2 != nil {
			return nil, err2
		}
		err2 = challenge_testing.TxSucceeded(ctx, tx, result.bridgeCreatorAddr, backend, err2)
		if err2 != nil {
			return nil, err2
		}
		return tx, nil
	})
	if err != nil {
		return common.Address{}, err
	}
	_, err = retry.UntilSucceeds(ctx, func() (*types.Transaction, error) {
		tx, err2 := result.bridgeCreator.UpdateERC20Templates(auth, erc20Templates)
		if err2 != nil {
			return nil, err2
		}
		err2 = challenge_testing.TxSucceeded(ctx, tx, result.bridgeCreatorAddr, backend, err2)
		if err2 != nil {
			return nil, err2
		}
		return tx, nil
	})
	if err != nil {
		return common.Address{}, err
	}
	return result.bridgeCreatorAddr, nil
}

func deployChallengeFactory(
	ctx context.Context,
	auth *bind.TransactOpts,
	backend protocol.ChainBackend,
	useMockOneStepProver bool,
) (common.Address, common.Address, error) {
	var ospEntryAddr common.Address
	if useMockOneStepProver {
		ospEntry, tx, _, err := mocksgen.DeploySimpleOneStepProofEntry(auth, backend)
		if waitErr := challenge_testing.WaitForTx(ctx, backend, tx); waitErr != nil {
			return common.Address{}, common.Address{}, errors.Wrap(err, "mocksgen.DeployMockOneStepProofEntry")
		}
		ospEntryAddr = ospEntry
	} else {
		log.Info("Deploying osp0")
		osp0, err := retry.UntilSucceeds(ctx, func() (common.Address, error) {
			osp0Addr, _, _, err2 := ospgen.DeployOneStepProver0(auth, backend)
			if err2 != nil {
				return common.Address{}, err2
			}
			return osp0Addr, nil
		})
		if err != nil {
			return common.Address{}, common.Address{}, err
		}
		log.Info("Deploying ospMem")
		ospMem, err := retry.UntilSucceeds(ctx, func() (common.Address, error) {
			ospMemAddr, _, _, err2 := ospgen.DeployOneStepProverMemory(auth, backend)
			if err2 != nil {
				return common.Address{}, err2
			}
			return ospMemAddr, nil
		})
		if err != nil {
			return common.Address{}, common.Address{}, err
		}
		log.Info("Deploying ospMath")
		ospMath, err := retry.UntilSucceeds(ctx, func() (common.Address, error) {
			ospMathAddr, _, _, err2 := ospgen.DeployOneStepProverMath(auth, backend)
			if err2 != nil {
				return common.Address{}, err2
			}
			return ospMathAddr, nil
		})
		if err != nil {
			return common.Address{}, common.Address{}, err
		}
		log.Info("Deploying ospHostIo")
		ospHostIo, err := retry.UntilSucceeds(ctx, func() (common.Address, error) {
			ospHostIoAddr, _, _, err2 := ospgen.DeployOneStepProverHostIo(auth, backend)
			if err2 != nil {
				return common.Address{}, err2
			}
			return ospHostIoAddr, nil
		})
		if err != nil {
			return common.Address{}, common.Address{}, err
		}
		log.Info("Deploying ospEntry")
		ospEntry, err := retry.UntilSucceeds(ctx, func() (common.Address, error) {
			ospEntryAddr2, _, _, err2 := ospgen.DeployOneStepProofEntry(auth, backend, osp0, ospMem, ospMath, ospHostIo)
			if err2 != nil {
				return common.Address{}, err2
			}
			return ospEntryAddr2, nil
		})
		if err != nil {
			return common.Address{}, common.Address{}, err
		}
		ospEntryAddr = ospEntry
	}
	log.Info("Deploying edge challenge manager")
	edgeChallengeManagerAddr, err := retry.UntilSucceeds(ctx, func() (common.Address, error) {
		edgeChallengeManagerAddr2, _, _, err2 := challengeV2gen.DeployEdgeChallengeManager(
			auth,
			backend,
		)
		if err2 != nil {
			return common.Address{}, err2
		}
		return edgeChallengeManagerAddr2, nil
	})
	if err != nil {
		return common.Address{}, common.Address{}, err
	}
	return ospEntryAddr, edgeChallengeManagerAddr, nil
}

func deployRollupCreator(
	ctx context.Context,
	backend protocol.ChainBackend,
	auth *bind.TransactOpts,
	useBlobs bool,
	useMockBridge bool,
	useMockOneStepProver bool,
) (*rollupgen.RollupCreator, common.Address, common.Address, common.Address, common.Address, error) {
	log.Info("Deploying bridge creator contracts")
	bridgeCreator, err := deployBridgeCreator(ctx, auth, useBlobs, backend, useMockBridge)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, common.Address{}, err
	}
	log.Info("Deploying challenge factory contracts")
	ospEntryAddr, challengeManagerAddr, err := deployChallengeFactory(ctx, auth, backend, useMockOneStepProver)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, common.Address{}, err
	}

	log.Info("Deploying admin logic contracts")
	rollupAdminLogic, err := retry.UntilSucceeds(ctx, func() (common.Address, error) {
		rollupAdminLogicAddr, tx, _, err2 := rollupgen.DeployRollupAdminLogic(auth, backend)
		if err2 != nil {
			return common.Address{}, err2
		}
		err2 = challenge_testing.TxSucceeded(ctx, tx, rollupAdminLogicAddr, backend, err2)
		if err2 != nil {
			return common.Address{}, err2
		}
		return rollupAdminLogicAddr, nil
	})
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, common.Address{}, err
	}

	log.Info("Deploying user logic contracts")
	rollupUserLogic, err := retry.UntilSucceeds(ctx, func() (common.Address, error) {
		rollupUserLogicAddr, tx, _, err2 := rollupgen.DeployRollupUserLogic(auth, backend)
		err2 = challenge_testing.TxSucceeded(ctx, tx, rollupUserLogicAddr, backend, err2)
		if err2 != nil {
			return common.Address{}, err2
		}
		return rollupUserLogicAddr, nil
	})
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, common.Address{}, err
	}

	type creatorResult struct {
		rollupCreatorAddress common.Address
		rollupCreator        *rollupgen.RollupCreator
	}

	log.Info("Deploying rollup creator contract")
	result, err := retry.UntilSucceeds(ctx, func() (*creatorResult, error) {
		rollupCreatorAddress, tx, rollupCreator, err2 := rollupgen.DeployRollupCreator(auth, backend)
		if err2 != nil {
			return nil, err2
		}
		err2 = challenge_testing.TxSucceeded(ctx, tx, rollupCreatorAddress, backend, err2)
		if err2 != nil {
			return nil, err2
		}
		return &creatorResult{rollupCreatorAddress: rollupCreatorAddress, rollupCreator: rollupCreator}, nil
	})
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, common.Address{}, err
	}

	log.Info("Deploying validator wallet creator contract")
	validatorWalletCreator, err := retry.UntilSucceeds(ctx, func() (common.Address, error) {
		validatorWalletCreatorAddr, tx, _, err2 := rollupgen.DeployValidatorWalletCreator(auth, backend)
		if err2 != nil {
			return common.Address{}, err2
		}
		err2 = challenge_testing.TxSucceeded(ctx, tx, validatorWalletCreatorAddr, backend, err2)
		if err2 != nil {
			return common.Address{}, err2
		}
		return validatorWalletCreatorAddr, nil
	})
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, common.Address{}, err
	}

	log.Info("Deploying upgrade executor mock")
	upgradeExecutor, err := retry.UntilSucceeds(ctx, func() (common.Address, error) {
		upgradeExecutorAddr, tx, _, err2 := mocksgen.DeployUpgradeExecutorMock(auth, backend)
		if err2 != nil {
			return common.Address{}, err2
		}
		err2 = challenge_testing.TxSucceeded(ctx, tx, upgradeExecutorAddr, backend, err2)
		if err2 != nil {
			return common.Address{}, err2
		}
		return upgradeExecutorAddr, nil
	})
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, common.Address{}, err
	}

	log.Info("Setting rollup templates")
	_, err = retry.UntilSucceeds(ctx, func() (*types.Transaction, error) {
		tx, err2 := result.rollupCreator.SetTemplates(
			auth,
			bridgeCreator,
			ospEntryAddr,
			challengeManagerAddr,
			rollupAdminLogic,
			rollupUserLogic,
			upgradeExecutor,
			validatorWalletCreator,
			common.Address{},
		)
		if err2 != nil {
			return nil, err2
		}
		if err2 := challenge_testing.WaitForTx(ctx, backend, tx); err2 != nil {
			return nil, err2
		}
		return tx, nil
	})
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, common.Address{}, err
	}
	return result.rollupCreator, rollupUserLogic, result.rollupCreatorAddress, common.Address{}, validatorWalletCreator, nil
}

// TestAccount represents a test EOA account in the simulated backend,
type TestAccount struct {
	AccountAddr common.Address
	TxOpts      *bind.TransactOpts
}

func Accounts(numAccounts uint64) ([]*TestAccount, *SimulatedBackendWrapper, error) {
	genesis := make(types.GenesisAlloc)
	gasLimit := uint64(100000000)

	accs := make([]*TestAccount, numAccounts)
	for i := uint64(0); i < numAccounts; i++ {
		privKey, err := crypto.GenerateKey()
		if err != nil {
			return nil, nil, err
		}
		pubKeyECDSA, ok := privKey.Public().(*ecdsa.PublicKey)
		if !ok {
			return nil, nil, errors.New("not ecdsa")
		}

		// Strip off the 0x and the first 2 characters 04 which is always the
		// EC prefix and is not required.
		publicKeyBytes := crypto.FromECDSAPub(pubKeyECDSA)[4:]
		var pubKey = make([]byte, 48)
		copy(pubKey, publicKeyBytes)

		addr := crypto.PubkeyToAddress(privKey.PublicKey)
		chainID := big.NewInt(1337)
		txOpts, err := bind.NewKeyedTransactorWithChainID(privKey, chainID)
		if err != nil {
			return nil, nil, err
		}
		startingBalance, _ := new(big.Int).SetString(
			"100000000000000000000000000000000000000",
			10,
		)
		genesis[addr] = types.Account{Balance: startingBalance}
		accs[i] = &TestAccount{
			AccountAddr: addr,
			TxOpts:      txOpts,
		}
	}
	backend := NewSimulatedBackendWrapper(simulated.NewBackend(genesis, simulated.WithBlockGasLimit(gasLimit)))
	return accs, backend, nil
}
