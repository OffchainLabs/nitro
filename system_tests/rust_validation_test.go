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

	"github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/offchainlabs/nitro/validator/client"
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

	pos := deployStylusContractAndCall(t, ctx, builder, auth)
	expectedState := computeExpectedState(t, ctx, builder, pos)
	actualState := validateBlockViaRustServer(t, ctx, builder, rvAddr, pos)

	if expectedState != actualState {
		Fatal(t, "GoGlobalState mismatch",
			"\n  expected:", expectedState,
			"\n  actual:  ", actualState)
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
