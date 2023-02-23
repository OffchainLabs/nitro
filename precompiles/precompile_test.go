// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package precompiles

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/state"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
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

	zeroHash := crypto.Keccak256([]byte{0x00})
	trueHash := common.Hash{}.Bytes()
	falseHash := common.Hash{}.Bytes()
	trueHash[31] = 0x01

	var data []byte
	payload := [][]byte{
		method.template.ID, // select the `Events` method
		falseHash,          // set the flag to false
		zeroHash,           // set the value to something known
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

	burnedToStorage := params.WarmStorageReadCostEIP2929            // the ArbOS version check costs a read
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

	if !bytes.Equal(basicTopics[1].Bytes(), zeroHash) || !bytes.Equal(mixedTopics[2].Bytes(), zeroHash) {
		Fail(t, "indexing a bytes32 didn't work")
	}
	if !bytes.Equal(mixedTopics[1].Bytes(), falseHash) {
		Fail(t, "indexing a bool didn't work")
	}
	if !bytes.Equal(mixedTopics[3].Bytes(), caller.Hash().Bytes()) {
		Fail(t, "indexing an address didn't work")
	}

	ArbDebugInfo, cerr := templates.NewArbDebug(common.Address{}, nil)
	basic, berr := ArbDebugInfo.ParseBasic(*logs[0])
	mixed, merr := ArbDebugInfo.ParseMixed(*logs[1])
	if cerr != nil || berr != nil || merr != nil {
		Fail(t, "failed to parse event logs", "\nprecompile:", cerr, "\nbasic:", berr, "\nmixed:", merr)
	}

	if basic.Flag != true || !bytes.Equal(basic.Value[:], zeroHash) {
		Fail(t, "event Basic's data isn't correct")
	}
	if mixed.Flag != false || mixed.Not != true || !bytes.Equal(mixed.Value[:], zeroHash) {
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

type FatalBurner struct {
	t       *testing.T
	count   uint64
	gasLeft uint64
}

func NewFatalBurner(t *testing.T, limit uint64) FatalBurner {
	return FatalBurner{t, 0, limit}
}

func (burner FatalBurner) Burn(amount uint64) error {
	burner.t.Helper()
	burner.count += 1
	if burner.gasLeft < amount {
		Fail(burner.t, "out of gas after", burner.count, "burns")
	}
	burner.gasLeft -= amount
	return nil
}
