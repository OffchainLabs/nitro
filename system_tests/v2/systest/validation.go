// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package systest

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/node"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/util/testhelpers/env"
	"github.com/offchainlabs/nitro/validator/server_api"
	"github.com/offchainlabs/nitro/validator/server_common"
	"github.com/offchainlabs/nitro/validator/valnode"
)

// validationBackstop bounds validateToHead's wait when e.Ctx has no deadline.
const validationBackstop = time.Hour

// configureValidation enables block validation on a node against the shared
// valnode. Errors if execCfg already pins an incompatible state scheme.
func configureValidation(execCfg *gethexec.Config, nodeCfg *arbnode.Config, valnodeURL string) error {
	if s := execCfg.Caching.StateScheme; s != "" && s != string(validationScheme) {
		return fmt.Errorf("systest: validation requires %s state scheme; remove the conflicting exec-config override (%q)", validationScheme, s)
	}
	if len(nodeCfg.BlockValidator.ValidationServerConfigs) == 0 {
		return fmt.Errorf("systest: validation requires a ValidationServerConfigs entry")
	}
	execCfg.Caching.Archive = true
	execCfg.Caching.StateScheme = string(validationScheme)
	nodeCfg.BlockValidator.Enable = true
	nodeCfg.BlockValidator.ValidationServerConfigs[0].URL = valnodeURL
	return nil
}

// validatorPool owns the shared JIT valnode for the worker binary.
type validatorPool struct {
	mu     sync.Mutex
	cancel context.CancelFunc
	node   *valnodeHandle
}

type valnodeHandle struct {
	stack   *node.Node
	valNode *valnode.ValidationNode
	wsURL   string
}

var validators = &validatorPool{}

// endpoint returns the shared valnode's WS URL, starting it once.
func (p *validatorPool) endpoint() (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.node != nil {
		return p.node.wsURL, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	h, err := startValnode(ctx)
	if err != nil {
		cancel()
		return "", err
	}
	p.cancel = cancel
	p.node = h
	return h.wsURL, nil
}

func (p *validatorPool) shutdown() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.node != nil {
		p.node.valNode.Stop() // join spawner goroutines before closing the stack
		closeStack("valnode", p.node.stack)
		p.node = nil
	}
	if p.cancel != nil {
		p.cancel()
	}
}

// machinesAvailable reports whether any wasm machine is on disk.
func machinesAvailable() (bool, error) {
	locator, err := server_common.NewMachineLocator("")
	if err != nil {
		return false, fmt.Errorf("locating wasm machines: %w", err)
	}
	return len(locator.ModuleRoots()) > 0, nil
}

func mustEnableValidation(t *testing.T, execCfg *gethexec.Config, nodeCfg *arbnode.Config, missingMsg string) {
	t.Helper()
	ok, err := machinesAvailable()
	if err != nil {
		t.Fatalf("systest: machinesAvailable: %v", err)
	}
	if !ok {
		t.Fatalf("systest: %s; run `make build-replay-env`", missingMsg)
	}
	url, err := validators.endpoint()
	if err != nil {
		t.Fatalf("systest: start valnode: %v", err)
	}
	if err := configureValidation(execCfg, nodeCfg, url); err != nil {
		t.Fatal(err)
	}
}

// startValnode brings up a standalone JIT validation node.
func startValnode(ctx context.Context) (*valnodeHandle, error) {
	stackConf := node.DefaultConfig
	stackConf.HTTPPort = 0
	stackConf.HTTPHost = ""
	stackConf.DataDir = ""
	stackConf.WSHost = "127.0.0.1"
	stackConf.WSPort = 0
	stackConf.WSModules = []string{server_api.Namespace}
	stackConf.P2P.NoDiscovery = true
	stackConf.P2P.ListenAddr = ""
	stackConf.DBEngine = env.GetTestDatabaseEngine()
	valnode.EnsureValidationExposedViaAuthRPC(&stackConf)

	stack, err := node.New(&stackConf)
	if err != nil {
		return nil, fmt.Errorf("valnode node.New: %w", err)
	}
	conf := valnode.TestValidationConfig
	conf.UseJit = true
	valNode, err := valnode.CreateValidationNode(func() *valnode.Config { return &conf }, stack, nil)
	if err != nil {
		closeStack("valnode", stack)
		return nil, fmt.Errorf("CreateValidationNode: %w", err)
	}
	if err := stack.Start(); err != nil {
		closeStack("valnode", stack)
		return nil, fmt.Errorf("valnode stack.Start: %w", err)
	}
	if err := valNode.Start(ctx); err != nil {
		closeStack("valnode", stack)
		return nil, fmt.Errorf("valnode start: %w", err)
	}
	return &valnodeHandle{stack: stack, valNode: valNode, wsURL: stack.WSEndpoint()}, nil
}

// validateToHead is the WithValidation post-hook: it waits for the node's block
// validator to validate every message the scenario produced.
func validateToHead(e *Env) error {
	if !e.Spec.Validate {
		return nil
	}
	if e.L2 == nil || e.L2.Consensus == nil || e.L2.Consensus.BlockValidator == nil {
		return fmt.Errorf("validation requested but block validator not configured")
	}
	bv := e.L2.Consensus.BlockValidator
	ctx, cancel := context.WithTimeout(e.Ctx, validationBackstop)
	defer cancel()
	// Trailing internal-tx-only blocks are never validated on their own; wait
	// only up to the last block carrying a scenario tx (v1 block_validator_test parity).
	block, err := e.L2.Client.BlockByNumber(ctx, nil)
	if err != nil {
		return fmt.Errorf("validation head: %w", err)
	}
	for !hasUsefulTx(block) {
		if block.NumberU64() == 0 {
			return fmt.Errorf("validation requested but the chain produced no messages")
		}
		block, err = e.L2.Client.BlockByHash(ctx, block.ParentHash())
		if err != nil {
			return fmt.Errorf("validation walk-back: %w", err)
		}
	}
	t, ok := e.t.(*testing.T)
	if !ok {
		return fmt.Errorf("block validation wait requires *testing.T, got %T", e.t)
	}
	// Block number equals message index on these chains (genesis = message 0).
	pos := arbutil.MessageIndex(block.NumberU64())
	if bv.WaitForPos(t, ctx, pos, validationBackstop) {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("block validation stalled at %d, want %d: %w", bv.GetValidated(), pos, err)
	}
	return fmt.Errorf("block validation stalled at %d, want %d", bv.GetValidated(), pos)
}

// hasUsefulTx reports whether the block carries any non-internal transaction.
func hasUsefulTx(block *types.Block) bool {
	for _, tx := range block.Transactions() {
		if tx.Type() != types.ArbitrumInternalTxType {
			return true
		}
	}
	return false
}
