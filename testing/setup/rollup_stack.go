package setup

import (
	"context"
	"crypto/ecdsa"
	"math/big"

	protocol "github.com/OffchainLabs/challenge-protocol-v2/chain-abstraction"
	solimpl "github.com/OffchainLabs/challenge-protocol-v2/chain-abstraction/sol-implementation"
	l2stateprovider "github.com/OffchainLabs/challenge-protocol-v2/layer2-state-provider"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/bridgegen"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/challengeV2gen"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/mocksgen"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/rollupgen"
	challenge_testing "github.com/OffchainLabs/challenge-protocol-v2/testing"
	simulated_backend "github.com/OffchainLabs/challenge-protocol-v2/testing/setup/simulated-backend"
	statemanager "github.com/OffchainLabs/challenge-protocol-v2/testing/toys/state-provider"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/pkg/errors"
)

type SetupBackend interface {
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
) (*CreatedValidatorFork, error) {
	setup, err := ChainsWithEdgeChallengeManager()
	if err != nil {
		return nil, err
	}

	// Advance the backend by some blocks to get over time delta failures when
	// using the assertion chain.
	for i := 0; i < 100; i++ {
		setup.Backend.Commit()
	}

	genesisState := &protocol.ExecutionState{
		GlobalState: protocol.GoGlobalState{
			BlockHash: common.Hash{},
		},
		MachineStatus: protocol.MachineStatusFinished,
	}
	_ = genesisState

	honestStateManager, err := statemanager.NewForSimpleMachine()
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

	evilStateManager, err := statemanager.NewForSimpleMachine(
		statemanager.WithBlockDivergenceHeight(cfg.DivergeBlockHeight),
		statemanager.WithDivergentBlockHeightOffset(cfg.BlockHeightDifference),
		statemanager.WithMachineDivergenceStep(cfg.DivergeMachineHeight),
	)
	if err != nil {
		return nil, err
	}
	genesisCreationInfo := &protocol.AssertionCreatedInfo{
		AfterState: (&protocol.ExecutionState{
			GlobalState:   protocol.GoGlobalState{},
			MachineStatus: protocol.MachineStatusFinished,
		}).AsSolidityStruct(),
		InboxMaxCount: big.NewInt(1),
	}

	honestPostState, err := honestStateManager.LatestExecutionState(ctx)
	if err != nil {
		return nil, err
	}
	assertion, err := setup.Chains[0].CreateAssertion(
		ctx,
		genesisCreationInfo,
		honestPostState,
	)
	if err != nil {
		return nil, err
	}

	evilPostState, err := evilStateManager.LatestExecutionState(ctx)
	if err != nil {
		return nil, err
	}
	forkedAssertion, err := setup.Chains[1].CreateAssertion(
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
	Chains   []*solimpl.AssertionChain
	Accounts []*TestAccount
	Addrs    *RollupAddresses
	Backend  *backends.SimulatedBackend
	L1Reader *headerreader.HeaderReader
}

func ChainsWithEdgeChallengeManager() (*ChainSetup, error) {
	ctx := context.Background()
	accs, backend, err := Accounts(3)
	if err != nil {
		return nil, err
	}

	prod := false
	wasmModuleRoot := common.Hash{}
	rollupOwner := accs[0].AccountAddr
	chainId := big.NewInt(1337)
	loserStakeEscrow := common.Address{}
	miniStake := big.NewInt(1)
	cfg := challenge_testing.GenerateRollupConfig(
		prod,
		wasmModuleRoot,
		rollupOwner,
		chainId,
		loserStakeEscrow,
		miniStake,
	)
	addresses, err := DeployFullRollupStack(
		ctx,
		backend,
		accs[0].TxOpts,
		common.Address{}, // Sequencer addr.
		cfg,
	)
	if err != nil {
		return nil, err
	}

	headerReader := headerreader.New(simulated_backend.Wrapper{SimulatedBackend: backend}, func() *headerreader.Config { return &headerreader.TestConfig })
	headerReader.Start(ctx)
	chains := make([]*solimpl.AssertionChain, 2)
	chain1, err := solimpl.NewAssertionChain(
		ctx,
		addresses.Rollup,
		accs[1].TxOpts,
		backend,
		headerReader,
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
		headerReader,
	)
	if err != nil {
		return nil, err
	}
	chains[1] = chain2
	return &ChainSetup{
		Chains:   chains,
		Accounts: accs,
		Addrs:    addresses,
		L1Reader: headerReader,
		Backend:  backend,
	}, nil
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
	backend SetupBackend,
	deployAuth *bind.TransactOpts,
	sequencer common.Address,
	config rollupgen.Config,
) (*RollupAddresses, error) {
	rollupCreator, rollupUserAddr, rollupCreatorAddress, validatorUtils, validatorWalletCreator, err := deployRollupCreator(ctx, backend, deployAuth)
	if err != nil {
		return nil, err
	}

	nonce, err := backend.PendingNonceAt(ctx, rollupCreatorAddress)
	if err != nil {
		return nil, err
	}

	expectedRollupAddr := crypto.CreateAddress(rollupCreatorAddress, nonce+2)

	tx, err := rollupCreator.CreateRollup(
		deployAuth,
		config,
		expectedRollupAddr,
	)
	if err != nil {
		return nil, err
	}
	if waitErr := challenge_testing.WaitForTx(ctx, backend, tx); waitErr != nil {
		return nil, errors.Wrap(waitErr, "failed waiting for create rollup transaction")
	}

	receipt, err := backend.TransactionReceipt(ctx, tx.Hash())
	if err != nil {
		return nil, err
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return nil, errors.New("receipt failed")
	}

	info, err := rollupCreator.ParseRollupCreated(*receipt.Logs[len(receipt.Logs)-1])
	if err != nil {
		return nil, err
	}

	sequencerInbox, err := bridgegen.NewSequencerInbox(info.SequencerInbox, backend)
	if err != nil {
		return nil, err
	}

	// if a zero sequencer address is specified, don't authorize any sequencers
	if sequencer != (common.Address{}) {
		tx, err = sequencerInbox.SetIsBatchPoster(deployAuth, sequencer, true)
		if err != nil {
			return nil, err
		}
		if waitErr := challenge_testing.WaitForTx(ctx, backend, tx); waitErr != nil {
			return nil, errors.Wrap(waitErr, "failed waiting for sequencerInbox.SetIsBatchPoster transaction")
		}
		receipt2, err2 := backend.TransactionReceipt(ctx, tx.Hash())
		if err2 != nil {
			return nil, err
		}
		if receipt2.Status != types.ReceiptStatusSuccessful {
			return nil, errors.New("receipt failed")
		}
	}

	rollup, err := rollupgen.NewRollupAdminLogic(info.RollupAddress, backend)
	if err != nil {
		return nil, err
	}

	tx, err = rollup.SetValidatorWhitelistDisabled(deployAuth, true)
	if err != nil {
		return nil, err
	}
	if waitErr := challenge_testing.WaitForTx(ctx, backend, tx); waitErr != nil {
		return nil, errors.Wrap(waitErr, "failed waiting for rollup.SetValidatorWhitelistDisabled transaction")
	}
	receipt2, err := backend.TransactionReceipt(ctx, tx.Hash())
	if err != nil {
		return nil, err
	}
	if receipt2.Status != types.ReceiptStatusSuccessful {
		return nil, errors.New("receipt failed")
	}

	return &RollupAddresses{
		Bridge:                 info.Bridge,
		Inbox:                  info.InboxAddress,
		SequencerInbox:         info.SequencerInbox,
		DeployedAt:             receipt.BlockNumber.Uint64(),
		Rollup:                 info.RollupAddress,
		RollupUserLogic:        rollupUserAddr,
		ValidatorUtils:         validatorUtils,
		ValidatorWalletCreator: validatorWalletCreator,
	}, nil
}

func deployBridgeCreator(
	ctx context.Context,
	auth *bind.TransactOpts,
	backend SetupBackend,
) (common.Address, error) {
	bridgeTemplate, tx, _, err := bridgegen.DeployBridge(auth, backend)
	if err != nil {
		return common.Address{}, err
	}
	err = challenge_testing.TxSucceeded(ctx, tx, bridgeTemplate, backend, err)
	if err != nil {
		return common.Address{}, errors.Wrap(err, "bridgegen.DeployBridge")
	}

	seqInboxTemplate, tx, _, err := bridgegen.DeploySequencerInbox(auth, backend)
	if err != nil {
		return common.Address{}, err
	}
	err = challenge_testing.TxSucceeded(ctx, tx, seqInboxTemplate, backend, err)
	if err != nil {
		return common.Address{}, errors.Wrap(err, "bridgegen.DeploySequencerInbox")
	}

	inboxTemplate, tx, _, err := bridgegen.DeployInbox(auth, backend)
	if err != nil {
		return common.Address{}, err
	}
	err = challenge_testing.TxSucceeded(ctx, tx, inboxTemplate, backend, err)
	if err != nil {
		return common.Address{}, errors.Wrap(err, "bridgegen.DeployInbox")
	}

	rollupEventBridgeTemplate, tx, _, err := rollupgen.DeployRollupEventInbox(auth, backend)
	if err != nil {
		return common.Address{}, err
	}
	err = challenge_testing.TxSucceeded(ctx, tx, rollupEventBridgeTemplate, backend, err)
	if err != nil {
		return common.Address{}, errors.Wrap(err, "rollupgen.DeployRollupEventInbox")
	}

	outboxTemplate, tx, _, err := bridgegen.DeployOutbox(auth, backend)
	if err != nil {
		return common.Address{}, err
	}
	err = challenge_testing.TxSucceeded(ctx, tx, outboxTemplate, backend, err)
	if err != nil {
		return common.Address{}, errors.Wrap(err, "bridgegen.DeployOutbox")
	}

	bridgeCreatorAddr, tx, bridgeCreator, err := rollupgen.DeployBridgeCreator(auth, backend)
	if err != nil {
		return common.Address{}, err
	}
	err = challenge_testing.TxSucceeded(ctx, tx, bridgeCreatorAddr, backend, err)
	if err != nil {
		return common.Address{}, errors.Wrap(err, "bridgegen.DeployBridgeCreator")
	}

	tx, err = bridgeCreator.UpdateTemplates(auth, bridgeTemplate, seqInboxTemplate, inboxTemplate, rollupEventBridgeTemplate, outboxTemplate)
	if err != nil {
		return common.Address{}, err
	}
	if waitErr := challenge_testing.WaitForTx(ctx, backend, tx); waitErr != nil {
		return common.Address{}, errors.Wrap(waitErr, "failed waiting for bridgeCreator.UpdateTemplates transaction")
	}
	receipt, err := backend.TransactionReceipt(ctx, tx.Hash())
	if err != nil {
		return common.Address{}, err
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return common.Address{}, errors.New("receipt failed")
	}
	return bridgeCreatorAddr, nil
}

func deployChallengeFactory(
	ctx context.Context,
	auth *bind.TransactOpts,
	backend SetupBackend,
) (common.Address, common.Address, error) {
	ospEntryAddr, tx, _, err := mocksgen.DeploySimpleOneStepProofEntry(auth, backend)
	err = challenge_testing.TxSucceeded(ctx, tx, ospEntryAddr, backend, err)
	if err != nil {
		return common.Address{}, common.Address{}, errors.Wrap(err, "mocksgen.DeployMockOneStepProofEntry")
	}

	edgeChallengeManagerAddr, tx, _, err := challengeV2gen.DeployEdgeChallengeManager(
		auth,
		backend,
	)
	if err != nil {
		return common.Address{}, common.Address{}, err
	}
	err = challenge_testing.TxSucceeded(ctx, tx, edgeChallengeManagerAddr, backend, err)
	if err != nil {
		return common.Address{}, common.Address{}, errors.Wrap(err, "challengeV2gen.DeployEdgeChallengeManager")
	}
	return ospEntryAddr, edgeChallengeManagerAddr, nil
}

func deployRollupCreator(
	ctx context.Context,
	backend SetupBackend,
	auth *bind.TransactOpts,
) (*rollupgen.RollupCreator, common.Address, common.Address, common.Address, common.Address, error) {
	bridgeCreator, err := deployBridgeCreator(ctx, auth, backend)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, common.Address{}, err
	}
	ospEntryAddr, challengeManagerAddr, err := deployChallengeFactory(ctx, auth, backend)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, common.Address{}, err
	}

	rollupAdminLogic, tx, _, err := rollupgen.DeployRollupAdminLogic(auth, backend)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, common.Address{}, err
	}
	err = challenge_testing.TxSucceeded(ctx, tx, rollupAdminLogic, backend, err)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, common.Address{}, errors.Wrap(err, "rollupgen.DeployRollupAdminLogic")
	}

	rollupUserLogic, tx, _, err := rollupgen.DeployRollupUserLogic(auth, backend)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, common.Address{}, err
	}
	err = challenge_testing.TxSucceeded(ctx, tx, rollupUserLogic, backend, err)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, common.Address{}, errors.Wrap(err, "rollupgen.DeployRollupUserLogic")
	}

	rollupCreatorAddress, tx, rollupCreator, err := rollupgen.DeployRollupCreator(auth, backend)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, common.Address{}, err
	}
	err = challenge_testing.TxSucceeded(ctx, tx, rollupCreatorAddress, backend, err)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, common.Address{}, errors.Wrap(err, "rollupgen.DeployRollupCreator")
	}

	validatorWalletCreator, tx, _, err := rollupgen.DeployValidatorWalletCreator(auth, backend)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, common.Address{}, err
	}
	err = challenge_testing.TxSucceeded(ctx, tx, validatorWalletCreator, backend, err)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, common.Address{}, errors.Wrap(err, "rollupgen.DeployValidatorWalletCreator")
	}

	tx, err = rollupCreator.SetTemplates(
		auth,
		bridgeCreator,
		ospEntryAddr,
		challengeManagerAddr,
		rollupAdminLogic,
		rollupUserLogic,
		common.Address{},
		validatorWalletCreator,
	)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, common.Address{}, err
	}
	if err := challenge_testing.WaitForTx(ctx, backend, tx); err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, common.Address{}, errors.Wrap(err, "failed waiting for rollupCreator.SetTemplates transaction")
	}
	return rollupCreator, rollupUserLogic, rollupCreatorAddress, common.Address{}, validatorWalletCreator, nil
}

// Represents a test EOA account in the simulated backend,
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
