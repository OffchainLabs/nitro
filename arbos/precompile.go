//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbos

import (
	"log"
	"math/big"
	"strings"
	"reflect"
	"unicode"

	templates "github.com/offchainlabs/arbstate/precompiles/go"
	pre "github.com/offchainlabs/arbstate/arbos/precompiles"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/state"
)

type ArbosPrecompile interface {
	GasToCharge(input []byte) uint64

	// Important fields: evm.StateDB and evm.Config.Tracer
	// NOTE: if precompileAddress != actingAsAddress, watch out! This is a delegatecall or callcode, so caller might be wrong. In that case, unless this precompile is pure, it should probably revert.
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

type Precompile struct {
	methods map[[4]byte]PrecompileMethod
}

type PrecompileMethod struct {
	name string
}

// Make a precompile for the given hardhat-to-geth bindings, ensuring that the implementer
// defines each method.
func makePrecompile(metadata *bind.MetaData, implementer interface{}) ArbosPrecompile {
	source, err := abi.JSON(strings.NewReader(metadata.ABI))
	if err != nil {
		log.Fatal("Bad ABI")
	}

	contract := reflect.TypeOf(implementer).Name()
	methods := make(map[[4]byte]PrecompileMethod)

	for _, method := range source.Methods {

		name := method.RawName
		capitalize := string(unicode.ToUpper(rune(name[0])))
		name = capitalize + name[1:]

		if len(method.ID) != 4 {
			log.Fatal("Method ID isn't 4 bytes")
		}
		id := *(*[4]byte)(method.ID)

		raw, ok := reflect.TypeOf(implementer).MethodByName(name)
		if !ok {
			log.Fatal("Precompile ", contract, " must implement ", name)
		}

		var needs = []reflect.Type{
			reflect.TypeOf(implementer),  // the contract itself
		}

		switch method.StateMutability {
		case "pure":
		case "view":
			needs = append(needs, reflect.TypeOf(&state.StateDB{}))
		case "nonpayable":
			needs = append(needs, reflect.TypeOf(&state.StateDB{}))
		case "payable":
			needs = append(needs, reflect.TypeOf(&state.StateDB{}))
			needs = append(needs, reflect.TypeOf(&big.Int{}))
		default:
			log.Fatal("Unknown state mutability ", method.StateMutability)
		}

		for _, arg := range method.Inputs {
			needs = append(needs, arg.Type.GetType())
		}

		signature := raw.Type
		
		if len(needs) != signature.NumIn() {
			log.Fatal("Precompile ", contract, "'s ", name, " doesn't have the args ", needs)
		}
		




		/*signature := raw.Type
		if len(method.Inputs) != signature.NumIn() {
			log.Fatal("Precompile ", contract, "'s ", name, " needs ", len(method.Inputs), " args")
		}*/

		/*for i := 0; i < len(method.Inputs); i++ {
			need := method.Inputs[i]
			have := signature.In(i)
			
		}*/
		
		methods[id] = PrecompileMethod{
			name,
		}
	}

	return Precompile{
		methods,
	}
}

func Precompiles() map[common.Address]ArbosPrecompile {
	return map[common.Address]ArbosPrecompile {
		addr("0x100"): makePrecompile(templates.ArbSysMetaData, pre.ArbSys{}),
		/*addr("0x102"): makePrecompile(pre.ArbAddressTableMetaData, Test{}),
		addr("0x109"): makePrecompile(pre.ArbAggregatorMetaData, Test{}),
		addr("0x103"): makePrecompile(pre.ArbBLSMetaData, Test{}),
		addr("0x104"): makePrecompile(pre.ArbFunctionTableMetaData, Test{}),
		addr("0x108"): makePrecompile(pre.ArbGasInfoMetaData, Test{}),
		addr("0x065"): makePrecompile(pre.ArbInfoMetaData, Test{}),
		addr("0x105"): makePrecompile(pre.ArbosTestMetaData, Test{}),
		addr("0x107"): makePrecompile(pre.ArbOwnerMetaData, Test{}),
		addr("0x110"): makePrecompile(pre.ArbRetryableTxMetaData, Test{}),
		addr("0x111"): makePrecompile(pre.ArbStatisticsMetaData, Test{}),*/
	}
}

func addr(s string) common.Address {
	return common.HexToAddress(s)
}

func (p Precompile) GasToCharge(input []byte) uint64 {
	return 0
}

func (p Precompile) Call(
	input []byte,
	precompileAddress common.Address,
	actingAsAddress common.Address,
	caller common.Address,
	value *big.Int,
	readOnly bool,
	evm *vm.EVM,
) (output []byte, err error) {
	return nil, nil
}
