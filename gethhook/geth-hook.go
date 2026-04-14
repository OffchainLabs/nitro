// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethhook

import (
	"errors"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/arbitrum/multigas"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/precompiles"
)

// arbOwnerPrecompile holds the *OwnerPrecompile retrieved from the precompile map during init().
// ExecutionNode.Initialize() configures it later when the node config is available.
var arbOwnerPrecompile *precompiles.OwnerPrecompile

func GetOwnerPrecompile() *precompiles.OwnerPrecompile {
	return arbOwnerPrecompile
}

type ArbosPrecompileWrapper struct {
	inner precompiles.ArbosPrecompile
}

func (p ArbosPrecompileWrapper) RequiredGas(input []byte) uint64 {
	panic("Non-advanced precompile method called")
}

func (p ArbosPrecompileWrapper) Run(input []byte) ([]byte, error) {
	panic("Non-advanced precompile method called")
}

func (p ArbosPrecompileWrapper) Name() string {
	return p.inner.Name()
}

func (p ArbosPrecompileWrapper) RunAdvanced(
	input []byte,
	gasSupplied uint64,
	info *vm.AdvancedPrecompileCall,
) (ret []byte, gasLeft uint64, usedMultiGas multigas.MultiGas, err error) {

	// Precompiles don't actually enter evm execution like normal calls do,
	// so we need to increment the depth here to simulate the callstack change.
	info.Evm.IncrementDepth()
	defer info.Evm.DecrementDepth()

	return p.inner.Call(
		input, info.ActingAsAddress,
		info.Caller, info.Value, info.ReadOnly, gasSupplied, info.Evm,
	)
}

func init() {
	core.ReadyEVMForL2 = func(evm *vm.EVM, msg *core.Message) {
		if evm.ChainConfig().IsArbitrum() {
			evm.ProcessingHook = arbos.NewTxProcessor(evm, msg)
		}
	}

	// Register each Arbitrum precompile at its declared ArbosVersion
	// so CALLs on pre-activation blocks fall through to account
	// dispatch, matching consensus.
	precompileErrors := make(map[[4]byte]abi.Error)
	arbosPrecompiles := precompiles.Precompiles()
	if ownerPC, ok := arbosPrecompiles[types.ArbOwnerAddress].(*precompiles.OwnerPrecompile); ok {
		arbOwnerPrecompile = ownerPC
	} else {
		panic("ArbOwner precompile is not an *OwnerPrecompile, disable-arbowner-ethcall flag will not work")
	}
	for addr, precompile := range arbosPrecompiles {
		for _, errABI := range precompile.Precompile().GetErrorABIs() {
			precompileErrors[[4]byte(errABI.ID.Bytes())] = errABI
		}
		wrapped := ArbosPrecompileWrapper{inner: precompile}
		registerArbOSPrecompile(precompile.Precompile().ArbosVersion(), addr, wrapped)
	}

	// List + activation versions live in ethereum_precompile_sets.go;
	// TestAllUpstreamPrecompileSetsCataloged catches any upstream set
	// we forgot to catalog.
	registerAllEthereumPrecompileSets()

	// Single authoritative install site; vm.SetArbOSPrecompileResolver
	// panics on a second call.
	vm.SetArbOSPrecompileResolver(arbOSPrecompilesFor)

	core.RenderRPCError = func(data []byte) error {
		if len(data) < 4 {
			return nil
		}
		var id [4]byte
		copy(id[:], data[:4])
		errABI, found := precompileErrors[id]
		if !found {
			return nil
		}
		rendered, err := precompiles.RenderSolError(errABI, data)
		if err != nil {
			log.Warn("failed to render rpc error", "err", err)
			return nil
		}
		return errors.New(rendered)
	}
}

// RequireHookedGeth does nothing, but forces an import to let the init function run
func RequireHookedGeth() {}
