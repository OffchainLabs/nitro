// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package programs

import (
	"errors"
	"fmt"
	"sort"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
)

var ErrStylusCallDepthExceeded = errors.New("stylus call depth exceeded")

// MinNativeStackSize is the floor enforced by Wasmer's set_stack_size (must match wasmer_vm clamping
// in crates/tools/wasmer/lib/vm/src/trap/traphandlers.rs, set_stack_size()).
const MinNativeStackSize = 8 * 1024 // 8 KB

// MaxNativeStackSize is the hard cap on Wasmer coroutine stack size (must match wasmer_vm::MAX_STACK_SIZE).
const MaxNativeStackSize = 100 * 1024 * 1024 // 100 MB

const DefaultTargetDescriptionArm = "arm64-linux-unknown+neon"
const DefaultTargetDescriptionX86 = "x86_64-linux-unknown+sse4.2+lzcnt+bmi"

type StylusTargetConfig struct {
	Arm64              string   `koanf:"arm64"`
	Amd64              string   `koanf:"amd64"`
	Host               string   `koanf:"host"`
	ExtraArchs         []string `koanf:"extra-archs"`
	AllowFallback      bool     `koanf:"allow-fallback"`
	MaxOpenPages       uint16   `koanf:"max-open-pages"`
	MaxStylusCallDepth uint16   `koanf:"max-stylus-call-depth"`
	NativeStackSize    uint64   `koanf:"native-stack-size"`

	// non-configurable values
	MaxSinglepassOutputSize uint64 // `koanf:"max-singlepass-output-size"`
	MaxWavmOps              uint64 // `koanf:"max-wavm-ops"`

	wasmTargets []rawdb.WasmTarget
}

func (c *StylusTargetConfig) WasmTargets() []rawdb.WasmTarget {
	return c.wasmTargets
}

func (c *StylusTargetConfig) SetUnconfigurableDefaults() {
	// override non-configurables, if not configured from test
	if c.MaxWavmOps == 0 {
		c.MaxWavmOps = DefaultStylusTargetConfig.MaxWavmOps
	}
	if c.MaxSinglepassOutputSize == 0 {
		c.MaxSinglepassOutputSize = DefaultStylusTargetConfig.MaxSinglepassOutputSize
	}
}

func (c *StylusTargetConfig) Validate() error {
	c.SetUnconfigurableDefaults()
	targetsSet := make(map[rawdb.WasmTarget]bool, len(c.ExtraArchs))
	for _, arch := range c.ExtraArchs {
		target := rawdb.WasmTarget(arch)
		if !rawdb.IsSupportedWasmTarget(target) {
			return fmt.Errorf("unsupported architecture: %v, possible values: %s, %s, %s, %s", arch, rawdb.TargetWavm, rawdb.TargetArm64, rawdb.TargetAmd64, rawdb.TargetHost)
		}
		targetsSet[target] = true
	}
	targetsSet[rawdb.LocalTarget()] = true
	targets := make([]rawdb.WasmTarget, 0, len(c.ExtraArchs)+1)
	for target := range targetsSet {
		targets = append(targets, target)
	}
	sort.Slice(
		targets,
		func(i, j int) bool {
			return targets[i] < targets[j]
		})
	c.wasmTargets = targets
	if c.NativeStackSize != 0 {
		if c.NativeStackSize < MinNativeStackSize || c.NativeStackSize > MaxNativeStackSize {
			return fmt.Errorf("native-stack-size must be between %d and %d bytes (or 0 for default), got %d",
				MinNativeStackSize, MaxNativeStackSize, c.NativeStackSize)
		}
	}
	return nil
}

var DefaultStylusTargetConfig = StylusTargetConfig{
	Arm64:                   DefaultTargetDescriptionArm,
	Amd64:                   DefaultTargetDescriptionX86,
	Host:                    "",
	ExtraArchs:              []string{string(rawdb.TargetWavm)},
	AllowFallback:           true,
	MaxOpenPages:            128, // fits the default stylus pageLimit; 0 disables the limit
	MaxStylusCallDepth:      0,   // 0 disables the limit
	NativeStackSize:         0,   // 0 means use the Wasmer default (1 MB)
	MaxSinglepassOutputSize: 10 * 1024 * 1024,
	MaxWavmOps:              1 << 23,
}

func StylusTargetConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.String(prefix+".arm64", DefaultStylusTargetConfig.Arm64, "stylus programs compilation target for arm64 linux")
	f.String(prefix+".amd64", DefaultStylusTargetConfig.Amd64, "stylus programs compilation target for amd64 linux")
	f.String(prefix+".host", DefaultStylusTargetConfig.Host, "stylus programs compilation target for system other than 64-bit ARM or 64-bit x86")
	f.StringSlice(prefix+".extra-archs", DefaultStylusTargetConfig.ExtraArchs, fmt.Sprintf("Comma separated list of extra architectures to cross-compile stylus program to and cache in wasm store (additionally to local target). Currently must include at least %s. (supported targets: %s, %s, %s, %s)", rawdb.TargetWavm, rawdb.TargetWavm, rawdb.TargetArm64, rawdb.TargetAmd64, rawdb.TargetHost))
	f.Bool(prefix+".allow-fallback", DefaultStylusTargetConfig.AllowFallback, "if true, fall back to an alternative compiler when compilation of a Stylus program fails")
	f.Uint16(prefix+".max-open-pages", DefaultStylusTargetConfig.MaxOpenPages, "max open WASM pages per tx; exceeding the limit rejects non-on-chain calls and filters sequencer-committed txs (delayed inbox is exempt); 0 disables the limit")
	f.Uint16(prefix+".max-stylus-call-depth", DefaultStylusTargetConfig.MaxStylusCallDepth, "max number of Stylus frames simultaneously on the call stack (counts only Stylus frames; EVM frames between two Stylus frames do not decrement it); exceeding the limit rejects non-on-chain calls; 0 disables the limit")
	f.Uint64(prefix+".native-stack-size", DefaultStylusTargetConfig.NativeStackSize, "initial native stack size in bytes for Wasmer coroutines used by Stylus execution (0 = default 1MB)")
	// f.Uint64(prefix+".max-singlepass-output-size", DefaultStylusTargetConfig.MaxSinglepassOutputSize, "maximum Singlepass compiler output size in bytes per Stylus module; 0 disables the limit")
	// f.Uint64(prefix+".max-wavm-ops", DefaultStylusTargetConfig.MaxWavmOps, "maximum wavm opcodes")
}

// GetArbNodeConfig returns the ArbNodeConfig stored on the state database, or
// nil if none is set or the stored value has an unexpected type. The wrong-type
// path is a wiring bug: it logs an error and fails open so the node continues
// to run with pre-feature behavior.
func getStylusConfigOrNil(statedb vm.StateDB) *StylusTargetConfig {
	raw := statedb.Database().ArbNodeConfig()
	if raw == nil {
		return nil
	}
	cfg, ok := raw.(*StylusTargetConfig)
	if !ok {
		log.Error("ArbNodeConfig unexpected type; node-level Stylus limits inactive",
			"type", fmt.Sprintf("%T", raw))
		return nil
	}
	return cfg
}

func GetStylusConfig(statedb vm.StateDB) *StylusTargetConfig {
	if cfg := getStylusConfigOrNil(statedb); cfg != nil {
		return cfg
	}
	return &DefaultStylusTargetConfig
}
