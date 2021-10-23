//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	templates "github.com/offchainlabs/arbstate/solgen/go2"
)

type ArbosPrecompile interface {
	GasToCharge(input []byte) uint64

	// Important fields: evm.StateDB and evm.Config.Tracer
	// NOTE: if precompileAddress != actingAsAddress, watch out!
	// This is a delegatecall or callcode, so caller might be wrong.
	// In that case, unless this precompile is pure, it should probably revert.
	Call(
		input []byte,
		precompileAddress common.Address,
		actingAsAddress common.Address,
		caller common.Address,
		value *big.Int,
		readOnly bool,
		evm *vm.EVM,
	) (output []byte, err error)
}

func addr(s string) common.Address {
	return common.HexToAddress(s)
}

func Precompiles() map[common.Address]ArbosPrecompile {
	return map[common.Address]ArbosPrecompile{
		addr("0x64"): templates.NewArbSys(ArbSys{}),
		addr("0x65"): templates.NewArbInfo(ArbInfo{}),
		addr("0x66"): templates.NewArbAddressTable(ArbAddressTable{}),
		addr("0x67"): templates.NewArbBLS(ArbBLS{}),
		addr("0x68"): templates.NewArbFunctionTable(ArbFunctionTable{}),
		addr("0x69"): templates.NewArbosTest( ArbosTest{}),
		addr("0x6b"): templates.NewArbOwner(ArbOwner{}),
		addr("0x6c"): templates.NewArbGasInfo(ArbGasInfo{}),
		addr("0x6d"): templates.NewArbAggregator( ArbAggregator{}),
		addr("0x6e"): templates.NewArbRetryableTx( ArbRetryableTx{}),
		addr("0x6f"): templates.NewArbStatistics( ArbStatistics{}),
	}
}
