// Package setup prepares a simulated backend for testing.
//
// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE
package setup

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"os"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	solimpl "github.com/OffchainLabs/bold/chain-abstraction/sol-implementation"
	l2stateprovider "github.com/OffchainLabs/bold/layer2-state-provider"
	retry "github.com/OffchainLabs/bold/runtime"
	"github.com/OffchainLabs/bold/solgen/go/bridgegen"
	"github.com/OffchainLabs/bold/solgen/go/challengeV2gen"
	"github.com/OffchainLabs/bold/solgen/go/mocksgen"
	"github.com/OffchainLabs/bold/solgen/go/ospgen"
	"github.com/OffchainLabs/bold/solgen/go/rollupgen"
	challenge_testing "github.com/OffchainLabs/bold/testing"
	statemanager "github.com/OffchainLabs/bold/testing/mocks/state-provider"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
)

var (
	srvlog = log.New("service", "setup")
)

func init() {
	srvlog.SetHandler(log.StreamHandler(os.Stdout, log.LogfmtFormat()))
}

type Backend interface {
	bind.DeployBackend
	bind.ContractBackend
}

type CreatedValidatorFork struct {
	Leaf1              protocol.Assertion
	Leaf2              protocol.Assertion
	Chains             []*solimpl.AssertionChain
	Accounts           []*TestAccount
	Backend            *backends.SimulatedBackend
	HonestStateManager l2stateprovider.Provider
	EvilStateManager   l2stateprovider.Provider
	Addrs              *RollupAddresses
}

type CreateForkConfig struct {
	DivergeBlockHeight    uint64
	BlockHeightDifference int64
	DivergeMachineHeight  uint64
}

func CreateTwoValidatorFork(
	ctx context.Context,
	cfg *CreateForkConfig,
	opts ...Opt,
) (*CreatedValidatorFork, error) {
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

	honestStateManager, err := statemanager.NewForSimpleMachine(setup.StateManagerOpts...)
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

	stateManagerOpts := setup.StateManagerOpts
	stateManagerOpts = append(
		stateManagerOpts,
		statemanager.WithBlockDivergenceHeight(cfg.DivergeBlockHeight),
		statemanager.WithDivergentBlockHeightOffset(cfg.BlockHeightDifference),
		statemanager.WithMachineDivergenceStep(cfg.DivergeMachineHeight),
	)
	evilStateManager, err := statemanager.NewForSimpleMachine(stateManagerOpts...)
	if err != nil {
		return nil, err
	}
	honestPostState, err := honestStateManager.ExecutionStateAfterBatchCount(ctx, 1)
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

	evilPostState, err := evilStateManager.ExecutionStateAfterBatchCount(ctx, 1)
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
	Chains               []*solimpl.AssertionChain
	Accounts             []*TestAccount
	Addrs                *RollupAddresses
	Backend              *backends.SimulatedBackend
	RollupConfig         rollupgen.Config
	useMockBridge        bool
	useMockOneStepProver bool
	challengeTestingOpts []challenge_testing.Opt
	StateManagerOpts     []statemanager.Opt
}

type Opt func(setup *ChainSetup)

func WithMockBridge() Opt {
	return func(setup *ChainSetup) {
		setup.useMockBridge = true
	}
}

func WithMockOneStepProver() Opt {
	return func(setup *ChainSetup) {
		setup.useMockOneStepProver = true
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

func ChainsWithEdgeChallengeManager(opts ...Opt) (*ChainSetup, error) {
	ctx := context.Background()
	setp := &ChainSetup{}
	for _, o := range opts {
		o(setp)
	}
	accs, backend, err := Accounts(4)
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
	accs[0].TxOpts.Value = value
	mintTx, err := tokenBindings.Deposit(accs[0].TxOpts)
	if err != nil {
		return nil, err
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

	prod := false
	wasmModuleRoot := common.Hash{}
	rollupOwner := accs[0].AccountAddr
	chainId := big.NewInt(1337)
	loserStakeEscrow := common.Address{}
	miniStake := big.NewInt(1)
	genesisExecutionState := rollupgen.ExecutionState{
		GlobalState:   rollupgen.GlobalState{},
		MachineStatus: 1,
	}
	genesisInboxCount := big.NewInt(0)
	anyTrustFastConfirmer := common.Address{}
	cfg := challenge_testing.GenerateRollupConfig(
		prod,
		wasmModuleRoot,
		rollupOwner,
		chainId,
		loserStakeEscrow,
		miniStake,
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
		setp.useMockBridge,
		setp.useMockOneStepProver,
	)
	if err != nil {
		return nil, err
	}

	chains := make([]*solimpl.AssertionChain, 3)
	chain1, err := solimpl.NewAssertionChain(
		ctx,
		addresses.Rollup,
		accs[1].TxOpts,
		backend,
	)
	if err != nil {
		return nil, err
	}
	chains[0] = chain1
	chain2, err := solimpl.NewAssertionChain(
		ctx,
		addresses.Rollup,
		accs[2].TxOpts,
		backend,
	)
	if err != nil {
		return nil, err
	}
	chains[1] = chain2
	chain3, err := solimpl.NewAssertionChain(
		ctx,
		addresses.Rollup,
		accs[3].TxOpts,
		backend,
	)
	if err != nil {
		return nil, err
	}
	chains[2] = chain3

	chalManager, err := chains[1].SpecChallengeManager(ctx)
	if err != nil {
		return nil, err
	}
	chalManagerAddr := chalManager.Address()
	seed, ok := new(big.Int).SetString("10000", 10)
	if !ok {
		return nil, errors.New("could not set big int")
	}
	for _, acc := range accs {
		transferTx, err := tokenBindings.TestWETH9Transactor.Transfer(accs[0].TxOpts, acc.TxOpts.From, seed)
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
		approveTx, err := tokenBindings.TestWETH9Transactor.Approve(acc.TxOpts, addresses.Rollup, value)
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
		approveTx, err = tokenBindings.TestWETH9Transactor.Approve(acc.TxOpts, chalManagerAddr, value)
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

	setp.Chains = chains
	setp.Accounts = accs
	setp.Addrs = addresses
	setp.Backend = backend
	setp.RollupConfig = cfg
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
	DeployedAt             uint64         `json:"deployed-at"`
}

func DeployFullRollupStack(
	ctx context.Context,
	backend Backend,
	deployAuth *bind.TransactOpts,
	sequencer common.Address,
	config rollupgen.Config,
	useMockBridge bool,
	useMockOneStepProver bool,
) (*RollupAddresses, error) {
	srvlog.Info("Deploying rollup creator")
	rollupCreator, rollupUserAddr, rollupCreatorAddress, validatorUtils, validatorWalletCreator, err := deployRollupCreator(ctx, backend, deployAuth, useMockBridge, useMockOneStepProver)
	if err != nil {
		return nil, err
	}

	srvlog.Info("Creating rollup")
	tx, err := retry.UntilSucceeds[*types.Transaction](ctx, func() (*types.Transaction, error) {
		creationTx, creationErr := rollupCreator.CreateRollup(
			deployAuth,
			config,
			common.Address{},
			[]common.Address{},
			true, // Permissionless validation.
			big.NewInt(challenge_testing.MaxDataSize),
		)
		if creationErr != nil {
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

	sequencerInbox, err := bridgegen.NewSequencerInbox(info.SequencerInbox, backend)
	if err != nil {
		return nil, err
	}

	// if a zero sequencer address is specified, don't authorize any sequencers
	if sequencer != (common.Address{}) {
		srvlog.Info("Setting is batch poster")
		_, err = retry.UntilSucceeds[*types.Transaction](ctx, func() (*types.Transaction, error) {
			batchTx, err2 := sequencerInbox.SetIsBatchPoster(deployAuth, sequencer, true)
			if err2 != nil {
				return nil, err2
			}
			if waitErr := challenge_testing.WaitForTx(ctx, backend, batchTx); waitErr != nil {
				return nil, errors.Wrap(waitErr, "errored waiting for sequencerInbox.SetIsBatchPoster transaction")
			}
			return batchTx, nil
		})
		if err != nil {
			return nil, err
		}
	}

	rollup, err := rollupgen.NewRollupAdminLogic(info.RollupAddress, backend)
	if err != nil {
		return nil, err
	}

	srvlog.Info("Setting whitelist disabled")
	_, err = retry.UntilSucceeds[*types.Transaction](ctx, func() (*types.Transaction, error) {
		setTx, err2 := rollup.SetValidatorWhitelistDisabled(deployAuth, true)
		if err2 != nil {
			return nil, err2
		}
		if waitErr := challenge_testing.WaitForTx(ctx, backend, setTx); waitErr != nil {
			return nil, errors.Wrap(waitErr, "errored waiting for rollup.SetValidatorWhitelistDisabled transaction")
		}
		return setTx, nil
	})
	if err != nil {
		return nil, err
	}

	if !creationReceipt.BlockNumber.IsUint64() {
		return nil, errors.New("block number was not a uint64")
	}

	return &RollupAddresses{
		Bridge:                 info.Bridge,
		Inbox:                  info.InboxAddress,
		SequencerInbox:         info.SequencerInbox,
		DeployedAt:             creationReceipt.BlockNumber.Uint64(),
		Rollup:                 info.RollupAddress,
		RollupUserLogic:        rollupUserAddr,
		ValidatorUtils:         validatorUtils,
		ValidatorWalletCreator: validatorWalletCreator,
	}, nil
}

func deployBridgeCreator(
	ctx context.Context,
	auth *bind.TransactOpts,
	backend Backend,
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
		srvlog.Info("Deploying bridge template")
		bridgeTemplate, err = retry.UntilSucceeds[common.Address](ctx, func() (common.Address, error) {
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

	maxDataSize := big.NewInt(challenge_testing.MaxDataSize)

	srvlog.Info("Deploying seq inbox")
	seqInboxTemplate, err := retry.UntilSucceeds[common.Address](ctx, func() (common.Address, error) {
		seqInboxTemplateAddr, tx, _, err2 := bridgegen.DeploySequencerInbox(auth, backend, maxDataSize)
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

	srvlog.Info("Deploying inbox")
	inboxTemplate, err := retry.UntilSucceeds[common.Address](ctx, func() (common.Address, error) {
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

	srvlog.Info("Deploying event bridge")
	rollupEventBridgeTemplate, err := retry.UntilSucceeds[common.Address](ctx, func() (common.Address, error) {
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

	srvlog.Info("Deploying outbox")
	outboxTemplate, err := retry.UntilSucceeds[common.Address](ctx, func() (common.Address, error) {
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

	type bridgeCreationResult struct {
		bridgeCreatorAddr common.Address
		bridgeCreator     *rollupgen.BridgeCreator
	}
	srvlog.Info("Deploying bridge creator itself")
	result, err := retry.UntilSucceeds[*bridgeCreationResult](ctx, func() (*bridgeCreationResult, error) {
		bridgeCreatorAddr, tx, bridgeCreator, err2 := rollupgen.DeployBridgeCreator(auth, backend, maxDataSize)
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

	srvlog.Info("Updating bridge creator templates")
	_, err = retry.UntilSucceeds[*types.Transaction](ctx, func() (*types.Transaction, error) {
		tx, err2 := result.bridgeCreator.UpdateTemplates(auth, bridgeTemplate, seqInboxTemplate, inboxTemplate, rollupEventBridgeTemplate, outboxTemplate)
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
	backend Backend,
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
		srvlog.Info("Deploying osp0")
		osp0, err := retry.UntilSucceeds[common.Address](ctx, func() (common.Address, error) {
			osp0Addr, _, _, err2 := ospgen.DeployOneStepProver0(auth, backend)
			if err2 != nil {
				return common.Address{}, err2
			}
			return osp0Addr, nil
		})
		if err != nil {
			return common.Address{}, common.Address{}, err
		}
		srvlog.Info("Deploying ospMem")
		ospMem, err := retry.UntilSucceeds[common.Address](ctx, func() (common.Address, error) {
			ospMemAddr, _, _, err2 := ospgen.DeployOneStepProverMemory(auth, backend)
			if err2 != nil {
				return common.Address{}, err2
			}
			return ospMemAddr, nil
		})
		if err != nil {
			return common.Address{}, common.Address{}, err
		}
		srvlog.Info("Deploying ospMath")
		ospMath, err := retry.UntilSucceeds[common.Address](ctx, func() (common.Address, error) {
			ospMathAddr, _, _, err2 := ospgen.DeployOneStepProverMath(auth, backend)
			if err2 != nil {
				return common.Address{}, err2
			}
			return ospMathAddr, nil
		})
		if err != nil {
			return common.Address{}, common.Address{}, err
		}
		srvlog.Info("Deploying ospHostIo")
		ospHostIo, err := retry.UntilSucceeds[common.Address](ctx, func() (common.Address, error) {
			ospHostIoAddr, _, _, err2 := ospgen.DeployOneStepProverHostIo(auth, backend)
			if err2 != nil {
				return common.Address{}, err2
			}
			return ospHostIoAddr, nil
		})
		if err != nil {
			return common.Address{}, common.Address{}, err
		}
		srvlog.Info("Deploying ospEntry")
		ospEntry, err := retry.UntilSucceeds[common.Address](ctx, func() (common.Address, error) {
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
	srvlog.Info("Deploying edge challenge manager")
	edgeChallengeManagerAddr, err := retry.UntilSucceeds[common.Address](ctx, func() (common.Address, error) {
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
	backend Backend,
	auth *bind.TransactOpts,
	useMockBridge bool,
	useMockOneStepProver bool,
) (*rollupgen.RollupCreator, common.Address, common.Address, common.Address, common.Address, error) {
	srvlog.Info("Deploying bridge creator contracts")
	bridgeCreator, err := deployBridgeCreator(ctx, auth, backend, useMockBridge)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, common.Address{}, err
	}
	srvlog.Info("Deploying challenge factory contracts")
	ospEntryAddr, challengeManagerAddr, err := deployChallengeFactory(ctx, auth, backend, useMockOneStepProver)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, common.Address{}, err
	}

	srvlog.Info("Deploying admin logic contracts")
	rollupAdminLogic, err := retry.UntilSucceeds[common.Address](ctx, func() (common.Address, error) {
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

	srvlog.Info("Deploying user logic contracts")
	rollupUserLogic, err := retry.UntilSucceeds[common.Address](ctx, func() (common.Address, error) {
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

	srvlog.Info("Deploying rollup creator contract")
	result, err := retry.UntilSucceeds[*creatorResult](ctx, func() (*creatorResult, error) {
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

	srvlog.Info("Deploying validator wallet creator contract")
	validatorWalletCreator, err := retry.UntilSucceeds[common.Address](ctx, func() (common.Address, error) {
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

	srvlog.Info("Setting rollup templates")
	_, err = retry.UntilSucceeds[*types.Transaction](ctx, func() (*types.Transaction, error) {
		tx, err2 := result.rollupCreator.SetTemplates(
			auth,
			bridgeCreator,
			ospEntryAddr,
			challengeManagerAddr,
			rollupAdminLogic,
			rollupUserLogic,
			validatorWalletCreator,
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

func Accounts(numAccounts uint64) ([]*TestAccount, *backends.SimulatedBackend, error) {
	genesis := make(core.GenesisAlloc)
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
		genesis[addr] = core.GenesisAccount{Balance: startingBalance}
		accs[i] = &TestAccount{
			AccountAddr: addr,
			TxOpts:      txOpts,
		}
	}
	backend := backends.NewSimulatedBackend(genesis, gasLimit)
	return accs, backend, nil
}
