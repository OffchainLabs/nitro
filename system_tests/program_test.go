// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package arbtest

import (
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
	tx := l2info.PrepareTxTo("Owner", &programAddress, l2info.TransferGas, big.NewInt(0), []byte{0x01})
	Require(t, l2client.SendTransaction(ctx, tx))
	_, err := EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)

	// ensure tx fails
	tx = l2info.PrepareTxTo("Owner", &programAddress, l2info.TransferGas, big.NewInt(0), []byte{0x00})
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

	tx := l2info.PrepareTxTo("Owner", &programAddress, l2info.TransferGas, big.NewInt(0), storeArgs)
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
	ctx, _, l2info, l2client, auth, callsAddr, cleanup := setupProgramTest(t, rustFile("calls"), true)
	defer cleanup()

	storeAddr := deployWasm(t, ctx, auth, l2client, rustFile("storage"))

	colors.PrintGrey("calls.wasm   ", callsAddr)
	colors.PrintGrey("storage.wasm ", storeAddr)

	ensure := func(tx *types.Transaction, err error) *types.Receipt {
		t.Helper()
		Require(t, err)
		receipt, err := EnsureTxSucceeded(ctx, l2client, tx)
		Require(t, err)
		return receipt
	}

	slots := make(map[common.Hash]common.Hash)

	var nest func(level uint) []uint8
	nest = func(level uint) []uint8 {
		args := []uint8{}

		if level == 0 {
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
		args = append(args, callsAddr[:]...)
		args = append(args, 2)

		for i := 0; i < 2; i++ {
			inner := nest(level - 1)
			args = append(args, arbmath.Uint32ToBytes(uint32(len(inner)))...)
			args = append(args, inner...)
		}
		return args
	}
	tree := nest(3)[20:]
	colors.PrintGrey(common.Bytes2Hex(tree))

	tx := l2info.PrepareTxTo("Owner", &callsAddr, 1e9, big.NewInt(0), tree)
	ensure(tx, l2client.SendTransaction(ctx, tx))

	for key, value := range slots {
		storedBytes, err := l2client.StorageAt(ctx, storeAddr, key, nil)
		Require(t, err)
		storedValue := common.BytesToHash(storedBytes)
		if value != storedValue {
			Fail(t, "wrong value", value, storedValue)
		}
	}

	// mechanisms for creating calldata
	burnArbGas, _ := util.NewCallParser(precompilesgen.ArbosTestABI, "burnArbGas")
	customRevert, _ := util.NewCallParser(precompilesgen.ArbDebugABI, "customRevert")
	legacyError, _ := util.NewCallParser(precompilesgen.ArbDebugABI, "legacyError")
	pack := func(data []byte, err error) []byte {
		Require(t, err)
		return data
	}
	makeCalldata := func(address common.Address, calldata []byte) []byte {
		args := []byte{0x01}
		args = append(args, arbmath.Uint32ToBytes(uint32(20+len(calldata)))...)
		args = append(args, address.Bytes()...)
		args = append(args, calldata...)
		return args
	}

	// Set a random, non-zero gas price
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, l2client)
	Require(t, err)
	wasmGasPrice := testhelpers.RandomUint64(1, 2000)
	ensure(arbOwner.SetWasmGasPrice(&auth, wasmGasPrice))
	colors.PrintBlue("Calling the ArbosTest precompile with wasmGasPrice=", wasmGasPrice)

	testPrecompile := func(gas uint64) uint64 {
		// Call the burnArbGas() precompile from Rust
		args := makeCalldata(types.ArbosTestAddress, pack(burnArbGas(big.NewInt(int64(gas)))))
		tx := l2info.PrepareTxTo("Owner", &callsAddr, 1e9, big.NewInt(0), args)
		return ensure(tx, l2client.SendTransaction(ctx, tx)).GasUsed
	}

	smallGas := testhelpers.RandomUint64(2000, 8000)
	largeGas := smallGas + testhelpers.RandomUint64(2000, 8000)
	small := testPrecompile(smallGas)
	large := testPrecompile(largeGas)

	if large-small != largeGas-smallGas {
		ratio := float64(large-small) / float64(largeGas-smallGas)
		Fail(t, "inconsistent burns", smallGas, largeGas, small, large, ratio)
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
	}

	colors.PrintBlue("Check consensus revert data")
	args := makeCalldata(types.ArbDebugAddress, pack(customRevert(uint64(32))))
	spider := ": error Custom(32, This spider family wards off bugs: /\\oo/\\ //\\(oo)//\\ /\\oo/\\, true)"
	expectFailure(callsAddr, args, spider)

	colors.PrintBlue("Check non-consensus revert data")
	args = makeCalldata(types.ArbDebugAddress, pack(legacyError()))
	expectFailure(callsAddr, args, "")

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

	// Drop the gas price to 0 half the time
	if testhelpers.RandomBool() {
		wasmGasPrice = 0
	}
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
