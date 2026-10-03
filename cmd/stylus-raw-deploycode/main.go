package main

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"strconv"

	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/offchainlabs/nitro/arbcompress"
)

func main() {
	if len(os.Args) < 2 || len(os.Args) > 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <file.wasm> [dictionary]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  dictionary: 0=empty (default), 1=stylus program\n")
		fmt.Fprintf(os.Stderr, "Outputs EVM init code (hex) that deploys the compressed WASM contract.\n")
		os.Exit(1)
	}

	wasmBytes, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	dict := arbcompress.EmptyDictionary
	if len(os.Args) == 3 {
		d, err := strconv.Atoi(os.Args[2])
		if err != nil || (d != 0 && d != 1) {
			fmt.Fprintf(os.Stderr, "Invalid dictionary: %v\n", err)
			os.Exit(1)
		}
		dict = arbcompress.Dictionary(d)
	}

	code, err := arbcompress.Compress(wasmBytes, arbcompress.LEVEL_WELL, dict)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Compression error: %v\n", err)
		os.Exit(1)
	}

	// Stylus prefix: 0xEF 0xF0 0x00 <dictionary_byte>
	// Matches state.NewStylusPrefix(byte(dict)) from go-ethereum/core/state/statedb_arbitrum.go
	prefix := []byte{0xEF, 0xF0, 0x00, byte(dict)}
	code = append(prefix, code...)

	// Build EVM init code matching deployContractInitCode from system_tests/common_test.go.
	// The 42-byte prelude copies the contract code from after itself and returns it.
	deploy := []byte{byte(vm.PUSH32)}
	deploy = append(deploy, math.U256Bytes(big.NewInt(int64(len(code))))...)
	deploy = append(deploy, byte(vm.DUP1))
	deploy = append(deploy, byte(vm.PUSH1))
	deploy = append(deploy, 42) // prelude length
	deploy = append(deploy, byte(vm.PUSH1))
	deploy = append(deploy, 0)
	deploy = append(deploy, byte(vm.CODECOPY))
	deploy = append(deploy, byte(vm.PUSH1))
	deploy = append(deploy, 0)
	deploy = append(deploy, byte(vm.RETURN))
	deploy = append(deploy, code...)

	fmt.Print(hex.EncodeToString(deploy))
}
