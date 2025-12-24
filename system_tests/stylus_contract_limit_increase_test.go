// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	_ "github.com/ethereum/go-ethereum/eth/tracers/js"
	"github.com/stretchr/testify/require"

	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/arbos/programs"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/colors"
)

func TestValidFragmentedContractNoConstructor(t *testing.T) {
	builder, auth, cleanup := setupProgramTest(t, true, func(b *NodeBuilder) {
		b.WithExtraArchs(allWasmTargets)
	})
	ctx := builder.ctx
	// l2info := builder.L2Info
	l2client := builder.L2.Client
	defer cleanup()
	// arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	// Require(t, err)

	// deploy fragments
	file := rustFile("storage")
	name := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
	fragments, sourceWasm, dictType := readFragmentedContractFile(t, file, 2)
	// todo: asset there are 2 fragments
	auth.GasLimit = 32000000 // skip gas estimation
	addresses := make([]common.Address, 0, len(fragments))
	for i, fragment := range fragments {
		fragmentAddress := deployContract(t, ctx, auth, l2client, fragment)
		colors.PrintGrey(name, ": fragment contract", i, " deployed to ", fragmentAddress.Hex())
		addresses = append(addresses, fragmentAddress)
	}

	// deploy root contract
	rootContract := constructRootContract(t, uint32(len(sourceWasm)), addresses, dictType)
	rootAddress := deployContract(t, ctx, auth, l2client, rootContract)
	colors.PrintGrey(name, ": root contract deployed to ", rootAddress.Hex())

	arbWasm, err := precompilesgen.NewArbWasm(types.ArbWasmAddress, l2client)
	Require(t, err)
	auth.Value = oneEth
	tx, err := arbWasm.ActivateProgram(&auth, rootAddress)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)
}

func TestValidFragmentedContractBelowLimitNoConstructor(t *testing.T) {
	builder, auth, cleanup := setupProgramTest(t, true, func(b *NodeBuilder) {
		b.WithExtraArchs(allWasmTargets)
	})
	ctx := builder.ctx
	// l2info := builder.L2Info
	l2client := builder.L2.Client
	defer cleanup()
	// arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	// Require(t, err)

	// deploy fragments
	file := rustFile("storage")
	name := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
	fragments, sourceWasm, dictType := readFragmentedContractFile(t, file, 1)
	// todo: asset there are 2 fragments
	auth.GasLimit = 32000000 // skip gas estimation
	addresses := make([]common.Address, 0, len(fragments))
	for i, fragment := range fragments {
		fragmentAddress := deployContract(t, ctx, auth, l2client, fragment)
		colors.PrintGrey(name, ": fragment contract", i, " deployed to ", fragmentAddress.Hex())
		addresses = append(addresses, fragmentAddress)
	}

	// deploy root contract
	rootContract := constructRootContract(t, uint32(len(sourceWasm)), addresses, dictType)
	rootAddress := deployContract(t, ctx, auth, l2client, rootContract)
	colors.PrintGrey(name, ": root contract deployed to ", rootAddress.Hex())

	arbWasm, err := precompilesgen.NewArbWasm(types.ArbWasmAddress, l2client)
	Require(t, err)
	auth.Value = oneEth
	tx, err := arbWasm.ActivateProgram(&auth, rootAddress)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)
}

func TestZeroFragmentsNoConstructor(t *testing.T) {
	builder, auth, cleanup := setupProgramTest(t, true, func(b *NodeBuilder) {
		b.WithExtraArchs(allWasmTargets)
	})
	ctx := builder.ctx
	// l2info := builder.L2Info
	l2client := builder.L2.Client
	defer cleanup()
	// arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	// Require(t, err)

	// deploy fragments
	file := rustFile("storage")
	name := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
	fragments, sourceWasm, dictType := readFragmentedContractFile(t, file, 0)
	// todo: asset there are 2 fragments
	auth.GasLimit = 32000000 // skip gas estimation
	addresses := make([]common.Address, 0, len(fragments))
	for i, fragment := range fragments {
		fragmentAddress := deployContract(t, ctx, auth, l2client, fragment)
		colors.PrintGrey(name, ": fragment contract", i, " deployed to ", fragmentAddress.Hex())
		addresses = append(addresses, fragmentAddress)
	}

	// deploy root contract
	rootContract := constructRootContract(t, uint32(len(sourceWasm)), addresses, dictType)
	rootAddress := deployContract(t, ctx, auth, l2client, rootContract)
	colors.PrintGrey(name, ": root contract deployed to ", rootAddress.Hex())

	arbWasm, err := precompilesgen.NewArbWasm(types.ArbWasmAddress, l2client)
	Require(t, err)
	auth.Value = oneEth
	tx, err := arbWasm.ActivateProgram(&auth, rootAddress)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)
}

func TestInvalidTooManyFragmentsNoConstructor(t *testing.T) {
	builder, auth, cleanup := setupProgramTest(t, true, func(b *NodeBuilder) {
		b.WithExtraArchs(allWasmTargets)
	})
	ctx := builder.ctx
	// l2info := builder.L2Info
	l2client := builder.L2.Client
	defer cleanup()
	// arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	// Require(t, err)

	// deploy fragments
	file := rustFile("storage")
	name := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
	fragments, sourceWasm, dictType := readFragmentedContractFile(t, file, 3)
	// todo: asset there are 2 fragments
	auth.GasLimit = 32000000 // skip gas estimation
	addresses := make([]common.Address, 0, len(fragments))
	for i, fragment := range fragments {
		fragmentAddress := deployContract(t, ctx, auth, l2client, fragment)
		colors.PrintGrey(name, ": fragment contract", i, " deployed to ", fragmentAddress.Hex())
		addresses = append(addresses, fragmentAddress)
	}

	// deploy root contract
	rootContract := constructRootContract(t, uint32(len(sourceWasm)), addresses, dictType)
	rootAddress := deployContract(t, ctx, auth, l2client, rootContract)
	colors.PrintGrey(name, ": root contract deployed to ", rootAddress.Hex())

	arbWasm, err := precompilesgen.NewArbWasm(types.ArbWasmAddress, l2client)
	Require(t, err)
	auth.Value = oneEth
	tx, err := arbWasm.ActivateProgram(&auth, rootAddress)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l2client, tx)
	require.Error(t, err, "We can't deploy fragmented contracts which have more fragments then the current limit")
}

// Returns
// - array of fragmented contracts
// - expected wasm recovered after construction
// - dictionary type
func readFragmentedContractFile(t *testing.T, file string, fragmentCount uint16) ([][]byte, []byte, arbcompress.Dictionary) {
	t.Helper()
	name := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
	source, err := os.ReadFile(file)
	Require(t, err)

	// chose a random dictionary for testing, but keep the same files consistent
	// #nosec G115
	randDict := arbcompress.Dictionary((len(file) + len(t.Name())) % 2)

	wasmSource, err := programs.Wat2Wasm(source)
	Require(t, err)
	compressedWasm, err := arbcompress.Compress(wasmSource, arbcompress.LEVEL_WELL, randDict)
	Require(t, err)

	toKb := func(data []byte) float64 { return float64(len(data)) / 1024.0 }
	colors.PrintGrey(fmt.Sprintf("%v: len %.2fK vs %.2fK", name, toKb(compressedWasm), toKb(wasmSource)))

	prefix := state.NewStylusFragmentPrefix()

	payloadLen := len(compressedWasm)
	chunkSize := (payloadLen + int(fragmentCount) - 1) / int(fragmentCount)

	fragments := make([][]byte, 0, fragmentCount)

	for i := 0; i < int(fragmentCount); i++ {
		start := i * chunkSize
		if start >= payloadLen {
			break
		}

		end := start + chunkSize
		if end > payloadLen {
			end = payloadLen
		}

		frag := make([]byte, 0, len(prefix)+(end-start))
		frag = append(frag, prefix...)
		frag = append(frag, compressedWasm[start:end]...)

		fragments = append(fragments, frag)
	}

	return fragments, wasmSource, randDict

}

// Returns
// - stylus root contract bytecode
func constructRootContract(
	t *testing.T,
	dictionaryTypeUncompressedWasmSize uint32,
	addresses []common.Address,
	dictionaryType arbcompress.Dictionary,
) []byte {
	t.Helper()

	// prefix 3 bytes + dict 1 byte + length 4 bytes + len(address) * 20 bytes
	contract := make([]byte, 0, 3+1+4+len(addresses)*common.AddressLength)
	fmt.Println("bluebirdduck", len(contract))
	contract = append(contract, state.NewStylusRootPrefix(byte(dictionaryType))...)
	fmt.Println("bluebirdduck 1", len(contract))

	var sizeBuf [4]byte
	binary.BigEndian.PutUint32(sizeBuf[:], dictionaryTypeUncompressedWasmSize)
	contract = append(contract, sizeBuf[:]...)
	fmt.Println("bluebirdduck 3", len(contract))

	for _, addr := range addresses {
		contract = append(contract, addr.Bytes()...)
		fmt.Println("bluebirdduck 4", len(contract))
	}

	return contract
}
