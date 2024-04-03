// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package precompiles

import (
	"fmt"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbos/storage"
	templates "github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/arbmath"
)

func TestEvents(t *testing.T) {
	blockNumber := 1024

	evm := newMockEVMForTesting()
	evm.Context.BlockNumber = big.NewInt(int64(blockNumber))

	debugContractAddr := common.HexToAddress("ff")
	contract := Precompiles()[debugContractAddr]

	var method *PrecompileMethod
	for _, available := range contract.Precompile().methods {
		if available.name == "Events" {
			method = available
			break
		}
	}

	zeroHash := crypto.Keccak256Hash([]byte{0x00})
	falseHash := common.Hash{}

	var data []byte
	payload := [][]byte{
		method.template.ID, // select the `Events` method
		falseHash.Bytes(),  // set the flag to false
		zeroHash.Bytes(),   // set the value to something known
	}
	for _, bytes := range payload {
		data = append(data, bytes...)
	}

	caller := common.HexToAddress("aaaaaaaabbbbbbbbccccccccdddddddd")
	number := big.NewInt(0x9364)

	output, gasLeft, err := contract.Call(
		data,
		debugContractAddr,
		debugContractAddr,
		caller,
		number,
		false,
		^uint64(0),
		evm,
	)
	Require(t, err, "call failed")

	burnedToStorage := storage.StorageReadCost                      // the ArbOS version check costs a read
	burnedToArgs := arbmath.WordsForBytes(32+32) * params.CopyGas   // bool and a bytes32
	burnedToResult := arbmath.WordsForBytes(32+32) * params.CopyGas // addr and a huge
	burnedToEvents := ^uint64(0) - gasLeft - (burnedToStorage + burnedToArgs + burnedToResult)

	if burnedToEvents != 3768 {
		Fail(t, "burned", burnedToEvents, "instead of", 3768, "gas")
	}

	outputAddr := common.BytesToAddress(output[:32])
	outputData := new(big.Int).SetBytes(output[32:])

	if outputAddr != caller {
		Fail(t, "unexpected output address", outputAddr, "instead of", caller)
	}
	if outputData.Cmp(number) != 0 {
		Fail(t, "unexpected output number", outputData, "instead of", number)
	}

	//nolint:errcheck
	logs := evm.StateDB.(*state.StateDB).Logs()
	for _, log := range logs {
		if log.Address != debugContractAddr {
			Fail(t, "address mismatch:", log.Address, "vs", debugContractAddr)
		}
		if log.BlockNumber != uint64(blockNumber) {
			Fail(t, "block number mismatch:", log.BlockNumber, "vs", blockNumber)
		}
		t.Log("topic", len(log.Topics), log.Topics)
		t.Log("data ", len(log.Data), log.Data)
	}

	basicTopics := logs[0].Topics
	mixedTopics := logs[1].Topics

	if basicTopics[1] != zeroHash || mixedTopics[2] != zeroHash {
		Fail(t, "indexing a bytes32 didn't work")
	}
	if mixedTopics[1] != falseHash {
		Fail(t, "indexing a bool didn't work")
	}
	if mixedTopics[3] != common.BytesToHash(caller.Bytes()) {
		Fail(t, "indexing an address didn't work")
	}

	ArbDebugInfo, cerr := templates.NewArbDebug(common.Address{}, nil)
	basic, berr := ArbDebugInfo.ParseBasic(*logs[0])
	mixed, merr := ArbDebugInfo.ParseMixed(*logs[1])
	if cerr != nil || berr != nil || merr != nil {
		Fail(t, "failed to parse event logs", "\nprecompile:", cerr, "\nbasic:", berr, "\nmixed:", merr)
	}

	if basic.Flag != true || basic.Value != zeroHash {
		Fail(t, "event Basic's data isn't correct")
	}
	if mixed.Flag != false || mixed.Not != true || mixed.Value != zeroHash {
		Fail(t, "event Mixed's data isn't correct")
	}
	if mixed.Conn != debugContractAddr || mixed.Caller != caller {
		Fail(t, "event Mixed's data isn't correct")
	}
}

func TestEventCosts(t *testing.T) {
	debugContractAddr := common.HexToAddress("ff")
	contract := Precompiles()[debugContractAddr]

	//nolint:errcheck
	impl := contract.Precompile().implementer.Interface().(*ArbDebug)

	testBytes := [...][]byte{
		nil,
		{0x01},
		{0x02, 0x32, 0x24, 0x48},
		common.Hash{}.Bytes(),
		common.Hash{}.Bytes(),
	}
	testBytes[4] = append(testBytes[4], common.Hash{}.Bytes()...)

	test := func(a bool, b addr, c huge, d hash, e []byte) uint64 {
		cost, err := impl.StoreGasCost(a, b, c, d, e)
		Require(t, err)
		return cost
	}

	tests := [...]uint64{
		test(true, addr{}, big.NewInt(24), common.Hash{}, testBytes[0]),
		test(false, addr{}, big.NewInt(8), common.Hash{}, testBytes[1]),
		test(false, addr{}, big.NewInt(8), common.Hash{}, testBytes[2]),
		test(true, addr{}, big.NewInt(32), common.Hash{}, testBytes[3]),
		test(true, addr{}, big.NewInt(64), common.Hash{}, testBytes[4]),
	}

	expected := [5]uint64{}

	for i, bytes := range testBytes {
		baseCost := params.LogGas + 3*params.LogTopicGas
		addrCost := 32 * params.LogDataGas
		hashCost := 32 * params.LogDataGas

		sizeBytes := 32
		offsetBytes := 32
		storeBytes := sizeBytes + offsetBytes + len(bytes)
		storeBytes = storeBytes + 31 - (storeBytes+31)%32 // round up to a multiple of 32
		storeCost := uint64(storeBytes) * params.LogDataGas

		expected[i] = baseCost + addrCost + hashCost + storeCost
	}

	if tests != expected {
		Fail(t, "Events are mispriced\nexpected:", expected, "\nbut have:", tests)
	}
}

func TestPrecompilesPerArbosVersion(t *testing.T) {
	// Set up a logger in case log.Crit is called by Precompiles()
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(log.LvlWarn)
	log.Root().SetHandler(glogger)

	expectedNewMethodsPerArbosVersion := map[uint64]int{
		0:  89,
		5:  3,
		10: 2,
		11: 4,
		20: 8 + 27, // 27 for stylus
	}

	precompiles := Precompiles()
	newMethodsPerArbosVersion := make(map[uint64]int)
	for _, precompile := range precompiles {
		for _, method := range precompile.Precompile().methods {
			newMethodsPerArbosVersion[method.arbosVersion]++
		}
	}

	if len(expectedNewMethodsPerArbosVersion) != len(newMethodsPerArbosVersion) {
		t.Errorf("expected %v ArbOS versions with new precompile methods but got %v", len(expectedNewMethodsPerArbosVersion), len(newMethodsPerArbosVersion))
	}
	for version, count := range newMethodsPerArbosVersion {
		fmt.Printf("got %v version count %v\n", version, count)
		if expectedNewMethodsPerArbosVersion[version] != count {
			t.Errorf("expected %v new precompile methods for ArbOS version %v but got %v", expectedNewMethodsPerArbosVersion[version], version, count)
		}
	}
}
