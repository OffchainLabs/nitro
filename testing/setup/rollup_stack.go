package setup

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"math/big"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	solimpl "github.com/OffchainLabs/challenge-protocol-v2/protocol/sol-implementation"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/bridgegen"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/challengeV2gen"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/mocksgen"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/rollupgen"
	challenge_testing "github.com/OffchainLabs/challenge-protocol-v2/testing"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
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
	Leaf1                      protocol.Assertion
	Leaf2                      protocol.Assertion
	Chains                     []*solimpl.AssertionChain
	Accounts                   []*TestAccount
	Backend                    *backends.SimulatedBackend
	HonestValidatorStateRoots  []common.Hash
	EvilValidatorStateRoots    []common.Hash
	HonestValidatorStates      []*protocol.ExecutionState
	EvilValidatorStates        []*protocol.ExecutionState
	HonestValidatorInboxCounts []*big.Int
	EvilValidatorInboxCounts   []*big.Int
	Addrs                      *RollupAddresses
}

type CreateForkConfig struct {
	NumBlocks     uint64
	DivergeHeight uint64
}

func CreateTwoValidatorFork(
	ctx context.Context,
	cfg *CreateForkConfig,
) (*CreatedValidatorFork, error) {
	divergenceHeight := cfg.DivergeHeight
	numBlocks := cfg.NumBlocks

	setup, err := SetupChainsWithEdgeChallengeManager()
	if err != nil {
		return nil, err
	}
	prevInboxMaxCount := big.NewInt(1)

	// Advance the backend by some blocks to get over time delta failures when
	// using the assertion chain.
	for i := 0; i < 100; i++ {
		setup.Backend.Commit()
	}

	genesis, err := setup.Chains[0].AssertionBySequenceNum(ctx, 1)
	if err != nil {
		return nil, err
	}

	genesisState := &protocol.ExecutionState{
		GlobalState: protocol.GoGlobalState{
			BlockHash: common.Hash{},
		},
		MachineStatus: protocol.MachineStatusFinished,
	}
	genesisStateHash := protocol.ComputeStateHash(genesisState, big.NewInt(1))

	actualGenesisStateHash, err := genesis.StateHash()
	if err != nil {
		return nil, err
	}
	if genesisStateHash != actualGenesisStateHash {
		return nil, errors.New("genesis state hash not equal")
	}

	height := uint64(0)
	honestValidatorStateRoots := []common.Hash{genesisStateHash}
	evilValidatorStateRoots := []common.Hash{genesisStateHash}
	honestValidatorStates := []*protocol.ExecutionState{genesisState}
	evilValidatorStates := []*protocol.ExecutionState{genesisState}
	honestValidatorInboxMaxCounts := []*big.Int{big.NewInt(1)}
	evilValidatorInboxMaxCounts := []*big.Int{big.NewInt(1)}

	var honestBlockHash common.Hash
	for i := uint64(1); i < numBlocks; i++ {
		height += 1
		honestBlockHash = setup.Backend.Commit()

		state := &protocol.ExecutionState{
			GlobalState: protocol.GoGlobalState{
				BlockHash:  honestBlockHash,
				Batch:      0,
				PosInBatch: i,
			},
			MachineStatus: protocol.MachineStatusFinished,
		}

		honestValidatorStateRoots = append(honestValidatorStateRoots, protocol.ComputeStateHash(state, big.NewInt(1)))
		honestValidatorStates = append(honestValidatorStates, state)
		honestValidatorInboxMaxCounts = append(honestValidatorInboxMaxCounts, big.NewInt(1))

		// Before the divergence height, the evil validator agrees.
		if i < divergenceHeight {
			evilValidatorStateRoots = append(evilValidatorStateRoots, protocol.ComputeStateHash(state, big.NewInt(1)))
			evilValidatorStates = append(evilValidatorStates, state)
			evilValidatorInboxMaxCounts = append(evilValidatorInboxMaxCounts, big.NewInt(1))
		} else {
			stateCopy := *state
			evilState := &stateCopy
			junkRoot := make([]byte, 32)
			_, err2 := rand.Read(junkRoot)
			if err2 != nil {
				return nil, err2
			}
			blockHash := crypto.Keccak256Hash(junkRoot)
			evilState.GlobalState.BlockHash = blockHash
			evilValidatorStateRoots = append(evilValidatorStateRoots, protocol.ComputeStateHash(evilState, big.NewInt(1)))
			evilValidatorStates = append(evilValidatorStates, evilState)
			evilValidatorInboxMaxCounts = append(evilValidatorInboxMaxCounts, big.NewInt(1))
		}
	}

	honestBlockHash = setup.Backend.Commit()
	honestPostState := &protocol.ExecutionState{
		GlobalState: protocol.GoGlobalState{
			BlockHash: honestBlockHash,
			Batch:     1,
		},
		MachineStatus: protocol.MachineStatusFinished,
	}
	assertion, err := setup.Chains[0].CreateAssertion(
		ctx,
		genesisState,
		honestPostState,
		prevInboxMaxCount,
	)
	if err != nil {
		return nil, err
	}

	assertionStateHash, err := assertion.StateHash()
	if err != nil {
		return nil, err
	}
	honestValidatorStateRoots = append(honestValidatorStateRoots, assertionStateHash)
	honestValidatorStates = append(honestValidatorStates, honestPostState)
	honestValidatorInboxMaxCounts = append(honestValidatorInboxMaxCounts, new(big.Int).SetUint64(2))

	evilPostState := &protocol.ExecutionState{
		GlobalState: protocol.GoGlobalState{
			BlockHash: common.BytesToHash([]byte("evilcommit")),
			Batch:     1,
		},
		MachineStatus: protocol.MachineStatusFinished,
	}
	forkedAssertion, err := setup.Chains[1].CreateAssertion(
		ctx,
		genesisState,
		evilPostState,
		prevInboxMaxCount,
	)
	if err != nil {
		return nil, err
	}

	forkedAssertionStateHash, err := forkedAssertion.StateHash()
	if err != nil {
		return nil, err
	}
	evilValidatorStateRoots = append(evilValidatorStateRoots, forkedAssertionStateHash)
	evilValidatorStates = append(evilValidatorStates, evilPostState)
	evilValidatorInboxMaxCounts = append(evilValidatorInboxMaxCounts, new(big.Int).SetUint64(2))

	return &CreatedValidatorFork{
		Leaf1:                      assertion,
		Leaf2:                      forkedAssertion,
		Chains:                     setup.Chains,
		Accounts:                   setup.Accounts,
		Backend:                    setup.Backend,
		Addrs:                      setup.Addrs,
		HonestValidatorStateRoots:  honestValidatorStateRoots,
		EvilValidatorStateRoots:    evilValidatorStateRoots,
		HonestValidatorStates:      honestValidatorStates,
		EvilValidatorStates:        evilValidatorStates,
		HonestValidatorInboxCounts: honestValidatorInboxMaxCounts,
		EvilValidatorInboxCounts:   evilValidatorInboxMaxCounts,
	}, nil
}

type ChainSetup struct {
	Chains   []*solimpl.AssertionChain
	Accounts []*TestAccount
	Addrs    *RollupAddresses
	Backend  *backends.SimulatedBackend
	L1Reader *headerreader.HeaderReader
}

func SetupChainsWithEdgeChallengeManager() (*ChainSetup, error) {
	ctx := context.Background()
	accs, backend, err := SetupAccounts(3)
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

	headerReader := headerreader.New(util.SimulatedBackendWrapper{SimulatedBackend: backend}, func() *headerreader.Config { return &headerreader.TestConfig })
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
	ospEntryAddr, tx, _, err := mocksgen.DeployMockOneStepProofEntry(auth, backend)
	err = challenge_testing.TxSucceeded(ctx, tx, ospEntryAddr, backend, err)
	if err != nil {
		return common.Address{}, common.Address{}, errors.Wrap(err, "mocksgen.DeployMockOneStepProofEntry")
	}

	// TODO(RJ): This assertion chain is not used, but still needed by challenge manager. Need to remove.
	genesisStateHash := common.BytesToHash([]byte("nyan"))

	assertionChainAddr, tx, _, err := challengeV2gen.DeployAssertionChain(auth, backend, genesisStateHash, big.NewInt(1))
	if err != nil {
		return common.Address{}, common.Address{}, err
	}
	err = challenge_testing.TxSucceeded(ctx, tx, assertionChainAddr, backend, err)
	if err != nil {
		return common.Address{}, common.Address{}, errors.Wrap(err, "challengeV2gen.DeployAssertionChain")
	}
	edgeChallengeManagerAddr, tx, _, err := challengeV2gen.DeployEdgeChallengeManager(
		auth,
		backend,
		assertionChainAddr,
		big.NewInt(10), // TODO: Challenge period length.
		ospEntryAddr,
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

	validatorUtils, tx, _, err := rollupgen.DeployValidatorUtils(auth, backend)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, common.Address{}, err
	}
	err = challenge_testing.TxSucceeded(ctx, tx, validatorUtils, backend, err)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, common.Address{}, errors.Wrap(err, "rollupgen.DeployValidatorUtils")
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
		validatorUtils,
		validatorWalletCreator,
	)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, common.Address{}, err
	}
	if err := challenge_testing.WaitForTx(ctx, backend, tx); err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, common.Address{}, errors.Wrap(err, "failed waiting for rollupCreator.SetTemplates transaction")
	}
	return rollupCreator, rollupUserLogic, rollupCreatorAddress, validatorUtils, validatorWalletCreator, nil
}

// Represents a test EOA account in the simulated backend,
type TestAccount struct {
	AccountAddr common.Address
	TxOpts      *bind.TransactOpts
}

func SetupAccounts(numAccounts uint64) ([]*TestAccount, *backends.SimulatedBackend, error) {
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

