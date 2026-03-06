// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build !race

package arbtest

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/rawdb"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/offchainlabs/nitro/util/testhelpers"
	"github.com/offchainlabs/nitro/validator"
	"github.com/offchainlabs/nitro/validator/client"
	"github.com/offchainlabs/nitro/validator/server_api"
)

// TestRustValidationServerAPI verifies that the Go ValidationClient can connect
// to the Rust validation server and that all handshake API methods work.
//
// Prerequisites: make build-validation-server && make build-replay-env
func TestRustValidationServerAPI(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	rvAddr := startRustValidatorServer(t, ctx)
	valClient := connectValidationClient(t, ctx, rvAddr)
	defer valClient.Stop()

	if valClient.Name() != "Rust JIT validator" {
		Fatal(t, "unexpected validator name:", valClient.Name())
	}
	if valClient.Capacity() < 2 {
		Fatal(t, "unexpected capacity:", valClient.Capacity())
	}

	roots, err := valClient.WasmModuleRoots()
	Require(t, err)
	if len(roots) == 0 {
		Fatal(t, "server reported no WASM module roots")
	}

	archs := valClient.StylusArchs()
	if len(archs) == 0 {
		Fatal(t, "server reported no stylus architectures")
	}
}

// TestRustServerValidation proves end-to-end block validation through
// the Rust validation server. It deploys and calls a Stylus contract,
// computes the expected GoGlobalState locally, sends the block to the
// Rust server, and asserts the results match.
//
// Prerequisites: make build-validation-server && make build-replay-env
func TestRustServerValidation(t *testing.T) {
	builder, auth, cleanup := setupProgramTest(t, false)
	defer cleanup()
	ctx := builder.ctx

	rvAddr := startRustValidatorServer(t, ctx)

	msgIdx := deployStylusContractAndCall(t, ctx, builder, auth)
	waitForMessageIndex(t, ctx, builder, msgIdx)
	expectedState := computeExpectedState(t, ctx, builder, msgIdx)
	actualState := validateBlockViaRustServer(t, ctx, builder, rvAddr, msgIdx)

	if expectedState != actualState {
		Fatal(t, "GoGlobalState mismatch - expected: ", expectedState, ", actual: ", actualState)
	}
	t.Logf("Validation succeeded: BlockHash=%s Batch=%d PosInBatch=%d",
		actualState.BlockHash.Hex(), actualState.Batch, actualState.PosInBatch)
}

func startRustValidatorServer(t *testing.T, ctx context.Context) string {
	t.Helper()
	root := projectRoot(t)

	validatorBin := filepath.Join(root, "target", "bin", "validator")
	if _, err := os.Stat(validatorBin); os.IsNotExist(err) {
		t.Skipf("Rust validator binary not found at %s; run 'make build-validation-server'", validatorBin)
	}

	addr := fmt.Sprintf("127.0.0.1:%d", getRandomPort(t))
	cmd := exec.CommandContext(ctx, validatorBin, "--address", addr)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	Require(t, cmd.Start())
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})

	waitForTCP(t, addr, 30*time.Second)
	return addr
}

func connectValidationClient(t *testing.T, ctx context.Context, addr string) *client.ValidationClient {
	t.Helper()
	config := rustValidatorClientConfig(addr)
	valClient := client.NewValidationClient(StaticFetcherFrom(t, &config), nil)
	Require(t, valClient.Start(ctx))
	return valClient
}

func rustValidatorClientConfig(addr string) rpcclient.ClientConfig {
	return rpcclient.ClientConfig{
		URL:       "http://" + addr,
		JWTSecret: "",
		Timeout:   120 * time.Second,
		Retries:   3,
	}
}

func projectRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		Fatal(t, "could not determine project root")
	}
	return filepath.Dir(filepath.Dir(filename))
}

// deployStylusContractAndCall deploys a Stylus "storage" contract, activates it,
// and performs a storage write call. Returns the message index of the call's block.
func deployStylusContractAndCall(t *testing.T, ctx context.Context, builder *NodeBuilder, auth bind.TransactOpts) arbutil.MessageIndex {
	t.Helper()
	l2client := builder.L2.Client
	l2info := builder.L2Info

	programAddress := deployWasm(t, ctx, auth, l2client, rustFile("storage"))

	tx := l2info.PrepareTxTo("Owner", &programAddress, l2info.TransferGas, nil, argsForStorageWrite(testhelpers.RandomHash(), testhelpers.RandomHash()))
	Require(t, l2client.SendTransaction(ctx, tx))
	receipt, err := EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)

	return arbutil.MessageIndex(receipt.BlockNumber.Uint64())
}

func waitForMessageIndex(t *testing.T, ctx context.Context, builder *NodeBuilder, pos arbutil.MessageIndex) {
	t.Helper()
	AdvanceL1(t, ctx, builder.L1.Client, builder.L1Info, 30)
	doUntil(t, 250*time.Millisecond, 20, func() bool {
		_, found, err := builder.L2.ConsensusNode.InboxTracker.FindInboxBatchContainingMessage(pos)
		Require(t, err)
		return found
	})
}

// computeExpectedState derives the expected GoGlobalState for a message
// position from the local execution engine and batch tracker.
func computeExpectedState(t *testing.T, ctx context.Context, builder *NodeBuilder, pos arbutil.MessageIndex) validator.GoGlobalState {
	t.Helper()
	result, err := builder.L2.ExecNode.ResultAtMessageIndex(pos).Await(ctx)
	Require(t, err)
	_, endPos, err := builder.L2.ConsensusNode.StatelessBlockValidator.GlobalStatePositionsAtCount(pos + 1)
	Require(t, err)
	return staker.BuildGlobalState(*result, endPos)
}

// validateBlockViaRustServer gets the ValidationInput for a block, sends it to
// the Rust validation server, and returns the GoGlobalState produced by the server.
func validateBlockViaRustServer(
	t *testing.T,
	ctx context.Context,
	builder *NodeBuilder,
	rustAddr string,
	pos arbutil.MessageIndex,
) validator.GoGlobalState {
	t.Helper()
	sbv := builder.L2.ConsensusNode.StatelessBlockValidator

	inputJSON, err := sbv.ValidationInputsAt(ctx, pos, rawdb.LocalTarget())
	Require(t, err)
	valInput, err := server_api.ValidationInputFromJson(&inputJSON)
	Require(t, err)

	moduleRoot := sbv.GetLatestWasmModuleRoot()

	valClient := connectValidationClient(t, ctx, rustAddr)
	defer valClient.Stop()

	run := valClient.Launch(valInput, moduleRoot)
	gs, err := run.Await(ctx)
	Require(t, err)
	return gs
}
