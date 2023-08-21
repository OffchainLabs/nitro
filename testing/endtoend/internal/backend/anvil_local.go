// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

package backend

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/OffchainLabs/bold/solgen/go/mocksgen"
	"github.com/OffchainLabs/bold/solgen/go/rollupgen"
	challenge_testing "github.com/OffchainLabs/bold/testing"
	"github.com/OffchainLabs/bold/testing/setup"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
)

var _ Backend = &AnvilLocal{}

type AnvilLocal struct {
	cancel context.CancelFunc

	alice   *bind.TransactOpts
	bob     *bind.TransactOpts
	charlie *bind.TransactOpts

	ctx    context.Context
	client *ethclient.Client
	rpc    *rpc.Client
	cmd    *exec.Cmd

	deployer *bind.TransactOpts

	addresses *setup.RollupAddresses
}

var anvilLocalChainID = big.NewInt(1002)

// NewAnvilLocal creates an anvil local backend with the following configuration:
//
//	anvil --block-time=1 --chain-id=1002
//
// You must call Start() on the returned backend to start the backend.
func NewAnvilLocal(ctx context.Context) (*AnvilLocal, error) {
	ctx, cancel := context.WithCancel(ctx)

	a := &AnvilLocal{
		cancel: cancel,

		ctx: ctx,
	}

	if err := a.loadAccounts(); err != nil {
		return nil, err
	}

	c, err := rpc.DialContext(ctx, "http://localhost:8545")
	if err != nil {
		return nil, err
	}

	a.rpc = c
	a.client = ethclient.NewClient(c)

	return a, nil
}

// Load accounts from test mnemonic. These are not real accounts. Don't even try to use them.
func (a *AnvilLocal) loadAccounts() error {
	// Load deployer from first account in test mnemonic.
	deployerPK, err := crypto.HexToECDSA("ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
	if err != nil {
		return err
	}
	deployerOpts, err := bind.NewKeyedTransactorWithChainID(deployerPK, anvilLocalChainID)
	if err != nil {
		return err
	}
	a.deployer = deployerOpts

	// Load Alice from second account in test mnemonic.
	alicePK, err := crypto.HexToECDSA("59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d")
	if err != nil {
		return err
	}
	aliceOpts, err := bind.NewKeyedTransactorWithChainID(alicePK, anvilLocalChainID)
	if err != nil {
		return err
	}
	a.alice = aliceOpts

	// Load Bob from third account in test mnemonic.
	bobPK, err := crypto.HexToECDSA("5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a")
	if err != nil {
		return err
	}
	bobOpts, err := bind.NewKeyedTransactorWithChainID(bobPK, anvilLocalChainID)
	if err != nil {
		return err
	}
	a.bob = bobOpts

	// Load Charlie from fourth account in test mnemonic.
	charliePK, err := crypto.HexToECDSA("2a871d0798f97d79848a013d4936a73bf4cc922c825d33c1cf7073dff6d409c6")
	if err != nil {
		return err
	}
	charlieOpts, err := bind.NewKeyedTransactorWithChainID(charliePK, anvilLocalChainID)
	if err != nil {
		return err
	}
	a.charlie = charlieOpts

	return nil
}

// Start the actual backend and wait for it to be ready to serve requests.
// This process also initializes the anvil blockchain by mining 100 blocks.
func (a *AnvilLocal) Start() error {
	// If the user has told us where the anvil binary is, we will use that.
	// When using bazel, the user can provide --test_env=ANVIL=$(which anvil).
	binaryPath, ok := os.LookupEnv("ANVIL")
	if !ok {
		// Otherwise, we assume it is installed at $HOME/.foundry/bin/anvil
		home, err := os.UserHomeDir()
		if err != nil {
			return errors.Wrap(err, "unable to determine user home directory")
		}
		binaryPath = path.Join(home, ".foundry/bin/anvil")
	}

	args := []string{
		"--block-time=1",
		"--chain-id=1002",
		"--gas-limit=50000000000",
	}

	cmd := exec.CommandContext(a.ctx, binaryPath, args...) // #nosec G204 -- Test only code.

	// Pipe stdout and stderr to test logs directory, if known.
	if outputsDir, ok := os.LookupEnv("TEST_UNDECLARED_OUTPUTS_DIR"); ok {
		stdoutFileName := path.Join(outputsDir, "anvil_out.log")
		stderrFileName := path.Join(outputsDir, "anvil_err.log")
		stdout, err := os.Create(stdoutFileName) // #nosec G304 -- Test only code.
		if err != nil {
			return err
		}
		stderr, err := os.Create(stderrFileName) // #nosec G304 -- Test only code.
		if err != nil {
			return err
		}

		cmd.Stdout = stdout
		cmd.Stderr = stderr

		fmt.Printf("Writing anvil stdout to %s\n", stdoutFileName)
		fmt.Printf("Writing anvil stderr to %s\n", stderrFileName)
	} else {
		fmt.Println("Warning: No environment variable found for TEST_UNDECLARED_OUTPUTS_DIR. Anvil output will not be captured.")
	}

	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, "could not start anvil")
	}

	// Wait until ready to serve a request.
	// It should be very fast.
	waitCtx, cancel := context.WithTimeout(a.ctx, 1*time.Second)
	defer cancel()
	for waitCtx.Err() == nil {
		cID, _ := a.client.ChainID(waitCtx)
		if cID != nil && cID.Cmp(anvilLocalChainID) == 0 {
			break
		}
	}

	a.cmd = cmd

	return nil
}

// Stop the backend and terminate the anvil process.
func (a *AnvilLocal) Stop() error {
	a.cancel()
	a.rpc.Close()
	return a.cmd.Process.Kill()
}

// Client returns the ethclient associated with the backend.
func (a *AnvilLocal) Client() *ethclient.Client {
	return a.client
}

// Alice returns the transactor for Alice's account.
func (a *AnvilLocal) Alice() *bind.TransactOpts {
	return a.alice
}

// Bob returns the transactor for Bob's account.`s`
func (a *AnvilLocal) Bob() *bind.TransactOpts {
	return a.bob
}

// Charlie returns the transactor for Charlie's account.`s`
func (a *AnvilLocal) Charlie() *bind.TransactOpts {
	return a.charlie
}

func (a *AnvilLocal) DeployRollup() (common.Address, error) {
	prod := false
	wasmModuleRoot := common.Hash{}
	rollupOwner := a.deployer.From
	loserStakeEscrow := common.Address{}
	anyTrustFastConfirmer := common.Address{}
	genesisExecutionState := rollupgen.ExecutionState{
		GlobalState:   rollupgen.GlobalState{},
		MachineStatus: 1,
	}
	genesisInboxCount := big.NewInt(0)
	miniStake := big.NewInt(1)

	ctx := context.TODO()
	stakeToken, tx, tokenBindings, err := mocksgen.DeployTestWETH9(
		a.deployer,
		a.client,
		"Weth",
		"WETH",
	)
	if err != nil {
		return common.Address{}, errors.Wrap(err, "could not deploy test weth")
	}
	if waitErr := challenge_testing.WaitForTx(ctx, a.client, tx); waitErr != nil {
		return common.Address{}, errors.Wrap(waitErr, "failed waiting for transaction")
	}
	receipt, err := a.client.TransactionReceipt(ctx, tx.Hash())
	if err != nil {
		return common.Address{}, errors.Wrap(err, "could not get tx hash")
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return common.Address{}, errors.New("receipt failed")
	}

	result, err := setup.DeployFullRollupStack(
		a.ctx,
		a.client,
		a.deployer,
		common.Address{}, // Sequencer
		challenge_testing.GenerateRollupConfig(
			prod,
			wasmModuleRoot,
			rollupOwner,
			anvilLocalChainID,
			loserStakeEscrow,
			miniStake,
			stakeToken,
			genesisExecutionState,
			genesisInboxCount,
			anyTrustFastConfirmer,
		),
		false, // Do not use a mock bridge.
	)
	if err != nil {
		return common.Address{}, errors.Wrap(err, "could not deploy rollup stack")
	}

	value, ok := new(big.Int).SetString("100000", 10)
	if !ok {
		return common.Address{}, errors.New("could not set value")
	}
	a.deployer.Value = value
	mintTx, err := tokenBindings.Deposit(a.deployer)
	if err != nil {
		return common.Address{}, errors.Wrap(err, "could not mint test weth")
	}
	if waitErr := challenge_testing.WaitForTx(ctx, a.client, mintTx); waitErr != nil {
		return common.Address{}, errors.Wrap(waitErr, "failed waiting for transaction")
	}
	receipt, err = a.client.TransactionReceipt(ctx, mintTx.Hash())
	if err != nil {
		return common.Address{}, errors.Wrap(err, "could not get tx hash")
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return common.Address{}, errors.New("receipt failed")
	}
	a.deployer.Value = big.NewInt(0)
	rollupCaller, err := rollupgen.NewRollupUserLogicCaller(result.Rollup, a.client)
	if err != nil {
		return common.Address{}, err
	}
	chalManagerAddr, err := rollupCaller.ChallengeManager(&bind.CallOpts{})
	if err != nil {
		return common.Address{}, err
	}
	accs := []*bind.TransactOpts{a.alice, a.bob, a.charlie}
	seed, ok := new(big.Int).SetString("1000", 10)
	if !ok {
		return common.Address{}, errors.New("could not set big int")
	}
	for _, acc := range accs {
		transferTx, err := tokenBindings.TestWETH9Transactor.Transfer(a.deployer, acc.From, seed)
		if err != nil {
			return common.Address{}, errors.Wrap(err, "could not approve account")
		}
		if waitErr := challenge_testing.WaitForTx(ctx, a.client, transferTx); waitErr != nil {
			return common.Address{}, errors.Wrap(waitErr, "failed waiting for transfer transaction")
		}
		receipt, err := a.client.TransactionReceipt(ctx, transferTx.Hash())
		if err != nil {
			return common.Address{}, errors.Wrap(err, "could not get tx receipt")
		}
		if receipt.Status != types.ReceiptStatusSuccessful {
			return common.Address{}, errors.New("receipt failed")
		}
		approveTx, err := tokenBindings.TestWETH9Transactor.Approve(acc, result.Rollup, value)
		if err != nil {
			return common.Address{}, errors.Wrap(err, "could not approve account")
		}
		if waitErr := challenge_testing.WaitForTx(ctx, a.client, approveTx); waitErr != nil {
			return common.Address{}, errors.Wrap(waitErr, "failed waiting for approval transaction")
		}
		receipt, err = a.client.TransactionReceipt(ctx, approveTx.Hash())
		if err != nil {
			return common.Address{}, errors.Wrap(err, "could not get tx receipt")
		}
		if receipt.Status != types.ReceiptStatusSuccessful {
			return common.Address{}, errors.New("receipt failed")
		}
		approveTx, err = tokenBindings.TestWETH9Transactor.Approve(acc, chalManagerAddr, value)
		if err != nil {
			return common.Address{}, errors.Wrap(err, "could not approve account")
		}
		if waitErr := challenge_testing.WaitForTx(ctx, a.client, approveTx); waitErr != nil {
			return common.Address{}, errors.Wrap(waitErr, "failed waiting for approval transaction")
		}
		receipt, err = a.client.TransactionReceipt(ctx, approveTx.Hash())
		if err != nil {
			return common.Address{}, errors.Wrap(err, "could not get tx receipt")
		}
		if receipt.Status != types.ReceiptStatusSuccessful {
			return common.Address{}, errors.New("receipt failed")
		}
	}

	a.addresses = result

	return result.Rollup, a.MineBlocks(100) // At least 75 blocks should be mined for a challenge to be possible.
}

// MineBlocks will call anvil to instantly mine n blocks.
func (a *AnvilLocal) MineBlocks(n uint64) error {
	return a.rpc.CallContext(a.ctx, nil, "anvil_mine", hexutil.EncodeUint64(n))
}

func (a *AnvilLocal) ContractAddresses() *setup.RollupAddresses {
	return a.addresses
}
