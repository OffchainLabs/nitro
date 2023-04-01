// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package arbtest

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/colors"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestProgramKeccakJIT(t *testing.T) {
	keccakTest(t, true)
}

func TestProgramKeccakArb(t *testing.T) {
	keccakTest(t, false)
}

func keccakTest(t *testing.T, jit bool) {
	ctx, node, _, l2client, auth, programAddress, cleanup := setupProgramTest(t, rustFile("keccak"), jit)
	defer cleanup()

	preimage := []byte("°º¤ø,¸,ø¤°º¤ø,¸,ø¤°º¤ø,¸ nyan nyan ~=[,,_,,]:3 nyan nyan")
	correct := crypto.Keccak256Hash(preimage)

	args := []byte{0x01} // keccak the preimage once
	args = append(args, preimage...)

	timed(t, "execute", func() {
		result := sendContractCall(t, ctx, programAddress, l2client, args)
		if len(result) != 32 {
			Fail(t, "unexpected return result: ", "result", result)
		}
		hash := common.BytesToHash(result)
		if hash != correct {
			Fail(t, "computed hash mismatch", hash, correct)
		}
		colors.PrintGrey("keccak(x) = ", hash)
	})

	ensure := func(tx *types.Transaction, err error) *types.Receipt {
		t.Helper()
		Require(t, err)
		receipt, err := EnsureTxSucceeded(ctx, l2client, tx)
		Require(t, err)
		return receipt
	}

	// do a mutating call for proving's sake
	_, tx, mock, err := mocksgen.DeployProgramTest(&auth, l2client)
	ensure(tx, err)
	ensure(mock.CallKeccak(&auth, programAddress, args))

	validateBlocks(t, 1, ctx, node, l2client)
}

func TestProgramErrorsJIT(t *testing.T) {
	errorTest(t, true)
}

func TestProgramErrorsArb(t *testing.T) {
	errorTest(t, false)
}

func errorTest(t *testing.T, jit bool) {
	ctx, node, l2info, l2client, _, programAddress, cleanup := setupProgramTest(t, rustFile("fallible"), jit)
	defer cleanup()

	// ensure tx passes
	tx := l2info.PrepareTxTo("Owner", &programAddress, l2info.TransferGas, nil, []byte{0x01})
	Require(t, l2client.SendTransaction(ctx, tx))
	_, err := EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)

	// ensure tx fails
	tx = l2info.PrepareTxTo("Owner", &programAddress, l2info.TransferGas, nil, []byte{0x00})
	Require(t, l2client.SendTransaction(ctx, tx))
	receipt, err := WaitForTx(ctx, l2client, tx.Hash(), 5*time.Second)
	Require(t, err)
	if receipt.Status != types.ReceiptStatusFailed {
		Fail(t, "call should have failed")
	}

	validateBlocks(t, 7, ctx, node, l2client)
}

func TestProgramStorage(t *testing.T) {
	ctx, _, l2info, l2client, _, programAddress, cleanup := setupProgramTest(t, rustFile("storage"), true)
	defer cleanup()

	ensure := func(tx *types.Transaction, err error) *types.Receipt {
		t.Helper()
		Require(t, err)
		receipt, err := EnsureTxSucceeded(ctx, l2client, tx)
		Require(t, err)
		return receipt
	}

	key := testhelpers.RandomHash()
	value := testhelpers.RandomHash()

	storeArgs := []byte{0x01}
	storeArgs = append(storeArgs, key.Bytes()...)
	storeArgs = append(storeArgs, value.Bytes()...)

	tx := l2info.PrepareTxTo("Owner", &programAddress, l2info.TransferGas, nil, storeArgs)
	ensure(tx, l2client.SendTransaction(ctx, tx))

	storedBytes, err := l2client.StorageAt(ctx, programAddress, key, nil)
	Require(t, err)
	storedValue := common.BytesToHash(storedBytes)
	if value != storedValue {
		Fail(t, "wrong value", value, storedValue)
	}

	// TODO: enable validation when prover side is PR'd
	// validateBlocks(t, 1, ctx, node, l2client)
}

func TestProgramCalls(t *testing.T) {
	ctx, _, l2info, l2client, auth, callsAddr, cleanup := setupProgramTest(t, rustFile("multicall"), true)
	defer cleanup()

	ensure := func(tx *types.Transaction, err error) *types.Receipt {
		t.Helper()
		Require(t, err)
		receipt, err := EnsureTxSucceeded(ctx, l2client, tx)
		Require(t, err)
		return receipt
	}

	expectFailure := func(to common.Address, data []byte, errMsg string) {
		t.Helper()
		msg := ethereum.CallMsg{
			To:    &to,
			Value: big.NewInt(0),
			Data:  data,
		}
		_, err := l2client.CallContract(ctx, msg, nil)
		if err == nil {
			Fail(t, "call should have failed with", errMsg)
		}
		expected := fmt.Sprintf("execution reverted%v", errMsg)
		if err.Error() != expected {
			Fail(t, "wrong error", err.Error(), expected)
		}

		// execute onchain for proving's sake
		tx := l2info.PrepareTxTo("Owner", &callsAddr, 1e9, nil, data)
		Require(t, l2client.SendTransaction(ctx, tx))
		receipt, err := WaitForTx(ctx, l2client, tx.Hash(), 5*time.Second)
		Require(t, err)
		if receipt.Status != types.ReceiptStatusFailed {
			Fail(t, "unexpected success")
		}
	}

	storeAddr := deployWasm(t, ctx, auth, l2client, rustFile("storage"))
	keccakAddr := deployWasm(t, ctx, auth, l2client, rustFile("keccak"))
	mockAddr, tx, _, err := mocksgen.DeployProgramTest(&auth, l2client)
	ensure(tx, err)

	colors.PrintGrey("multicall.wasm ", callsAddr)
	colors.PrintGrey("storage.wasm   ", storeAddr)
	colors.PrintGrey("keccak.wasm    ", keccakAddr)
	colors.PrintGrey("mock.evm       ", mockAddr)

	kinds := make(map[vm.OpCode]byte)
	kinds[vm.CALL] = 0x00
	kinds[vm.DELEGATECALL] = 0x01
	kinds[vm.STATICCALL] = 0x02

	makeCalldata := func(opcode vm.OpCode, address common.Address, value *big.Int, calldata []byte) []byte {
		args := []byte{0x01}
		length := 21 + len(calldata)
		if opcode == vm.CALL {
			length += 32
		}
		args = append(args, arbmath.Uint32ToBytes(uint32(length))...)
		args = append(args, kinds[opcode])
		if opcode == vm.CALL {
			if value == nil {
				value = common.Big0
			}
			args = append(args, common.BigToHash(value).Bytes()...)
		}
		args = append(args, address.Bytes()...)
		args = append(args, calldata...)
		return args
	}
	appendCall := func(calls []byte, opcode vm.OpCode, address common.Address, inner []byte) []byte {
		calls[0] += 1 // add another call
		calls = append(calls, makeCalldata(opcode, address, nil, inner)[1:]...)
		return calls
	}

	checkTree := func(opcode vm.OpCode, dest common.Address) map[common.Hash]common.Hash {
		colors.PrintBlue("Checking storage after call tree with ", opcode)
		slots := make(map[common.Hash]common.Hash)
		zeroHashBytes := common.BigToHash(common.Big0).Bytes()

		var nest func(level uint) []uint8
		nest = func(level uint) []uint8 {
			args := []uint8{}

			if level == 0 {
				// call storage.wasm
				args = append(args, kinds[opcode])
				if opcode == vm.CALL {
					args = append(args, zeroHashBytes...)
				}
				args = append(args, storeAddr[:]...)

				key := testhelpers.RandomHash()
				value := testhelpers.RandomHash()
				slots[key] = value

				// insert value @ key
				args = append(args, 0x01)
				args = append(args, key[:]...)
				args = append(args, value[:]...)
				return args
			}

			// do the two following calls
			args = append(args, 0x00)
			args = append(args, zeroHashBytes...)
			args = append(args, callsAddr[:]...)
			args = append(args, 2)

			for i := 0; i < 2; i++ {
				inner := nest(level - 1)
				args = append(args, arbmath.Uint32ToBytes(uint32(len(inner)))...)
				args = append(args, inner...)
			}
			return args
		}
		tree := nest(3)[53:]
		tx = l2info.PrepareTxTo("Owner", &callsAddr, 1e9, nil, tree)
		ensure(tx, l2client.SendTransaction(ctx, tx))

		for key, value := range slots {
			storedBytes, err := l2client.StorageAt(ctx, dest, key, nil)
			Require(t, err)
			storedValue := common.BytesToHash(storedBytes)
			if value != storedValue {
				Fail(t, "wrong value", value, storedValue)
			}
		}

		return slots
	}

	slots := checkTree(vm.CALL, storeAddr)
	checkTree(vm.DELEGATECALL, callsAddr)

	colors.PrintBlue("Checking static call")
	calldata := []byte{0}
	expected := []byte{}
	for key, value := range slots {
		readKey := append([]byte{0x00}, key[:]...)
		calldata = appendCall(calldata, vm.STATICCALL, storeAddr, readKey)
		expected = append(expected, value[:]...)
	}
	values := sendContractCall(t, ctx, callsAddr, l2client, calldata)
	if !bytes.Equal(expected, values) {
		Fail(t, "wrong results static call", common.Bytes2Hex(expected), common.Bytes2Hex(values))
	}
	tx = l2info.PrepareTxTo("Owner", &callsAddr, 1e9, nil, calldata)
	ensure(tx, l2client.SendTransaction(ctx, tx))

	colors.PrintBlue("Checking static call write protection")
	writeKey := append([]byte{0x1}, testhelpers.RandomHash().Bytes()...)
	writeKey = append(writeKey, testhelpers.RandomHash().Bytes()...)
	expectFailure(callsAddr, makeCalldata(vm.STATICCALL, storeAddr, nil, writeKey), "")

	// mechanisms for creating calldata
	burnArbGas, _ := util.NewCallParser(precompilesgen.ArbosTestABI, "burnArbGas")
	customRevert, _ := util.NewCallParser(precompilesgen.ArbDebugABI, "customRevert")
	legacyError, _ := util.NewCallParser(precompilesgen.ArbDebugABI, "legacyError")
	callKeccak, _ := util.NewCallParser(mocksgen.ProgramTestABI, "callKeccak")
	pack := func(data []byte, err error) []byte {
		Require(t, err)
		return data
	}

	colors.PrintBlue("Calling the ArbosTest precompile (Rust => precompile)")
	testPrecompile := func(gas uint64) uint64 {
		// Call the burnArbGas() precompile from Rust
		args := makeCalldata(vm.CALL, types.ArbosTestAddress, nil, pack(burnArbGas(big.NewInt(int64(gas)))))
		tx := l2info.PrepareTxTo("Owner", &callsAddr, 1e9, nil, args)
		receipt := ensure(tx, l2client.SendTransaction(ctx, tx))
		return receipt.GasUsed - receipt.GasUsedForL1
	}

	smallGas := testhelpers.RandomUint64(2000, 8000)
	largeGas := smallGas + testhelpers.RandomUint64(2000, 8000)
	small := testPrecompile(smallGas)
	large := testPrecompile(largeGas)

	if !arbmath.Within(large-small, largeGas-smallGas, 1) {
		ratio := float64(int64(large)-int64(small)) / float64(int64(largeGas)-int64(smallGas))
		Fail(t, "inconsistent burns", large, small, largeGas, smallGas, ratio)
	}

	colors.PrintBlue("Checking consensus revert data (Rust => precompile)")
	args := makeCalldata(vm.CALL, types.ArbDebugAddress, nil, pack(customRevert(uint64(32))))
	spider := ": error Custom(32, This spider family wards off bugs: /\\oo/\\ //\\(oo)//\\ /\\oo/\\, true)"
	expectFailure(callsAddr, args, spider)

	colors.PrintBlue("Checking non-consensus revert data (Rust => precompile)")
	args = makeCalldata(vm.CALL, types.ArbDebugAddress, nil, pack(legacyError()))
	expectFailure(callsAddr, args, "")

	colors.PrintBlue("Checking success (Rust => Solidity => Rust)")
	rustArgs := append([]byte{0x01}, []byte(spider)...)
	mockArgs := makeCalldata(vm.CALL, mockAddr, nil, pack(callKeccak(keccakAddr, rustArgs)))
	tx = l2info.PrepareTxTo("Owner", &callsAddr, 1e9, nil, mockArgs)
	ensure(tx, l2client.SendTransaction(ctx, tx))

	colors.PrintBlue("Checking call with value (Rust => EOA)")
	eoa := testhelpers.RandomAddress()
	value := big.NewInt(1 + rand.Int63n(1e12))
	args = makeCalldata(vm.CALL, eoa, value, []byte{})
	tx = l2info.PrepareTxTo("Owner", &callsAddr, 1e9, value, args)
	ensure(tx, l2client.SendTransaction(ctx, tx))
	balance := GetBalance(t, ctx, l2client, eoa)
	if !arbmath.BigEquals(balance, value) {
		Fail(t, balance, value)
	}

	// TODO: enable validation when prover side is PR'd
	// validateBlocks(t, 1, ctx, node, l2client)
}

func TestProgramLogs(t *testing.T) {
	ctx, _, l2info, l2client, _, logAddr, cleanup := setupProgramTest(t, rustFile("log"), true)
	defer cleanup()

	ensure := func(tx *types.Transaction, err error) *types.Receipt {
		t.Helper()
		Require(t, err)
		receipt, err := EnsureTxSucceeded(ctx, l2client, tx)
		Require(t, err)
		return receipt
	}

	encode := func(topics []common.Hash, data []byte) []byte {
		args := []byte{byte(len(topics))}
		for _, topic := range topics {
			args = append(args, topic[:]...)
		}
		args = append(args, data...)
		return args
	}
	randBytes := func(min, max uint64) []byte {
		return testhelpers.RandomSlice(testhelpers.RandomUint64(min, max))
	}

	for i := 0; i <= 4; i++ {
		colors.PrintGrey("Emitting ", i, " topics")
		topics := make([]common.Hash, i)
		for j := 0; j < i; j++ {
			topics[j] = testhelpers.RandomHash()
		}
		data := randBytes(0, 48)
		args := encode(topics, data)
		tx := l2info.PrepareTxTo("Owner", &logAddr, 1e9, nil, args)
		receipt := ensure(tx, l2client.SendTransaction(ctx, tx))

		if len(receipt.Logs) != 1 {
			Fail(t, "wrong number of logs", len(receipt.Logs))
		}
		log := receipt.Logs[0]
		if !bytes.Equal(log.Data, data) {
			Fail(t, "data mismatch", log.Data, data)
		}
		if len(log.Topics) != len(topics) {
			Fail(t, "topics mismatch", len(log.Topics), len(topics))
		}
		for j := 0; j < i; j++ {
			if log.Topics[j] != topics[j] {
				Fail(t, "topic mismatch", log.Topics, topics)
			}
		}
	}

	tooMany := encode([]common.Hash{{}, {}, {}, {}, {}}, []byte{})
	tx := l2info.PrepareTxTo("Owner", &logAddr, l2info.TransferGas, nil, tooMany)
	Require(t, l2client.SendTransaction(ctx, tx))
	receipt, err := WaitForTx(ctx, l2client, tx.Hash(), 5*time.Second)
	Require(t, err)
	if receipt.Status != types.ReceiptStatusFailed {
		Fail(t, "call should have failed")
	}

	// TODO: enable validation when prover side is PR'd
	// validateBlocks(t, 1, ctx, node, l2client)
}

func setupProgramTest(t *testing.T, file string, jit bool) (
	context.Context, *arbnode.Node, *BlockchainTestInfo, *ethclient.Client, bind.TransactOpts, common.Address, func(),
) {
	ctx, cancel := context.WithCancel(context.Background())
	rand.Seed(time.Now().UTC().UnixNano())

	chainConfig := params.ArbitrumDevTestChainConfig()
	l2config := arbnode.ConfigDefaultL1Test()
	l2config.BlockValidator.Enable = true
	l2config.BatchPoster.Enable = true
	l2config.L1Reader.Enable = true
	l2config.Sequencer.MaxRevertGasReject = 0
	AddDefaultValNode(t, ctx, l2config, jit)

	l2info, node, l2client, _, _, _, l1stack := createTestNodeOnL1WithConfig(t, ctx, true, l2config, chainConfig, nil)

	cleanup := func() {
		requireClose(t, l1stack)
		node.StopAndWait()
		cancel()
	}

	auth := l2info.GetDefaultTransactOpts("Owner", ctx)

	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, l2client)
	Require(t, err)
	arbDebug, err := precompilesgen.NewArbDebug(types.ArbDebugAddress, l2client)
	Require(t, err)

	ensure := func(tx *types.Transaction, err error) *types.Receipt {
		t.Helper()
		Require(t, err)
		receipt, err := EnsureTxSucceeded(ctx, l2client, tx)
		Require(t, err)
		return receipt
	}

	// Set random pricing params. Note that the WASM gas price is measured in bips,
	// so a gas price of 10k means that 1 evm gas buys exactly 1 wasm gas.
	// We choose a range on both sides of this value.
	wasmGasPrice := testhelpers.RandomUint64(0, 20000)  // evm to wasm gas
	wasmHostioCost := testhelpers.RandomUint64(0, 5000) // amount of wasm gas
	colors.PrintMint(fmt.Sprintf("WASM gas price=%d, HostIO cost=%d", wasmGasPrice, wasmHostioCost))

	ensure(arbDebug.BecomeChainOwner(&auth))
	ensure(arbOwner.SetWasmGasPrice(&auth, wasmGasPrice))
	ensure(arbOwner.SetWasmHostioCost(&auth, wasmHostioCost))

	programAddress := deployWasm(t, ctx, auth, l2client, file)

	return ctx, node, l2info, l2client, auth, programAddress, cleanup
}

func deployWasm(
	t *testing.T, ctx context.Context, auth bind.TransactOpts, l2client *ethclient.Client, file string,
) common.Address {
	wasmSource, err := os.ReadFile(file)
	Require(t, err)
	wasm, err := arbcompress.CompressWell(wasmSource)
	Require(t, err)

	toKb := func(data []byte) float64 { return float64(len(data)) / 1024.0 }
	colors.PrintMint(fmt.Sprintf("WASM len %.2fK vs %.2fK", toKb(wasm), toKb(wasmSource)))

	wasm = append(state.StylusPrefix, wasm...)

	programAddress := deployContract(t, ctx, auth, l2client, wasm)
	colors.PrintBlue("program deployed to ", programAddress.Hex())

	arbWasm, err := precompilesgen.NewArbWasm(types.ArbWasmAddress, l2client)
	Require(t, err)

	timed(t, "compile", func() {
		tx, err := arbWasm.CompileProgram(&auth, programAddress)
		Require(t, err)
		_, err = EnsureTxSucceeded(ctx, l2client, tx)
		Require(t, err)
	})

	return programAddress
}

func rustFile(name string) string {
	return fmt.Sprintf("../arbitrator/stylus/tests/%v/target/wasm32-unknown-unknown/release/%v.wasm", name, name)
}

func validateBlocks(t *testing.T, start uint64, ctx context.Context, node *arbnode.Node, l2client *ethclient.Client) {
	colors.PrintGrey("Validating blocks from ", start, " onward")

	doUntil(t, 20*time.Millisecond, 50, func() bool {
		batchCount, err := node.InboxTracker.GetBatchCount()
		Require(t, err)
		meta, err := node.InboxTracker.GetBatchMetadata(batchCount - 1)
		Require(t, err)
		messageCount, err := node.ArbInterface.TransactionStreamer().GetMessageCount()
		Require(t, err)
		return meta.MessageCount == messageCount
	})

	blockHeight, err := l2client.BlockNumber(ctx)
	Require(t, err)

	success := true
	for block := start; block <= blockHeight; block++ {
		header, err := l2client.HeaderByNumber(ctx, arbmath.UintToBig(block))
		Require(t, err)

		now := time.Now()
		correct, err := node.StatelessBlockValidator.ValidateBlock(ctx, header, false, common.Hash{})
		Require(t, err, "block", block)
		passed := formatTime(time.Since(now))
		if correct {
			colors.PrintMint("yay!! we validated block ", block, " in ", passed)
		} else {
			colors.PrintRed("failed to validate block ", block, " in ", passed)
		}
		success = success && correct
	}
	if !success {
		Fail(t)
	}
}

func timed(t *testing.T, message string, lambda func()) {
	t.Helper()
	now := time.Now()
	lambda()
	passed := time.Since(now)
	colors.PrintBlue("Time to ", message, ": ", passed.String())
}

func formatTime(duration time.Duration) string {
	span := float64(duration.Nanoseconds())
	unit := 0
	units := []string{"ns", "μs", "ms", "s", "min", "h", "d", "w", "mo", "yr", "dec", "cent", "mill", "eon"}
	scale := []float64{1000., 1000., 1000., 60., 60., 24., 7., 4.34, 12., 10., 10., 10., 1000000.}
	for span >= scale[unit] && unit < len(scale) {
		span /= scale[unit]
		unit += 1
	}
	return fmt.Sprintf("%.2f%s", span, units[unit])
}
