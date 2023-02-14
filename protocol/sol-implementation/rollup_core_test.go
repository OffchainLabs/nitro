package solimpl

import (
	"context"
	"fmt"
	"math/big"

	"crypto/ecdsa"
	"errors"
	"testing"

	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/bridgegen"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/challengeV2gen"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/ospgen"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/rollupgen"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestDeployFullRollupStack(t *testing.T) {
	ctx := context.Background()
	acc, err := setupAccount()
	require.NoError(t, err)
	prod := false
	wasmModuleRoot := common.Hash{}
	rollupOwner := acc.accountAddr
	chainId := big.NewInt(1337)
	loserStakeEscrow := common.Address{}
	cfg := generateRollupConfig(prod, wasmModuleRoot, rollupOwner, chainId, loserStakeEscrow)
	numValidators := uint64(10)
	deployFullRollupStack(
		t,
		ctx,
		acc.backend,
		acc.txOpts,
		common.Address{},
		numValidators,
		cfg,
	)
}

type rollupAddresses struct {
	Bridge                 common.Address `json:"bridge"`
	Inbox                  common.Address `json:"inbox"`
	SequencerInbox         common.Address `json:"sequencer-inbox"`
	Rollup                 common.Address `json:"rollup"`
	RollupUserLogic        common.Address `json:"rollup-user-logic"`
	ValidatorUtils         common.Address `json:"validator-utils"`
	ValidatorWalletCreator common.Address `json:"validator-wallet-creator"`
	DeployedAt             uint64         `json:"deployed-at"`
}

func deployFullRollupStack(
	t *testing.T,
	ctx context.Context,
	backend *backends.SimulatedBackend,
	deployAuth *bind.TransactOpts,
	sequencer common.Address,
	authorizeValidators uint64,
	config rollupgen.Config,
) *rollupAddresses {
	t.Helper()
	rollupCreator, rollupUserLogic, rollupCreatorAddress, validatorUtils, validatorWalletCreator := deployRollupCreator(t, ctx, backend, deployAuth)

	nonce, err := backend.PendingNonceAt(ctx, rollupCreatorAddress)
	require.NoError(t, err)

	expectedRollupAddr := crypto.CreateAddress(rollupCreatorAddress, nonce+2)

	tx, err := rollupCreator.CreateRollup(
		deployAuth,
		config,
		expectedRollupAddr,
	)
	require.NoError(t, err)
	backend.Commit()

	receipt, err := backend.TransactionReceipt(ctx, tx.Hash())
	require.NoError(t, err)
	require.Equal(t, uint64(1), receipt.Status)

	info, err := rollupCreator.ParseRollupCreated(*receipt.Logs[len(receipt.Logs)-1])
	require.NoError(t, err)

	sequencerInbox, err := bridgegen.NewSequencerInbox(info.SequencerInbox, backend)
	require.NoError(t, err)

	// if a zero sequencer address is specified, don't authorize any sequencers
	if sequencer != (common.Address{}) {
		tx, err = sequencerInbox.SetIsBatchPoster(deployAuth, sequencer, true)
		backend.Commit()
		require.NoError(t, err)

		receipt2, err := backend.TransactionReceipt(ctx, tx.Hash())
		require.NoError(t, err)
		require.Equal(t, uint64(1), receipt2.Status)
	}

	var allowValidators []bool
	var validatorAddrs []common.Address
	for i := uint64(1); i <= authorizeValidators; i++ {
		validatorAddrs = append(validatorAddrs, crypto.CreateAddress(validatorWalletCreator, i))
		allowValidators = append(allowValidators, true)
	}
	if len(validatorAddrs) > 0 {
		rollup, err := rollupgen.NewRollupAdminLogic(info.RollupAddress, backend)
		require.NoError(t, err)

		tx, err = rollup.SetValidator(deployAuth, validatorAddrs, allowValidators)
		backend.Commit()
		require.NoError(t, err)

		receipt2, err := backend.TransactionReceipt(ctx, tx.Hash())
		require.NoError(t, err)
		require.Equal(t, uint64(1), receipt2.Status)
	}

	return &rollupAddresses{
		Bridge:                 info.Bridge,
		Inbox:                  info.InboxAddress,
		SequencerInbox:         info.SequencerInbox,
		DeployedAt:             receipt.BlockNumber.Uint64(),
		Rollup:                 info.RollupAddress,
		RollupUserLogic:        rollupUserLogic,
		ValidatorUtils:         validatorUtils,
		ValidatorWalletCreator: validatorWalletCreator,
	}
}

func deployBridgeCreator(
	t *testing.T,
	ctx context.Context,
	auth *bind.TransactOpts,
	backend *backends.SimulatedBackend,
) common.Address {
	t.Helper()
	bridgeTemplate, tx, _, err := bridgegen.DeployBridge(auth, backend)
	backend.Commit()
	err = andTxSucceeded(ctx, tx, bridgeTemplate, backend, err)
	require.NoError(t, err)

	seqInboxTemplate, tx, _, err := bridgegen.DeploySequencerInbox(auth, backend)
	backend.Commit()
	err = andTxSucceeded(ctx, tx, seqInboxTemplate, backend, err)
	require.NoError(t, err)

	inboxTemplate, tx, _, err := bridgegen.DeployInbox(auth, backend)
	backend.Commit()
	err = andTxSucceeded(ctx, tx, inboxTemplate, backend, err)
	require.NoError(t, err)

	rollupEventBridgeTemplate, tx, _, err := rollupgen.DeployRollupEventInbox(auth, backend)
	backend.Commit()
	err = andTxSucceeded(ctx, tx, rollupEventBridgeTemplate, backend, err)
	require.NoError(t, err)

	outboxTemplate, tx, _, err := bridgegen.DeployOutbox(auth, backend)
	backend.Commit()
	err = andTxSucceeded(ctx, tx, outboxTemplate, backend, err)
	require.NoError(t, err)

	bridgeCreatorAddr, tx, bridgeCreator, err := rollupgen.DeployBridgeCreator(auth, backend)
	backend.Commit()
	err = andTxSucceeded(ctx, tx, bridgeCreatorAddr, backend, err)
	require.NoError(t, err)

	tx, err = bridgeCreator.UpdateTemplates(auth, bridgeTemplate, seqInboxTemplate, inboxTemplate, rollupEventBridgeTemplate, outboxTemplate)
	backend.Commit()
	require.NoError(t, err)

	receipt, err := backend.TransactionReceipt(ctx, tx.Hash())
	require.NoError(t, err)
	require.Equal(t, uint64(1), receipt.Status)

	return bridgeCreatorAddr
}

func deployChallengeFactory(
	t *testing.T,
	ctx context.Context,
	auth *bind.TransactOpts,
	backend *backends.SimulatedBackend,
) (common.Address, common.Address) {
	t.Helper()
	osp0, tx, _, err := ospgen.DeployOneStepProver0(auth, backend)
	backend.Commit()
	err = andTxSucceeded(ctx, tx, osp0, backend, err)
	require.NoError(t, err)

	ospMem, _, _, err := ospgen.DeployOneStepProverMemory(auth, backend)
	backend.Commit()
	err = andTxSucceeded(ctx, tx, ospMem, backend, err)
	require.NoError(t, err)

	ospMath, _, _, err := ospgen.DeployOneStepProverMath(auth, backend)
	backend.Commit()
	err = andTxSucceeded(ctx, tx, ospMath, backend, err)
	require.NoError(t, err)

	ospHostIo, _, _, err := ospgen.DeployOneStepProverHostIo(auth, backend)
	backend.Commit()
	err = andTxSucceeded(ctx, tx, ospHostIo, backend, err)
	require.NoError(t, err)

	ospEntryAddr, tx, _, err := ospgen.DeployOneStepProofEntry(auth, backend, osp0, ospMem, ospMath, ospHostIo)
	backend.Commit()
	err = andTxSucceeded(ctx, tx, ospEntryAddr, backend, err)
	require.NoError(t, err)

	genesisStateHash := common.BytesToHash([]byte("nyan"))
	challengePeriodSeconds := big.NewInt(100)

	assertionChainAddr, tx, _, err := challengeV2gen.DeployAssertionChain(auth, backend, genesisStateHash, challengePeriodSeconds)
	backend.Commit()
	err = andTxSucceeded(ctx, tx, assertionChainAddr, backend, err)
	require.NoError(t, err)

	miniStakeValue := big.NewInt(1)
	challengeManagerAddr, tx, _, err := challengeV2gen.DeployChallengeManagerImpl(
		auth,
		backend,
		assertionChainAddr,
		miniStakeValue,
		challengePeriodSeconds,
		ospEntryAddr,
	)
	backend.Commit()
	err = andTxSucceeded(ctx, tx, challengeManagerAddr, backend, err)
	require.NoError(t, err)

	return ospEntryAddr, challengeManagerAddr
}

func deployRollupCreator(
	t *testing.T,
	ctx context.Context,
	backend *backends.SimulatedBackend,
	auth *bind.TransactOpts,
) (*rollupgen.RollupCreator, common.Address, common.Address, common.Address, common.Address) {
	t.Helper()
	bridgeCreator := deployBridgeCreator(t, ctx, auth, backend)
	ospEntryAddr, challengeManagerAddr := deployChallengeFactory(t, ctx, auth, backend)

	rollupAdminLogic, tx, _, err := rollupgen.DeployRollupAdminLogic(auth, backend)
	backend.Commit()
	err = andTxSucceeded(ctx, tx, rollupAdminLogic, backend, err)
	require.NoError(t, err)

	rollupUserLogic, tx, _, err := rollupgen.DeployRollupUserLogic(auth, backend)
	backend.Commit()
	err = andTxSucceeded(ctx, tx, rollupUserLogic, backend, err)
	require.NoError(t, err)

	rollupCreatorAddress, tx, rollupCreator, err := rollupgen.DeployRollupCreator(auth, backend)
	backend.Commit()
	err = andTxSucceeded(ctx, tx, rollupCreatorAddress, backend, err)
	require.NoError(t, err)

	validatorUtils, tx, _, err := rollupgen.DeployValidatorUtils(auth, backend)
	backend.Commit()
	err = andTxSucceeded(ctx, tx, validatorUtils, backend, err)
	require.NoError(t, err)

	validatorWalletCreator, tx, _, err := rollupgen.DeployValidatorWalletCreator(auth, backend)
	backend.Commit()
	err = andTxSucceeded(ctx, tx, validatorWalletCreator, backend, err)
	require.NoError(t, err)

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
	backend.Commit()
	require.NoError(t, err)

	receipt, err := backend.TransactionReceipt(ctx, tx.Hash())
	require.NoError(t, err)
	require.Equal(t, uint64(1), receipt.Status)
	return rollupCreator, rollupUserLogic, rollupCreatorAddress, validatorUtils, validatorWalletCreator
}

func generateRollupConfig(
	prod bool,
	wasmModuleRoot common.Hash,
	rollupOwner common.Address,
	chainId *big.Int,
	loserStakeEscrow common.Address,
) rollupgen.Config {
	var confirmPeriod uint64
	if prod {
		confirmPeriod = 45818
	} else {
		confirmPeriod = 20
	}
	return rollupgen.Config{
		ConfirmPeriodBlocks:      confirmPeriod,
		ExtraChallengeTimeBlocks: 200,
		StakeToken:               common.Address{},
		BaseStake:                big.NewInt(100),
		WasmModuleRoot:           wasmModuleRoot,
		Owner:                    rollupOwner,
		LoserStakeEscrow:         loserStakeEscrow,
		ChainId:                  chainId,
		SequencerInboxMaxTimeVariation: rollupgen.ISequencerInboxMaxTimeVariation{
			DelayBlocks:   big.NewInt(60 * 60 * 24 / 15),
			FutureBlocks:  big.NewInt(12),
			DelaySeconds:  big.NewInt(60 * 60 * 24),
			FutureSeconds: big.NewInt(60 * 60),
		},
	}
}

// Represents a test EOA account in the simulated backend,
type testAccount struct {
	accountAddr common.Address
	backend     *backends.SimulatedBackend
	txOpts      *bind.TransactOpts
}

func setupAccount() (*testAccount, error) {
	genesis := make(core.GenesisAlloc)
	privKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	pubKeyECDSA, ok := privKey.Public().(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("error casting public key to ECDSA")
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
		return nil, err
	}
	startingBalance, _ := new(big.Int).SetString(
		"100000000000000000000000000000000000000",
		10,
	)
	genesis[addr] = core.GenesisAccount{Balance: startingBalance}
	gasLimit := uint64(100000000)
	backend := backends.NewSimulatedBackend(genesis, gasLimit)
	return &testAccount{
		accountAddr: addr,
		backend:     backend,
		txOpts:      txOpts,
	}, nil
}

func andTxSucceeded(
	ctx context.Context,
	tx *types.Transaction,
	addr common.Address,
	backend *backends.SimulatedBackend,
	err error,
) error {
	if err != nil {
		return fmt.Errorf("error submitting tx: %w", err)
	}
	receipt, err := backend.TransactionReceipt(ctx, tx.Hash())
	if err != nil {
		return err
	}
	if receipt.Status != 1 {
		return errors.New("tx failed")
	}
	code, err := backend.CodeAt(ctx, addr, nil)
	if err != nil {
		return err
	}
	if len(code) == 0 {
		return errors.New("contract not deployed")
	}
	return nil
}
