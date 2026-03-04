// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// race detection makes things slow and miss timeouts
//go:build !race

package arbtest

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/offchainlabs/nitro/validator"
	"github.com/offchainlabs/nitro/validator/client"
	"github.com/offchainlabs/nitro/validator/server_api"
	"github.com/offchainlabs/nitro/validator/server_arb"
	"github.com/offchainlabs/nitro/validator/valnode"
)

// proxySpawner wraps a real ValidationSpawner (typically a ValidationClient
// connected to a real validation node) and returns timeout errors for the first N
// validation attempts. After that, it delegates to the inner spawner which produces
// correct results. This simulates transient timeout conditions during validation.
type proxySpawner struct {
	remainingTimeouts atomic.Int64
	inner             validator.ValidationSpawner
}

// ---------------------------------------------------------------------------------------------------------------------
// ---- ValidationSpawner interface implementation ---------------------------------------------------------------------
// ---------------------------------------------------------------------------------------------------------------------
func (s *proxySpawner) Launch(entry *validator.ValidationInput, moduleRoot common.Hash) validator.ValidationRun {
	if s.remainingTimeouts.Add(-1) >= 0 {
		run := &mockValRun{
			Promise: containers.NewPromise[validator.GoGlobalState](nil),
			root:    moduleRoot,
		}
		run.ProduceError(context.DeadlineExceeded)
		return run
	}
	return s.inner.Launch(entry, moduleRoot)
}
func (s *proxySpawner) WasmModuleRoots() ([]common.Hash, error) { return s.inner.WasmModuleRoots() }
func (s *proxySpawner) Start(c context.Context) error           { return s.inner.Start(c) }
func (s *proxySpawner) Stop()                                   { s.inner.Stop() }
func (s *proxySpawner) Name() string                            { return s.inner.Name() }
func (s *proxySpawner) StylusArchs() []rawdb.WasmTarget         { return s.inner.StylusArchs() }
func (s *proxySpawner) Capacity() int                           { return s.inner.Capacity() }

// ---------------------------------------------------------------------------------------------------------------------

// createProxyValidationNode creates an RPC server that serves the
// proxySpawner as a validation node. The block validator connects to
// this node for validation requests.
func createProxyValidationNode(t *testing.T, ctx context.Context, spawner *proxySpawner) *node.Node {
	stackConf := node.DefaultConfig
	stackConf.HTTPPort = 0
	stackConf.DataDir = ""
	stackConf.WSHost = "127.0.0.1"
	stackConf.WSPort = 0
	stackConf.WSModules = []string{server_api.Namespace}
	stackConf.P2P.NoDiscovery = true
	stackConf.P2P.ListenAddr = ""

	stack, err := node.New(&stackConf)
	Require(t, err)

	configFetcher := func() *server_arb.ArbitratorSpawnerConfig {
		return &server_arb.DefaultArbitratorSpawnerConfig
	}
	// Use mockSpawner for the ExecutionSpawner parameter — it won't be called
	// during normal block validation (only used for BOLD execution runs).
	serverAPI := valnode.NewExecutionServerAPI(spawner, &mockSpawner{}, configFetcher)

	valAPIs := []rpc.API{{
		Namespace:     server_api.Namespace,
		Version:       "1.0",
		Service:       serverAPI,
		Public:        true,
		Authenticated: false,
	}}
	stack.RegisterAPIs(valAPIs)

	err = stack.Start()
	Require(t, err)

	serverAPI.Start(ctx)

	go func() {
		<-ctx.Done()
		stack.Close()
		serverAPI.StopOnly()
	}()

	return stack
}

// TestBlockValidatorTimeoutRetry verifies that timeout errors during validation
// do not crash the node. With FailureIsFatal=true (the default), validation
// failures normally crash the node. But timeout errors should be retried.
//
// Architecture: The test creates a proxy validation node that sits between the
// block validator and a real validation node. The proxy returns timeout errors
// for the first N requests, then forwards to the real node for correct results.
func TestBlockValidatorTimeoutRetry(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	// PathDB is not supported for block validation
	builder.RequireScheme(t, rawdb.HashScheme)
	cleanup := builder.Build(t)
	defer cleanup()

	// Create a real validation node that can correctly validate blocks.
	_, realValStack := createTestValidationNode(t, ctx, &valnode.TestValidationConfig)

	// Create a validation client connected to the real validation node.
	realClient := client.NewValidationClient(StaticFetcherFrom(t, &rpcclient.TestClientConfig), realValStack)
	Require(t, realClient.Start(ctx))

	// Create proxy spawner that returns DeadlineExceeded for the first 3 validation attempts, then delegates to the real validation client.
	proxy := &proxySpawner{inner: realClient}
	proxy.remainingTimeouts.Store(3)
	proxyStack := createProxyValidationNode(t, ctx, proxy)

	// Prepare the second node (with the block validator).
	validatorConfig := arbnode.ConfigDefaultL1NonSequencerTest()
	validatorConfig.BlockValidator.Enable = true
	validatorConfig.BlockValidator.FailureIsFatal = true
	validatorConfig.BlockValidator.ValidationSpawningAllowedAttempts = 1
	configByValidationNode(validatorConfig, proxyStack)

	testClientB, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{nodeConfig: validatorConfig})
	defer cleanupB()

	// Send tx and wait until it's observed by the B node. If the block validator is not retrying on timeouts, it will
	// fail to validate the block and the tx will never be observed.
	block := checkBatchPosting(t, ctx, builder, testClientB.Client)

	// Double check that the block has been validated successfully.
	timeout := getDeadlineTimeout(t, time.Second*10)
	if !testClientB.ConsensusNode.BlockValidator.WaitForPos(t, ctx, arbutil.MessageIndex(block.Uint64()), timeout) {
		Fatal(t, "did not validate the block - timeout errors should have been retried")
	}
}
