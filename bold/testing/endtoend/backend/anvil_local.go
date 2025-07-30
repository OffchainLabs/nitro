// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

package backend

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"

	protocol "github.com/offchainlabs/bold/chain-abstraction"
	challenge_testing "github.com/offchainlabs/bold/testing"
	"github.com/offchainlabs/bold/testing/setup"
	"github.com/offchainlabs/bold/util"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
)

var _ Backend = &AnvilLocal{}

type AnvilLocal struct {
	client    protocol.ChainBackend
	cmd       *exec.Cmd
	addresses *setup.RollupAddresses
	accounts  []*bind.TransactOpts
}

var anvilLocalChainID = big.NewInt(1002)

// NewAnvilLocal creates an anvil local backend with the following configuration:
//
//	anvil --block-time=1 --chain-id=1002
//
// You must call Start() on the returned backend to start the backend.
func NewAnvilLocal(ctx context.Context) (*AnvilLocal, error) {
	a := &AnvilLocal{}
	if err := a.loadAccounts(); err != nil {
		return nil, err
	}
	c, err := rpc.DialContext(ctx, "http://localhost:8686")
	if err != nil {
		return nil, err
	}
	a.client = util.NewBackendWrapper(ethclient.NewClient(c), rpc.LatestBlockNumber)
	return a, nil
}

// Load accounts from test mnemonic. These are not real accounts. Don't even try to use them.
func (a *AnvilLocal) loadAccounts() error {
	accounts := make([]*bind.TransactOpts, 0)
	for i := 0; i < len(anvilPrivKeyHexStrings); i++ {
		privKeyHex := hexutil.MustDecode(anvilPrivKeyHexStrings[i])
		privKey, err := crypto.ToECDSA(privKeyHex)
		if err != nil {
			return err
		}
		txOpts, err := bind.NewKeyedTransactorWithChainID(privKey, anvilLocalChainID)
		if err != nil {
			return err
		}
		accounts = append(accounts, txOpts)
	}
	a.accounts = accounts
	return nil
}

// Start the actual backend and wait for it to be ready to serve requests.
// This process also initializes the anvil blockchain by mining 100 blocks.
func (a *AnvilLocal) Start(ctx context.Context) error {
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
		"--port=8686",
	}

	cmd := exec.CommandContext(ctx, binaryPath, args...) // #nosec G204 -- Test only code.

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
	waitCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	for waitCtx.Err() == nil {
		cID, _ := a.client.ChainID(waitCtx)
		if cID != nil && cID.Cmp(anvilLocalChainID) == 0 {
			break
		}
	}

	a.cmd = cmd

	go func() {
		<-ctx.Done()
		a.client.Close()
		if err := a.cmd.Process.Kill(); err != nil {
			fmt.Printf("Could not kill anvil process: %v\n", err)
		}
	}()

	return nil
}

// Client returns the ethclient associated with the backend.
func (a *AnvilLocal) Client() protocol.ChainBackend {
	return a.client
}

func (a *AnvilLocal) Accounts() []*bind.TransactOpts {
	return a.accounts
}

func (a *AnvilLocal) Commit() common.Hash {
	return common.Hash{}
}

func (a *AnvilLocal) DeployRollup(ctx context.Context, opts ...challenge_testing.Opt) (*setup.RollupAddresses, error) {
	prod := false
	wasmModuleRoot := common.Hash{}
	rollupOwner := a.accounts[0].From
	loserStakeEscrow := rollupOwner
	anyTrustFastConfirmer := common.Address{}
	genesisExecutionState := rollupgen.AssertionState{
		GlobalState:   rollupgen.GlobalState{},
		MachineStatus: 1,
	}
	genesisInboxCount := big.NewInt(0)

	stakeToken, tx, tokenBindings, err := mocksgen.DeployTestWETH9(
		a.accounts[0],
		a.client,
		"Weth",
		"WETH",
	)
	if err != nil {
		return nil, errors.Wrap(err, "could not deploy test weth")
	}
	if waitErr := challenge_testing.WaitForTx(ctx, a.client, tx); waitErr != nil {
		return nil, errors.Wrap(waitErr, "errored waiting for transaction")
	}
	receipt, err := a.client.TransactionReceipt(ctx, tx.Hash())
	if err != nil {
		return nil, errors.Wrap(err, "could not get tx hash")
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return nil, errors.New("receipt not successful")
	}

	miniStakeValues := []*big.Int{
		big.NewInt(1),
		big.NewInt(2),
		big.NewInt(3),
	}
	result, err := setup.DeployFullRollupStack(
		ctx,
		a.client,
		a.accounts[0],
		common.Address{}, // Sequencer
		challenge_testing.GenerateRollupConfig(
			prod,
			wasmModuleRoot,
			rollupOwner,
			anvilLocalChainID,
			loserStakeEscrow,
			miniStakeValues,
			stakeToken,
			genesisExecutionState,
			genesisInboxCount,
			anyTrustFastConfirmer,
			opts...,
		),
		setup.RollupStackConfig{
			UseMockBridge:          false,
			UseMockOneStepProver:   true,
			UseBlobs:               true,
			MinimumAssertionPeriod: 0,
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "could not deploy rollup stack")
	}

	value, ok := new(big.Int).SetString("100000", 10)
	if !ok {
		return nil, errors.New("could not set value")
	}
	a.accounts[0].Value = value
	mintTx, err := tokenBindings.Deposit(a.accounts[0])
	if err != nil {
		return nil, errors.Wrap(err, "could not mint test weth")
	}
	if waitErr := challenge_testing.WaitForTx(ctx, a.client, mintTx); waitErr != nil {
		return nil, errors.Wrap(waitErr, "errored waiting for transaction")
	}
	receipt, err = a.client.TransactionReceipt(ctx, mintTx.Hash())
	if err != nil {
		return nil, errors.Wrap(err, "could not get tx hash")
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return nil, errors.New("receipt errored")
	}
	a.accounts[0].Value = big.NewInt(0)
	rollupCaller, err := rollupgen.NewRollupUserLogicCaller(result.Rollup, a.client)
	if err != nil {
		return nil, err
	}
	chalManagerAddr, err := rollupCaller.ChallengeManager(&bind.CallOpts{})
	if err != nil {
		return nil, err
	}
	seed, ok := new(big.Int).SetString("1000", 10)
	if !ok {
		return nil, errors.New("could not set big int")
	}
	for _, acc := range a.accounts[1:] {
		transferTx, err := tokenBindings.Transfer(a.accounts[0], acc.From, seed)
		if err != nil {
			return nil, errors.Wrap(err, "could not approve account")
		}
		if waitErr := challenge_testing.WaitForTx(ctx, a.client, transferTx); waitErr != nil {
			return nil, errors.Wrap(waitErr, "errored waiting for transfer transaction")
		}
		receipt, err := a.client.TransactionReceipt(ctx, transferTx.Hash())
		if err != nil {
			return nil, errors.Wrap(err, "could not get tx receipt")
		}
		if receipt.Status != types.ReceiptStatusSuccessful {
			return nil, errors.New("receipt not successful")
		}
		approveTx, err := tokenBindings.Approve(acc, result.Rollup, value)
		if err != nil {
			return nil, errors.Wrap(err, "could not approve account")
		}
		if waitErr := challenge_testing.WaitForTx(ctx, a.client, approveTx); waitErr != nil {
			return nil, errors.Wrap(waitErr, "errored waiting for approval transaction")
		}
		receipt, err = a.client.TransactionReceipt(ctx, approveTx.Hash())
		if err != nil {
			return nil, errors.Wrap(err, "could not get tx receipt")
		}
		if receipt.Status != types.ReceiptStatusSuccessful {
			return nil, errors.New("receipt not successful")
		}
		approveTx, err = tokenBindings.Approve(acc, chalManagerAddr, value)
		if err != nil {
			return nil, errors.Wrap(err, "could not approve account")
		}
		if waitErr := challenge_testing.WaitForTx(ctx, a.client, approveTx); waitErr != nil {
			return nil, errors.Wrap(waitErr, "errored waiting for approval transaction")
		}
		receipt, err = a.client.TransactionReceipt(ctx, approveTx.Hash())
		if err != nil {
			return nil, errors.Wrap(err, "could not get tx receipt")
		}
		if receipt.Status != types.ReceiptStatusSuccessful {
			return nil, errors.New("receipt not successful")
		}
	}

	a.addresses = result

	return result, a.MineBlocks(ctx, 100) // At least 100 blocks should be mined for a challenge to be possible.
}

// MineBlocks will call anvil to instantly mine n blocks.
func (a *AnvilLocal) MineBlocks(ctx context.Context, n uint64) error {
	return a.client.Client().CallContext(ctx, nil, "anvil_mine", hexutil.EncodeUint64(n))
}

func (a *AnvilLocal) ContractAddresses() *setup.RollupAddresses {
	return a.addresses
}
