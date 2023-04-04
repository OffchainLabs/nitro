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
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/ospgen"
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

type CreatedValidatorFork struct {
	Leaf1                     protocol.Assertion
	Leaf2                     protocol.Assertion
	Chains                    []*solimpl.AssertionChain
	Accounts                  []*TestAccount
	Backend                   *backends.SimulatedBackend
	HonestValidatorStateRoots []common.Hash
	EvilValidatorStateRoots   []common.Hash
	Addrs                     *RollupAddresses
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

	genesis, err := setup.Chains[0].AssertionBySequenceNum(ctx, 0)
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
	honestValidatorStateRoots := make([]common.Hash, 0)
	evilValidatorStateRoots := make([]common.Hash, 0)
	honestValidatorStateRoots = append(honestValidatorStateRoots, genesisStateHash)
	evilValidatorStateRoots = append(evilValidatorStateRoots, genesisStateHash)

	var honestBlockHash common.Hash
	for i := uint64(1); i < numBlocks; i++ {
		height += 1
		honestBlockHash = setup.Backend.Commit()

		state := &protocol.ExecutionState{
			GlobalState: protocol.GoGlobalState{
				BlockHash: honestBlockHash,
				Batch:     1,
			},
			MachineStatus: protocol.MachineStatusFinished,
		}

		honestValidatorStateRoots = append(honestValidatorStateRoots, protocol.ComputeStateHash(state, big.NewInt(1)))

		// Before the divergence height, the evil validator agrees.
		if i < divergenceHeight {
			evilValidatorStateRoots = append(evilValidatorStateRoots, protocol.ComputeStateHash(state, big.NewInt(1)))
		} else {
			junkRoot := make([]byte, 32)
			_, err2 := rand.Read(junkRoot)
			if err2 != nil {
				return nil, err2
			}
			blockHash := crypto.Keccak256Hash(junkRoot)
			state.GlobalState.BlockHash = blockHash
			evilValidatorStateRoots = append(evilValidatorStateRoots, protocol.ComputeStateHash(state, big.NewInt(1)))
		}

	}

	height += 1
	honestBlockHash = setup.Backend.Commit()
	assertion, err := setup.Chains[0].CreateAssertion(
		ctx,
		height,
		genesis.SeqNum(),
		genesisState,
		&protocol.ExecutionState{
			GlobalState: protocol.GoGlobalState{
				BlockHash: honestBlockHash,
				Batch:     1,
			},
			MachineStatus: protocol.MachineStatusFinished,
		},
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

	evilPostState := &protocol.ExecutionState{
		GlobalState: protocol.GoGlobalState{
			BlockHash: common.BytesToHash([]byte("evilcommit")),
			Batch:     1,
		},
		MachineStatus: protocol.MachineStatusFinished,
	}
	forkedAssertion, err := setup.Chains[1].CreateAssertion(
		ctx,
		height,
		genesis.SeqNum(),
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

	return &CreatedValidatorFork{
		Leaf1:                     assertion,
		Leaf2:                     forkedAssertion,
		Chains:                    setup.Chains,
		Accounts:                  setup.Accounts,
		Backend:                   setup.Backend,
		Addrs:                     setup.Addrs,
		HonestValidatorStateRoots: honestValidatorStateRoots,
		EvilValidatorStateRoots:   evilValidatorStateRoots,
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
	challengePeriodSeconds := big.NewInt(100)
	miniStake := big.NewInt(1)
	cfg := challenge_testing.GenerateRollupConfig(
		prod,
		wasmModuleRoot,
		rollupOwner,
		chainId,
		loserStakeEscrow,
		challengePeriodSeconds,
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
		&bind.CallOpts{},
		accs[1].AccountAddr,
		backend,
		headerReader,
		addresses.EdgeChallengeManager,
	)
	if err != nil {
		return nil, err
	}
	chains[0] = chain1
	chain2, err := solimpl.NewAssertionChain(
		ctx,
		addresses.Rollup,
		accs[2].TxOpts,
		&bind.CallOpts{},
		accs[2].AccountAddr,
		backend,
		headerReader,
		addresses.EdgeChallengeManager,
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
	EdgeChallengeManager   common.Address `json:"edge-challenge-manager"`
}

func DeployFullRollupStack(
	ctx context.Context,
	backend *backends.SimulatedBackend,
	deployAuth *bind.TransactOpts,
	sequencer common.Address,
	config rollupgen.Config,
) (*RollupAddresses, error) {
	rollupCreator, rollupUserAddr, rollupCreatorAddress, validatorUtils, validatorWalletCreator, ospEntryAddr, err := deployRollupCreator(ctx, backend, deployAuth)
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
	backend.Commit()

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
		backend.Commit()
		if err != nil {
			return nil, err
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
	backend.Commit()
	if err != nil {
		return nil, err
	}

	receipt2, err := backend.TransactionReceipt(ctx, tx.Hash())
	if err != nil {
		return nil, err
	}
	if receipt2.Status != types.ReceiptStatusSuccessful {
		return nil, errors.New("receipt failed")
	}

	edgeChallengeManagerAddr, tx, _, err := challengeV2gen.DeployEdgeChallengeManager(
		deployAuth,
		backend,
		info.RollupAddress,
		big.NewInt(1), // TODO: Challenge period length.
		ospEntryAddr,
	)
	backend.Commit()
	err = challenge_testing.TxSucceeded(ctx, tx, edgeChallengeManagerAddr, backend, err)
	if err != nil {
		return nil, err
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
		EdgeChallengeManager:   edgeChallengeManagerAddr,
	}, nil
}

func deployBridgeCreator(
	ctx context.Context,
	auth *bind.TransactOpts,
	backend *backends.SimulatedBackend,
) (common.Address, error) {
	bridgeTemplate, tx, _, err := bridgegen.DeployBridge(auth, backend)
	backend.Commit()
	err = challenge_testing.TxSucceeded(ctx, tx, bridgeTemplate, backend, err)
	if err != nil {
		return common.Address{}, err
	}

	seqInboxTemplate, tx, _, err := bridgegen.DeploySequencerInbox(auth, backend)
	backend.Commit()
	err = challenge_testing.TxSucceeded(ctx, tx, seqInboxTemplate, backend, err)
	if err != nil {
		return common.Address{}, err
	}

	inboxTemplate, tx, _, err := bridgegen.DeployInbox(auth, backend)
	backend.Commit()
	err = challenge_testing.TxSucceeded(ctx, tx, inboxTemplate, backend, err)
	if err != nil {
		return common.Address{}, err
	}

	rollupEventBridgeTemplate, tx, _, err := rollupgen.DeployRollupEventInbox(auth, backend)
	backend.Commit()
	err = challenge_testing.TxSucceeded(ctx, tx, rollupEventBridgeTemplate, backend, err)
	if err != nil {
		return common.Address{}, err
	}

	outboxTemplate, tx, _, err := bridgegen.DeployOutbox(auth, backend)
	backend.Commit()
	err = challenge_testing.TxSucceeded(ctx, tx, outboxTemplate, backend, err)
	if err != nil {
		return common.Address{}, err
	}

	bridgeCreatorAddr, tx, bridgeCreator, err := rollupgen.DeployBridgeCreator(auth, backend)
	backend.Commit()
	err = challenge_testing.TxSucceeded(ctx, tx, bridgeCreatorAddr, backend, err)
	if err != nil {
		return common.Address{}, err
	}

	tx, err = bridgeCreator.UpdateTemplates(auth, bridgeTemplate, seqInboxTemplate, inboxTemplate, rollupEventBridgeTemplate, outboxTemplate)
	backend.Commit()
	if err != nil {
		return common.Address{}, err
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
	backend *backends.SimulatedBackend,
) (common.Address, common.Address, error) {
	osp0, tx, _, err := ospgen.DeployOneStepProver0(auth, backend)
	backend.Commit()
	err = challenge_testing.TxSucceeded(ctx, tx, osp0, backend, err)
	if err != nil {
		return common.Address{}, common.Address{}, err
	}

	ospMem, _, _, err := ospgen.DeployOneStepProverMemory(auth, backend)
	backend.Commit()
	err = challenge_testing.TxSucceeded(ctx, tx, ospMem, backend, err)
	if err != nil {
		return common.Address{}, common.Address{}, err
	}

	ospMath, _, _, err := ospgen.DeployOneStepProverMath(auth, backend)
	backend.Commit()
	err = challenge_testing.TxSucceeded(ctx, tx, ospMath, backend, err)
	if err != nil {
		return common.Address{}, common.Address{}, err
	}

	ospHostIo, _, _, err := ospgen.DeployOneStepProverHostIo(auth, backend)
	backend.Commit()
	err = challenge_testing.TxSucceeded(ctx, tx, ospHostIo, backend, err)
	if err != nil {
		return common.Address{}, common.Address{}, err
	}

	ospEntryAddr, tx, _, err := ospgen.DeployOneStepProofEntry(auth, backend, osp0, ospMem, ospMath, ospHostIo)
	backend.Commit()
	err = challenge_testing.TxSucceeded(ctx, tx, ospEntryAddr, backend, err)
	if err != nil {
		return common.Address{}, common.Address{}, err
	}

	// TODO(RJ): This assertion chain is not used, but still needed by challenge manager. Need to remove.
	genesisStateHash := common.BytesToHash([]byte("nyan"))

	assertionChainAddr, tx, _, err := challengeV2gen.DeployAssertionChain(auth, backend, genesisStateHash, big.NewInt(1))
	backend.Commit()
	err = challenge_testing.TxSucceeded(ctx, tx, assertionChainAddr, backend, err)
	if err != nil {
		return common.Address{}, common.Address{}, err
	}

	miniStakeValue := big.NewInt(1)
	challengeManagerAddr, tx, _, err := challengeV2gen.DeployChallengeManagerImpl(
		auth,
		backend,
		assertionChainAddr,
		miniStakeValue,
		big.NewInt(1),
		ospEntryAddr,
	)
	backend.Commit()
	err = challenge_testing.TxSucceeded(ctx, tx, challengeManagerAddr, backend, err)
	if err != nil {
		return common.Address{}, common.Address{}, err
	}
	return ospEntryAddr, challengeManagerAddr, nil
}

func deployRollupCreator(
	ctx context.Context,
	backend *backends.SimulatedBackend,
	auth *bind.TransactOpts,
) (*rollupgen.RollupCreator, common.Address, common.Address, common.Address, common.Address, common.Address, error) {
	bridgeCreator, err := deployBridgeCreator(ctx, auth, backend)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, common.Address{}, common.Address{}, err
	}
	ospEntryAddr, challengeManagerAddr, err := deployChallengeFactory(ctx, auth, backend)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, common.Address{}, common.Address{}, err
	}

	rollupAdminLogic, tx, _, err := rollupgen.DeployRollupAdminLogic(auth, backend)
	backend.Commit()
	err = challenge_testing.TxSucceeded(ctx, tx, rollupAdminLogic, backend, err)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, common.Address{}, common.Address{}, err
	}

	rollupUserLogic, tx, _, err := rollupgen.DeployRollupUserLogic(auth, backend)
	backend.Commit()
	err = challenge_testing.TxSucceeded(ctx, tx, rollupUserLogic, backend, err)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, common.Address{}, common.Address{}, err
	}

	rollupCreatorAddress, tx, rollupCreator, err := rollupgen.DeployRollupCreator(auth, backend)
	backend.Commit()
	err = challenge_testing.TxSucceeded(ctx, tx, rollupCreatorAddress, backend, err)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, common.Address{}, common.Address{}, err
	}

	validatorUtils, tx, _, err := rollupgen.DeployValidatorUtils(auth, backend)
	backend.Commit()
	err = challenge_testing.TxSucceeded(ctx, tx, validatorUtils, backend, err)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, common.Address{}, common.Address{}, err
	}

	validatorWalletCreator, tx, _, err := rollupgen.DeployValidatorWalletCreator(auth, backend)
	backend.Commit()
	err = challenge_testing.TxSucceeded(ctx, tx, validatorWalletCreator, backend, err)
	if err != nil {
		return nil, common.Address{}, common.Address{}, common.Address{}, common.Address{}, common.Address{}, err
	}

	_, err = rollupCreator.SetTemplates(
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
		return nil, common.Address{}, common.Address{}, common.Address{}, common.Address{}, common.Address{}, err
	}
	backend.Commit()
	return rollupCreator, rollupUserLogic, rollupCreatorAddress, validatorUtils, validatorWalletCreator, ospEntryAddr, nil
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
