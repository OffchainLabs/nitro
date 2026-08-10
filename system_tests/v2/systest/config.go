// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package systest

// Default configuration values + generic config-fetcher plumbing. Knobs
// that don't fit defaults are mutated via overrides applied in build_common.go
// after defaults are seeded.

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/consensus"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/headerreader"
)

var defaultForwarderConfig = gethexec.ForwarderConfig{
	ConnectionTimeout:     2 * time.Second,
	IdleConnectionTimeout: 2 * time.Second,
	MaxIdleConnections:    1,
	RedisUrl:              "",
	UpdateInterval:        10 * time.Millisecond,
	RetryInterval:         3 * time.Millisecond,
}

var defaultSequencerConfig = gethexec.SequencerConfig{
	Enable:                       true,
	MaxBlockSpeed:                10 * time.Millisecond,
	PollInterval:                 10 * time.Millisecond,
	MaxRevertGasReject:           params.TxGas + 10000,
	MaxAcceptableTimestampDelta:  time.Hour,
	SenderWhitelist:              []string{},
	Forwarder:                    defaultForwarderConfig,
	QueueSize:                    128,
	QueueTimeout:                 5 * time.Second,
	NonceCacheSize:               4,
	MaxTxDataSize:                95000,
	NonceFailureCacheSize:        1024,
	NonceFailureCacheExpiry:      time.Second,
	ExpectedSurplusSoftThreshold: "default",
	ExpectedSurplusHardThreshold: "default",
	ExpectedSurplusGasPriceMode:  "CalldataPrice",
	EnableProfiling:              false,
	ExperimentalPGA:              gethexec.DefaultPGAConfig,
}

func defaultExecConfig(t *testing.T, stateScheme containers.Option[StateScheme]) *gethexec.Config {
	t.Helper()
	cfg := gethexec.ConfigDefault
	if stateScheme.IsSome() {
		cfg.Caching.StateScheme = string(stateScheme.Unwrap())
	}
	cfg.Sequencer = defaultSequencerConfig
	cfg.ParentChainReader = headerreader.TestConfig
	cfg.ForwardingTarget = "null"
	cfg.TxPreChecker.Strictness = gethexec.TxPreCheckerStrictnessNone
	cfg.ExposeMultiGas = true
	cfg.TransactionFiltering.EnableETHCallFilter = false // v1 parity
	if err := cfg.Validate(); err != nil {
		t.Fatalf("invalid exec config: %v", err)
	}
	return cloneConfig(&cfg)
}

func configureConsensusExecutionOverRPC(execCfg *gethexec.Config, nodeCfg *arbnode.Config, stackCfg *node.Config) {
	if stackCfg.WSHost == "" {
		stackCfg.WSHost = "localhost"
	}
	stackCfg.WSModules = append(stackCfg.WSModules, consensus.RPCNamespace, execution.RPCNamespace)
	nodeCfg.RPCServer.Enable = true
	nodeCfg.RPCServer.Public = true
	nodeCfg.RPCServer.Authenticated = false
	nodeCfg.ExecutionRPCClient.URL = "self"
	execCfg.RPCServer.Enable = true
	execCfg.RPCServer.Public = true
	execCfg.RPCServer.Authenticated = false
	execCfg.ConsensusRPCClient.URL = "self"
}

type configFetcher[T any] struct {
	config atomic.Pointer[T]
}

func newConfigFetcher[T any](cfg *T) *configFetcher[T] {
	cloned := cloneConfig(cfg)
	f := &configFetcher[T]{}
	f.config.Store(cloned)
	return f
}

func (f *configFetcher[T]) Get() *T               { return f.config.Load() }
func (f *configFetcher[T]) Start(context.Context) {}
func (f *configFetcher[T]) StopAndWait()          {}
func (f *configFetcher[T]) Started() bool         { return true }

// cloneConfig deep-copies via gob, then re-validates so unexported fields
// stripped by gob (e.g. StylusTargetConfig.wasmTargets cache) get
// repopulated. CAUTION: any new config field that isn't reconstructed by
// Validate() will be silently zeroed across the clone boundary.
func cloneConfig[T any](cfg *T) *T {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(cfg); err != nil {
		panic(fmt.Sprintf("cloneConfig: gob encode %T: %v", *cfg, err))
	}
	var out T
	if err := gob.NewDecoder(&buf).Decode(&out); err != nil {
		panic(fmt.Sprintf("cloneConfig: gob decode %T: %v", out, err))
	}
	if v, ok := any(&out).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			panic(fmt.Sprintf("cloneConfig: Validate %T: %v", out, err))
		}
	}
	return &out
}
