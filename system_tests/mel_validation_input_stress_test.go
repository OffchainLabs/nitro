// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// Data stress test for MEL WASM validation preimage loading.
//
// Goal: produce a MASSIVE L1 block with MASSIVE receipts, record the receipt
// and transaction trie preimages, serialize them into a valid InputJSON, and
// see if the arbitrator running the unified replay binary can even load that
// many preimages.
//
// We do NOT run MEL validation here — there are no L2 messages to extract.
// We only stress-test the preimage path: serialization, deserialization, and
// machine preimage-table construction.
//
// Run with:
//   go test -tags mel_validation_input_stress -run TestMELValidationInputStress -v -timeout 30m ./system_tests/
//
// Then feed the generated file to benchbin:
//   ./target/release/benchbin \
//     --json-inputs ~/.arbitrum/validation-inputs/mel-stress-test/<timestamp>/block_inputs_1.json \
//     --binary target/machines/latest/machine.v2.wavm.br

//go:build mel_validation_input_stress

package arbtest

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"

	melrecording "github.com/offchainlabs/nitro/arbnode/mel/recording"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/util/jsonapi"
	"github.com/offchainlabs/nitro/validator"
	"github.com/offchainlabs/nitro/validator/inputs"
	"github.com/offchainlabs/nitro/validator/server_api"
)

// buildLogCannonInitCode builds EVM init code that deploys a contract whose
// runtime code emits `numLogs` LOG4 events per call, each with 4 topics and
// `bytesPerLog` bytes of data.
//
// Runtime code size limit (EIP-170) is 24KB. With MSTORE-based memory setup
// (39 bytes of bytecode per 32-byte word) and topic pushes, the practical
// ceiling is around 30 LOG4 events with 512 bytes each.
func buildLogCannonInitCode(numLogs, bytesPerLog int) []byte {
	if bytesPerLog%32 != 0 {
		panic("bytesPerLog must be a multiple of 32")
	}
	wordsPerLog := bytesPerLog / 32

	var runtime []byte
	for i := 0; i < numLogs; i++ {
		// Fill memory[0:bytesPerLog] with unique data for this log.
		for j := 0; j < wordsPerLog; j++ {
			var word [32]byte
			word[0] = byte(i)
			word[1] = byte(j)
			for k := 2; k < 32; k++ {
				word[k] = 0xAB
			}
			// PUSH32 <word> PUSH2 <offset> MSTORE
			runtime = append(runtime, byte(vm.PUSH32))
			runtime = append(runtime, word[:]...)
			offset := j * 32
			runtime = append(runtime, byte(vm.PUSH2), byte(offset>>8), byte(offset&0xFF))
			runtime = append(runtime, byte(vm.MSTORE))
		}
		// Push topics in reverse stack order (topic3 pushed first, popped last).
		uniqueTopic := crypto.Keccak256Hash([]byte(fmt.Sprintf("stress-log-%d", i)))
		// topic3
		runtime = append(runtime, byte(vm.PUSH32))
		runtime = append(runtime, uniqueTopic[:]...)
		// topic2
		runtime = append(runtime, byte(vm.PUSH32))
		runtime = append(runtime, uniqueTopic[:]...)
		// topic1 (index-based)
		var t1 [32]byte
		t1[31] = byte(i & 0xFF)
		t1[30] = byte((i >> 8) & 0xFF)
		runtime = append(runtime, byte(vm.PUSH32))
		runtime = append(runtime, t1[:]...)
		// topic0 (fixed event signature)
		eventSig := crypto.Keccak256Hash([]byte("StressEvent(uint256,bytes32,bytes32,bytes32)"))
		runtime = append(runtime, byte(vm.PUSH32))
		runtime = append(runtime, eventSig[:]...)
		// LOG4(offset=0, size=bytesPerLog, topic0..topic3)
		runtime = append(runtime, byte(vm.PUSH2), byte(bytesPerLog>>8), byte(bytesPerLog&0xFF))
		runtime = append(runtime, byte(vm.PUSH1), 0x00)
		runtime = append(runtime, byte(vm.LOG4))
	}
	runtime = append(runtime, byte(vm.STOP))
	return wrapAsInitCode(runtime)
}

// wrapAsInitCode wraps runtime bytecode in init code that deploys it.
// Init prefix: PUSH2 runtimeLen, PUSH2 initPrefixLen, PUSH1 0, CODECOPY,
//
//	PUSH2 runtimeLen, PUSH1 0, RETURN — 15 bytes total.
func wrapAsInitCode(runtime []byte) []byte {
	runtimeLen := len(runtime)
	const initPrefixLen = 15
	var initCode []byte
	initCode = append(initCode, byte(vm.PUSH2), byte(runtimeLen>>8), byte(runtimeLen&0xFF))
	initCode = append(initCode, byte(vm.PUSH2), byte(initPrefixLen>>8), byte(initPrefixLen&0xFF))
	initCode = append(initCode, byte(vm.PUSH1), 0x00)
	initCode = append(initCode, byte(vm.CODECOPY))
	initCode = append(initCode, byte(vm.PUSH2), byte(runtimeLen>>8), byte(runtimeLen&0xFF))
	initCode = append(initCode, byte(vm.PUSH1), 0x00)
	initCode = append(initCode, byte(vm.RETURN))
	initCode = append(initCode, runtime...)
	return initCode
}

// buildGiantLogInitCode builds EVM init code that deploys a contract whose
// runtime code emits ONE LOG4 event with `bytesPerLog` bytes of (zero-filled)
// log data per call. No MSTOREs — LOG4 lazily expands memory and the log data
// is whatever's in memory at the time (zeros).
//
// For receipt stress testing, zero-filled log data is fine: the receipt still
// stores bytesPerLog bytes.
//
// `topic1` is set to BLOCK.NUMBER so each block's receipt differs, preventing
// preimage deduplication across blocks.
func buildGiantLogInitCode(bytesPerLog int) []byte {
	var runtime []byte
	topic3 := crypto.Keccak256Hash([]byte("giant-topic-3"))
	topic2 := crypto.Keccak256Hash([]byte("giant-topic-2"))
	topic0 := crypto.Keccak256Hash([]byte("GiantStressEvent(uint256,bytes32,bytes32,bytes)"))
	// Push topics in reverse stack order (topic3 pushed first, popped last).
	runtime = append(runtime, byte(vm.PUSH32))
	runtime = append(runtime, topic3[:]...)
	runtime = append(runtime, byte(vm.PUSH32))
	runtime = append(runtime, topic2[:]...)
	// topic1 = BLOCK.NUMBER (free per-block variation, prevents dedup)
	runtime = append(runtime, byte(vm.NUMBER))
	// topic0 = event signature
	runtime = append(runtime, byte(vm.PUSH32))
	runtime = append(runtime, topic0[:]...)
	// size (PUSH3 supports up to 16 MB)
	runtime = append(runtime, byte(vm.PUSH3))
	runtime = append(runtime, byte(bytesPerLog>>16), byte((bytesPerLog>>8)&0xFF), byte(bytesPerLog&0xFF))
	// offset = 0
	runtime = append(runtime, byte(vm.PUSH1), 0x00)
	// LOG4(offset, size, topic0, topic1, topic2, topic3)
	runtime = append(runtime, byte(vm.LOG4))
	runtime = append(runtime, byte(vm.STOP))
	return wrapAsInitCode(runtime)
}

// writeValidationInputJSON writes a synthetic InputJSON to disk that contains
// the given preimages and benign empty values for everything else.
//
// CRITICAL: Rust's ValidationRequest deserializer rejects `null` for BatchInfo
// and UserWasms (they're typed as Vec<> / HashMap<>). Go's `json.Marshal`
// serializes nil slices/maps as `null`. We explicitly initialize them to
// empty so the JSON round-trips to Rust.
func writeValidationInputJSON(
	t *testing.T,
	preimages daprovider.PreimagesMap,
	slug string,
) *server_api.InputJSON {
	t.Helper()
	jsonPreimagesMap := make(map[arbutil.PreimageType]*jsonapi.PreimagesMapJson)
	for ty, innerMap := range preimages {
		jsonPreimagesMap[ty] = jsonapi.NewPreimagesMapJson(innerMap)
	}
	input := &server_api.InputJSON{
		Id:           1,
		PreimagesB64: jsonPreimagesMap,
		StartState:   validator.GoGlobalState{},
		// These must be non-nil; Rust deserializer rejects `null` here.
		BatchInfo: []server_api.BatchInfoJson{},
		UserWasms: make(map[rawdb.WasmTarget]map[common.Hash]string),
	}
	writer, err := inputs.NewWriter(
		inputs.WithSlug(slug),
		inputs.WithTimestampDirEnabled(true),
	)
	Require(t, err)
	Require(t, writer.Write(input))
	return input
}

// TestMELValidationInputStress creates a massive L1 block with massive receipts
// and serializes the recorded preimages as a validation-input JSON. The goal
// is to check whether the arbitrator replay binary can load such a heavy
// preimage table (no L2 messages are extracted).
//
// Parameters (tuned to stay within the geth dev L1 gas limit of 15M):
//   - 30 LOG4 events per call (each 512 bytes of data + 4 topics = ~5.9K gas/log)
//   - Up to 75 calls per L1 block (~200K gas each)
//   - 400 total calls spread across ~5-6 L1 blocks
//   - Total: ~12K receipts, ~12K logs, several MB of log data
func TestMELValidationInputStress(t *testing.T) {
	const (
		logsPerCall  = 30
		bytesPerLog  = 512
		gasPerCall   = 300_000 // headroom for memory + setup overhead
		totalL1Calls = 400
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	cleanup := builder.Build(t)
	defer cleanup()

	l1Client := builder.L1.Client
	l1Info := builder.L1Info

	// --- Phase 1: Deploy log-cannon contract on L1 ---
	t.Logf("Phase 1: Deploying log-cannon (%d LOG4 events per call, %d bytes each)...",
		logsPerCall, bytesPerLog)
	initCode := buildLogCannonInitCode(logsPerCall, bytesPerLog)
	t.Logf("  Init code size: %d bytes", len(initCode))
	deployTx := l1Info.PrepareTxTo("Faucet", nil, 10_000_000, common.Big0, initCode)
	Require(t, l1Client.SendTransaction(ctx, deployTx))
	deployReceipt, err := EnsureTxSucceeded(ctx, l1Client, deployTx)
	Require(t, err)
	contractAddr := deployReceipt.ContractAddress
	t.Logf("  Contract deployed at %s (gas used: %d, runtime size: %d bytes)",
		contractAddr.Hex(), deployReceipt.GasUsed, len(initCode)-15)

	// --- Phase 2: Spam L1 with calls to the log cannon ---
	t.Logf("Phase 2: Sending %d L1 transactions to the log cannon...", totalL1Calls)
	l1Txs := make([]*types.Transaction, totalL1Calls)
	for i := 0; i < totalL1Calls; i++ {
		tx := l1Info.PrepareTxTo("Faucet", &contractAddr, gasPerCall, common.Big0, nil)
		Require(t, l1Client.SendTransaction(ctx, tx))
		l1Txs[i] = tx
	}
	t.Log("  All txs submitted, waiting for mining...")
	firstReceipt, err := EnsureTxSucceeded(ctx, l1Client, l1Txs[0])
	Require(t, err)
	var lastReceipt *types.Receipt
	for i, tx := range l1Txs {
		receipt, err := EnsureTxSucceeded(ctx, l1Client, tx)
		Require(t, err)
		if i == totalL1Calls-1 {
			lastReceipt = receipt
		}
	}
	startBlock := firstReceipt.BlockNumber.Uint64()
	endBlock := lastReceipt.BlockNumber.Uint64()
	t.Logf("  All txs mined across L1 blocks %d-%d (%d blocks)",
		startBlock, endBlock, endBlock-startBlock+1)

	// --- Phase 3: Record preimages for every L1 block in range ---
	// This mirrors what MELValidator.CreateNextValidationEntry does per L1 block.
	t.Log("Phase 3: Recording receipt + transaction trie preimages...")
	preimages := make(daprovider.PreimagesMap)
	preimages[arbutil.Keccak256PreimageType] = make(map[common.Hash][]byte)

	totalReceipts := 0
	totalLogs := 0
	totalLogDataBytes := 0
	for blockNum := startBlock; blockNum <= endBlock; blockNum++ {
		block, err := l1Client.BlockByNumber(ctx, new(big.Int).SetUint64(blockNum))
		Require(t, err)
		txCount := len(block.Transactions())
		gasPct := float64(block.GasUsed()) / float64(block.GasLimit()) * 100
		t.Logf("  Block %d: %d txs, gas %d / %d (%.1f%%)",
			blockNum, txCount, block.GasUsed(), block.GasLimit(), gasPct)

		logsFetcher, err := melrecording.RecordReceipts(ctx, l1Client, block.Hash(), preimages)
		Require(t, err)
		logs, err := logsFetcher.LogsForBlockHash(ctx, block.Hash())
		Require(t, err)
		totalReceipts += txCount
		totalLogs += len(logs)
		for _, lg := range logs {
			totalLogDataBytes += len(lg.Data) + 32*len(lg.Topics)
		}

		txRecorder, err := melrecording.NewTransactionRecorder(l1Client, block.Hash(), preimages)
		Require(t, err)
		Require(t, txRecorder.Initialize(ctx))
	}

	// Also record the L1 headers themselves (MEL validator stores these as preimages).
	for blockNum := startBlock; blockNum <= endBlock; blockNum++ {
		block, err := l1Client.BlockByNumber(ctx, new(big.Int).SetUint64(blockNum))
		Require(t, err)
		header := block.Header()
		headerRLP, err := rlp.EncodeToBytes(header)
		Require(t, err)
		preimages[arbutil.Keccak256PreimageType][header.Hash()] = headerRLP
	}

	// --- Phase 4: Report sizes ---
	totalPreimageCount := 0
	totalPreimageBytes := 0
	for preimageType, innerMap := range preimages {
		count := len(innerMap)
		var byteCount int
		for _, v := range innerMap {
			byteCount += len(v)
		}
		t.Logf("  PreimageType %d: count=%d, totalBytes=%d (%.2f MB)",
			preimageType, count, byteCount, float64(byteCount)/(1024*1024))
		totalPreimageCount += count
		totalPreimageBytes += byteCount
	}

	// --- Phase 5: Serialize to JSON and write to disk ---
	t.Log("Phase 5: Serializing to InputJSON...")
	startMarshal := time.Now()
	input := writeValidationInputJSON(t, preimages, "mel-stress-test")
	marshalDuration := time.Since(startMarshal)

	jsonBytes, err := input.Marshal()
	Require(t, err)

	t.Log("")
	t.Log("=== SUMMARY ===")
	t.Logf("  L1 blocks:              %d", endBlock-startBlock+1)
	t.Logf("  Receipts:               %d", totalReceipts)
	t.Logf("  Logs:                   %d", totalLogs)
	t.Logf("  Log data (incl topics): %d bytes (%.2f MB)",
		totalLogDataBytes, float64(totalLogDataBytes)/(1024*1024))
	t.Logf("  Total preimages:        %d", totalPreimageCount)
	t.Logf("  Preimage raw bytes:     %d (%.2f MB)",
		totalPreimageBytes, float64(totalPreimageBytes)/(1024*1024))
	t.Logf("  InputJSON size:         %d bytes (%.2f MB)",
		len(jsonBytes), float64(len(jsonBytes))/(1024*1024))
	t.Logf("  JSON write duration:    %v", marshalDuration)
	if totalPreimageBytes > 0 {
		t.Logf("  JSON / raw overhead:    %.2fx",
			float64(len(jsonBytes))/float64(totalPreimageBytes))
	}
	t.Log("")
	t.Log("JSON written to ~/.arbitrum/validation-inputs/mel-stress-test/<timestamp>/block_inputs_1.json")
	t.Log("Run benchbin to test arbitrator preimage loading:")
	t.Log("  ./target/release/benchbin \\")
	t.Log("    --json-inputs <above path> \\")
	t.Log("    --binary target/machines/latest/machine.v2.wavm.br")
}

// TestMELValidationInputStressMaxReceipt creates the MAXIMUM possible receipt
// size per L1 block — one tx consuming ~14.9M gas and emitting one LOG4 event
// with ~1.3 MB of data. Each L1 block contains exactly one such tx.
//
// Goal: push arbitrator preimage loading to its limit with giant individual
// leaf preimages (~1.3 MB each) rather than many small ones.
//
// Expected per-block output:
//   - 1 receipt, ~1.3 MB raw (one LOG4 with ~1.3 MB data + 4 topics)
//   - Receipts trie has 1 leaf (= root)
//   - Tx trie has 1 leaf (= root)
//
// Across `numL1Blocks` blocks: ~numL1Blocks × 1.3 MB total preimage data.
func TestMELValidationInputStressMaxReceipt(t *testing.T) {
	const (
		// 1.3 MB of log data per tx. See plan for gas budget math.
		bytesPerLog = 1_363_148
		gasPerTx    = 14_900_000
		// 17 blocks → ~52 MB JSON. 175 blocks → ~500 MB JSON (each block
		// adds ~3 MB of preimages base64-encoded).
		numL1Blocks = 175
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	cleanup := builder.Build(t)
	defer cleanup()

	l1Client := builder.L1.Client
	l1Info := builder.L1Info

	// Lower the L1 gas price so 14.9M-gas txs stay under geth's default 1 ETH
	// RPC fee cap (at 100 GWei the tx would cost 1.49 ETH → rejected).
	// With 60 GWei and 14.9M gas, fee = 0.894 ETH ≤ 1 ETH cap.
	// Basefee is 50 GWei so miner still gets 10 GWei tip.
	originalGasPrice := new(big.Int).Set(l1Info.GasPrice)
	l1Info.GasPrice = big.NewInt(60 * 1_000_000_000) // 60 GWei
	defer func() { l1Info.GasPrice = originalGasPrice }()

	// --- Phase 1: Deploy giant-log contract on L1 ---
	t.Logf("Phase 1: Deploying giant-log contract (one LOG4 with %d bytes per call)...", bytesPerLog)
	initCode := buildGiantLogInitCode(bytesPerLog)
	t.Logf("  Init code size: %d bytes (runtime: %d bytes)", len(initCode), len(initCode)-15)
	deployTx := l1Info.PrepareTxTo("Faucet", nil, 1_000_000, common.Big0, initCode)
	Require(t, l1Client.SendTransaction(ctx, deployTx))
	deployReceipt, err := EnsureTxSucceeded(ctx, l1Client, deployTx)
	Require(t, err)
	contractAddr := deployReceipt.ContractAddress
	t.Logf("  Contract deployed at %s (gas used: %d)", contractAddr.Hex(), deployReceipt.GasUsed)

	// --- Phase 2: Send giant-log txs one-by-one (each fills its own block) ---
	// We wait for each tx to be mined before sending the next, so each one
	// ends up alone in its block. If we flooded the pool, the miner might
	// try to pack multiple together and blow past the gas limit.
	t.Logf("Phase 2: Sending %d giant-log txs (each fills its own L1 block)...", numL1Blocks)
	txs := make([]*types.Transaction, numL1Blocks)
	for i := 0; i < numL1Blocks; i++ {
		tx := l1Info.PrepareTxTo("Faucet", &contractAddr, gasPerTx, common.Big0, nil)
		Require(t, l1Client.SendTransaction(ctx, tx))
		_, err := EnsureTxSucceeded(ctx, l1Client, tx)
		Require(t, err)
		txs[i] = tx
	}

	firstReceipt, err := EnsureTxSucceeded(ctx, l1Client, txs[0])
	Require(t, err)
	lastReceipt, err := EnsureTxSucceeded(ctx, l1Client, txs[numL1Blocks-1])
	Require(t, err)
	startBlock := firstReceipt.BlockNumber.Uint64()
	endBlock := lastReceipt.BlockNumber.Uint64()
	t.Logf("  Giant txs in L1 blocks %d-%d (%d blocks total)",
		startBlock, endBlock, endBlock-startBlock+1)

	// --- Phase 3: Record preimages ---
	t.Log("Phase 3: Recording receipt + transaction trie preimages...")
	preimages := make(daprovider.PreimagesMap)
	preimages[arbutil.Keccak256PreimageType] = make(map[common.Hash][]byte)

	totalReceipts := 0
	totalLogs := 0
	totalLogDataBytes := 0
	for blockNum := startBlock; blockNum <= endBlock; blockNum++ {
		block, err := l1Client.BlockByNumber(ctx, new(big.Int).SetUint64(blockNum))
		Require(t, err)
		txCount := len(block.Transactions())
		gasPct := float64(block.GasUsed()) / float64(block.GasLimit()) * 100
		t.Logf("  Block %d: %d txs, gas %d / %d (%.1f%%)",
			blockNum, txCount, block.GasUsed(), block.GasLimit(), gasPct)

		logsFetcher, err := melrecording.RecordReceipts(ctx, l1Client, block.Hash(), preimages)
		Require(t, err)
		logs, err := logsFetcher.LogsForBlockHash(ctx, block.Hash())
		Require(t, err)
		totalReceipts += txCount
		totalLogs += len(logs)
		for _, lg := range logs {
			totalLogDataBytes += len(lg.Data) + 32*len(lg.Topics)
		}

		txRecorder, err := melrecording.NewTransactionRecorder(l1Client, block.Hash(), preimages)
		Require(t, err)
		Require(t, txRecorder.Initialize(ctx))
	}

	// Record block header preimages too.
	for blockNum := startBlock; blockNum <= endBlock; blockNum++ {
		block, err := l1Client.BlockByNumber(ctx, new(big.Int).SetUint64(blockNum))
		Require(t, err)
		header := block.Header()
		headerRLP, err := rlp.EncodeToBytes(header)
		Require(t, err)
		preimages[arbutil.Keccak256PreimageType][header.Hash()] = headerRLP
	}

	// --- Phase 4: Report ---
	totalPreimageCount := 0
	totalPreimageBytes := 0
	largestPreimage := 0
	for preimageType, innerMap := range preimages {
		count := len(innerMap)
		var byteCount int
		for _, v := range innerMap {
			byteCount += len(v)
			if len(v) > largestPreimage {
				largestPreimage = len(v)
			}
		}
		t.Logf("  PreimageType %d: count=%d, totalBytes=%d (%.2f MB)",
			preimageType, count, byteCount, float64(byteCount)/(1024*1024))
		totalPreimageCount += count
		totalPreimageBytes += byteCount
	}

	// --- Phase 5: Serialize to JSON ---
	t.Log("Phase 5: Serializing to InputJSON (this may take a while with large preimages)...")
	startMarshal := time.Now()
	input := writeValidationInputJSON(t, preimages, "mel-stress-test-max")
	marshalDuration := time.Since(startMarshal)

	jsonBytes, err := input.Marshal()
	Require(t, err)

	t.Log("")
	t.Log("=== MAX-RECEIPT STRESS TEST SUMMARY ===")
	t.Logf("  L1 blocks:                 %d", endBlock-startBlock+1)
	t.Logf("  Receipts:                  %d", totalReceipts)
	t.Logf("  Logs:                      %d", totalLogs)
	t.Logf("  Log data (incl topics):    %d bytes (%.2f MB)",
		totalLogDataBytes, float64(totalLogDataBytes)/(1024*1024))
	t.Logf("  Total preimages:           %d", totalPreimageCount)
	t.Logf("  Preimage raw bytes:        %d (%.2f MB)",
		totalPreimageBytes, float64(totalPreimageBytes)/(1024*1024))
	t.Logf("  Largest single preimage:   %d bytes (%.2f MB)",
		largestPreimage, float64(largestPreimage)/(1024*1024))
	t.Logf("  InputJSON size:            %d bytes (%.2f MB)",
		len(jsonBytes), float64(len(jsonBytes))/(1024*1024))
	t.Logf("  JSON write+marshal time:   %v", marshalDuration)
	if totalPreimageBytes > 0 {
		t.Logf("  JSON / raw overhead:       %.2fx",
			float64(len(jsonBytes))/float64(totalPreimageBytes))
	}
	t.Log("")
	t.Log("JSON written to ~/.arbitrum/validation-inputs/mel-stress-test-max/<timestamp>/block_inputs_1.json")
	t.Log("Run benchbin to test arbitrator preimage loading:")
	t.Log("  ./target/release/benchbin \\")
	t.Log("    --json-inputs <above path> \\")
	t.Log("    --binary target/machines/latest/machine.v2.wavm.br")
}

// TestMELExtractionStepStress generates a validation input that maximizes the
// number of WAVM steps the arbitrator must execute during MEL extraction.
//
// Unlike the data stress tests (Part 1), this test produces a *valid* MEL
// extraction validation entry — the arbitrator's unified replay binary will
// take the `extractMessagesUpTo` path and run `melextraction.ExtractMessages`
// for every L1 block in the range.
//
// Step-consuming work:
//  1. Many delayed messages enqueued on L1 (each = hash chain accumulation step)
//  2. Many L2 transactions in a sequencer batch (decompression + RLP iteration)
//  3. The batch reads many delayed messages (each read advances accumulator)
//  4. Many L1 blocks in the extraction range (loop iterations of ExtractMessages)
//
// The InputJSON written to disk can be fed to benchbin to measure WAVM step
// throughput across step sizes.
func TestMELExtractionStepStress(t *testing.T) {
	runMELExtractionStress(t, 200, 500, "mel-step-stress", "STEP-STRESS")
}

// TestMELExtractionMaxMsgsPerBatch packs many minimal L2 transfer txs across
// multiple sequencer batches. This is the worst case for the arbitrator's
// per-batch decompression + RLP iteration cost.
//
// Note: at 20K txs the BatchPoster splits across many batches and the
// resulting validation input runs the unified replay binary into a
// "missing preimage 0x00..." error during MEL extraction (likely a corner
// case in MEL recording when very many batches span the recording window).
// At 1500 minimal txs the BatchPoster fits everything in a single batch
// (compressed payload < 99 KB threshold). We include 5 delayed messages
// because the validation input fails with "missing preimage 0x00..." when
// the recording window contains zero delayed-message activity (this corner
// case in MEL recording is worth investigating separately).
func TestMELExtractionMaxMsgsPerBatch(t *testing.T) {
	runMELExtractionStress(t, 5, 1_500, "mel-step-stress-max-msgs", "MAX-MSGS-PER-BATCH")
}

// TestMELExtractionMaxDelayedPerBlock spams `Inbox.SendL2Message` calls in a
// tight loop without waiting for each, letting the L1 miner pack them into a
// small number of L1 blocks. Then forces a batch post that reads them all.
// This exercises the per-block delayed-msg accumulator hashing path.
//
// We include some L2 txs because the BatchPoster won't post a batch with
// zero pending L2 messages, even if many delayed messages are waiting.
// 50 minimal transfers is negligible work next to 300 delayed messages.
func TestMELExtractionMaxDelayedPerBlock(t *testing.T) {
	runMELExtractionStress(t, 300, 50, "mel-step-stress-max-delayed", "MAX-DELAYED-PER-BLOCK")
}

// runMELExtractionStress is the shared driver for the three step-stress tests.
//
//   - numDelayedMsgs: how many `Inbox.SendL2Message` calls to make on L1.
//     Sent without waiting for each tx individually so the L1 miner can pack
//     many into a single block.
//   - numL2Txs: how many minimal L2 transfers to send to the sequencer before
//     forcing a batch post. They all land in one batch (subject to
//     BatchPoster size thresholds).
//   - slug: subdirectory under ~/.arbitrum/validation-inputs/ for the output.
//   - summaryLabel: header used in the printed summary.
func runMELExtractionStress(
	t *testing.T,
	numDelayedMsgs int,
	numL2Txs int,
	slug string,
	summaryLabel string,
) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// --- Setup: full Arbitrum stack with MEL extraction + validation enabled ---
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.L2Info.GenerateAccount("User2")
	builder.nodeConfig.BatchPoster.MaxDelay = time.Hour
	builder.nodeConfig.BatchPoster.PollInterval = time.Hour
	builder.nodeConfig.MessageExtraction.Enable = true
	builder.nodeConfig.MessageExtraction.RetryInterval = 100 * time.Millisecond
	builder.nodeConfig.MELValidator.Enable = true
	// BlockValidator must be enabled for MELValidator to share its spawner setup.
	// We don't actually wait for it to validate every L2 message — we just call
	// MELValidator.CreateNextValidationEntry directly once MEL has caught up.
	builder.nodeConfig.BlockValidator.Enable = true
	builder.nodeConfig.BlockValidator.EnableMEL = true
	builder.nodeConfig.BlockValidator.ForwardBlocks = 0
	builder.nodeConfig.BlockValidator.ClearMsgPreimagesPoll = time.Hour
	cleanup := builder.Build(t)
	defer cleanup()

	l1Client := builder.L1.Client
	l1Info := builder.L1Info

	testClientB, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{})
	defer cleanupB()

	// Capture the L1 block number BEFORE any test-induced traffic. The MEL
	// extraction range will start from this point, ensuring all subsequent
	// delayed-msg posts and the batch land within the validation window.
	startMELState, err := builder.L2.ConsensusNode.MessageExtractor.GetHeadState()
	Require(t, err)
	startPCB := startMELState.ParentChainBlockNumber
	t.Logf("Starting MEL state: PCB=%d, MsgCount=%d, DelayedRead=%d",
		startPCB, startMELState.MsgCount, startMELState.DelayedMessagesRead)

	// --- Phase 1: Send many delayed messages via L1 inbox ---
	if numDelayedMsgs > 0 {
		t.Logf("Phase 1: Sending %d delayed messages via L1 inbox (tightly packed)...", numDelayedMsgs)
		delayedInbox, err := bridgegen.NewInbox(l1Info.GetAddress("Inbox"), l1Client)
		Require(t, err)
		// With MEL enabled, InboxTracker is nil — use MessageExtractor instead.
		delayedCountBefore, err := builder.L2.ConsensusNode.MessageExtractor.GetDelayedCount()
		Require(t, err)
		// Submit all txs without waiting for each — the L1 miner will pack them.
		// Manually set nonce per call (the auto-nonce path queries the client
		// for "latest", which returns the same value before any tx mines, so
		// every call after the first would be a "replacement underpriced" dup).
		l1Txs := make([]*types.Transaction, 0, numDelayedMsgs)
		usertxopts := l1Info.GetDefaultTransactOpts("User", ctx)
		startNonce, err := l1Client.PendingNonceAt(ctx, usertxopts.From)
		Require(t, err)
		for i := 0; i < numDelayedMsgs; i++ {
			tx := builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas,
				big.NewInt(int64(i+1)*1e6), nil)
			txBytes, err := tx.MarshalBinary()
			Require(t, err)
			txWrapped := append([]byte{arbos.L2MessageKind_SignedTx}, txBytes...)
			usertxopts.Nonce = new(big.Int).SetUint64(startNonce + uint64(i))
			l1tx, err := delayedInbox.SendL2Message(&usertxopts, txWrapped)
			Require(t, err)
			l1Txs = append(l1Txs, l1tx)
		}
		// Now wait for them all.
		for _, l1tx := range l1Txs {
			_, err := EnsureTxSucceeded(ctx, l1Client, l1tx)
			Require(t, err)
		}
		AdvanceL1(t, ctx, l1Client, l1Info, 30)

		// Wait for the message extractor to register all delayed messages.
		deadline := time.Now().Add(2 * time.Minute)
		for time.Now().Before(deadline) {
			count, err := builder.L2.ConsensusNode.MessageExtractor.GetDelayedCount()
			Require(t, err)
			if count >= delayedCountBefore+uint64(numDelayedMsgs) {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		t.Logf("  All %d delayed messages registered", numDelayedMsgs)
	}

	// --- Phase 2: Send many L2 txs (the sequencer will batch them) ---
	if numL2Txs > 0 {
		t.Logf("Phase 2: Sending %d L2 transactions to sequencer...", numL2Txs)
		var l2Txs types.Transactions
		for i := 0; i < numL2Txs; i++ {
			tx := builder.L2Info.PrepareTx("Faucet", "User2", builder.L2Info.TransferGas,
				big.NewInt(1), nil)
			Require(t, builder.L2.Client.SendTransaction(ctx, tx))
			l2Txs = append(l2Txs, tx)
		}
		// Wait for the LAST tx — sequential nonces from same account guarantee
		// all earlier txs already mined. (Waiting per-tx is too slow at 20K+.)
		_, err := builder.L2.EnsureTxSucceeded(l2Txs[len(l2Txs)-1])
		Require(t, err)
		// Give the BatchPoster a moment to register the new messages.
		time.Sleep(2 * time.Second)
		t.Logf("  All %d L2 txs accepted by sequencer", numL2Txs)
	}

	// --- Phase 3: Force a batch post (reads delayed messages too) ---
	t.Log("Phase 3: Posting sequencer batch (reads pending delayed messages)...")
	initialBatchCount := GetBatchCount(t, builder)
	builder.nodeConfig.BatchPoster.MaxDelay = 0
	builder.L2.ConsensusConfigFetcher.Set(builder.nodeConfig)
	posted, err := builder.L2.ConsensusNode.BatchPoster.MaybePostSequencerBatch(ctx)
	Require(t, err)
	if !posted {
		Fatal(t, "sequencer batch was not posted")
	}
	// MaybePostSequencerBatch returns true once the batch tx is sent to L1, but
	// the L1 tx hasn't necessarily been mined yet (so on-chain batch count
	// hasn't moved). Drain any follow-on batches the BatchPoster wants to
	// produce, then wait for the on-chain batch count to advance.
	for {
		more, err := builder.L2.ConsensusNode.BatchPoster.MaybePostSequencerBatch(ctx)
		Require(t, err)
		if !more {
			break
		}
	}
	batchDeadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(batchDeadline) {
		if GetBatchCount(t, builder) > initialBatchCount {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	finalBatchCount := GetBatchCount(t, builder)
	if finalBatchCount <= initialBatchCount {
		Fatal(t, "no batches posted (timed out waiting for L1 batch tx)")
	}
	_ = testClientB
	builder.nodeConfig.BatchPoster.MaxDelay = time.Hour
	builder.L2.ConsensusConfigFetcher.Set(builder.nodeConfig)

	// --- Phase 4: Wait for MEL to catch up ---
	// We only need MEL extraction to have processed the L1 blocks containing
	// our delayed messages and the batch. We don't need any L2 execution
	// validation — that's separate (and would be very slow).
	t.Log("Phase 4: Waiting for MEL to catch up...")
	AdvanceL1(t, ctx, l1Client, l1Info, 40)
	extractedMsgCount, err := builder.L2.ConsensusNode.TxStreamer.GetMessageCount()
	Require(t, err)
	melDeadline := time.Now().Add(5 * time.Minute)
	for time.Now().Before(melDeadline) {
		melMsgCount, err := builder.L2.ConsensusNode.MessageExtractor.GetMsgCount()
		Require(t, err)
		if melMsgCount >= extractedMsgCount {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Logf("  MEL caught up to msg %d", extractedMsgCount-1)

	// --- Phase 5: Generate the MEL-extraction validation entry ---
	t.Log("Phase 5: Generating MEL-extraction validation entry...")
	melValidator := builder.L2.ConsensusNode.MELValidator
	headState, err := builder.L2.ConsensusNode.MessageExtractor.GetHeadState()
	Require(t, err)
	t.Logf("  Head MEL state: PCB=%d, MsgCount=%d, DelayedRead=%d",
		headState.ParentChainBlockNumber, headState.MsgCount, headState.DelayedMessagesRead)
	// CreateNextValidationEntry's loop breaks when `endState.MsgCount >
	// validateMsgExtractionTill`. Passing headState.MsgCount exactly means the
	// loop never breaks (no MEL state has MsgCount > head), iterates past the
	// chain head, and returns nil. Pass MsgCount-1 so it breaks after reaching
	// the head state.
	target := headState.MsgCount - 1
	t.Logf("  Extracting from PCB=%d to MsgCount=%d (%d-block range)",
		startPCB, target, headState.ParentChainBlockNumber-startPCB)

	entry, _, err := melValidator.CreateNextValidationEntry(ctx, startPCB, target)
	Require(t, err)
	if entry == nil {
		Fatal(t, "CreateNextValidationEntry returned nil entry")
	}

	input, err := entry.ToInput([]rawdb.WasmTarget{})
	Require(t, err)
	inputJSON := server_api.ValidationInputToJson(input)

	// Critical: ensure non-nil slices/maps so Rust deserializer doesn't reject `null`.
	if inputJSON.BatchInfo == nil {
		inputJSON.BatchInfo = []server_api.BatchInfoJson{}
	}
	if inputJSON.UserWasms == nil {
		inputJSON.UserWasms = make(map[rawdb.WasmTarget]map[common.Hash]string)
	}

	// --- Phase 6: Per-block stats so we can see how the work is distributed ---
	maxTxsPerL1Block := 0
	maxGasPerL1Block := uint64(0)
	for blockNum := startPCB + 1; blockNum <= headState.ParentChainBlockNumber; blockNum++ {
		block, err := l1Client.BlockByNumber(ctx, new(big.Int).SetUint64(blockNum))
		if err != nil {
			continue
		}
		if len(block.Transactions()) > maxTxsPerL1Block {
			maxTxsPerL1Block = len(block.Transactions())
		}
		if block.GasUsed() > maxGasPerL1Block {
			maxGasPerL1Block = block.GasUsed()
		}
	}

	// --- Phase 7: Measure and write to disk ---
	jsonBytes, err := inputJSON.Marshal()
	Require(t, err)

	totalPreimageCount := 0
	totalPreimageBytes := 0
	for _, innerMap := range inputJSON.PreimagesB64 {
		totalPreimageCount += len(innerMap.Map)
		for _, v := range innerMap.Map {
			totalPreimageBytes += len(v)
		}
	}

	totalBatchInfoBytes := 0
	for _, b := range inputJSON.BatchInfo {
		totalBatchInfoBytes += len(b.DataB64)
	}

	t.Log("")
	t.Logf("=== MEL EXTRACTION %s SUMMARY ===", summaryLabel)
	t.Logf("  L1 PCB range:              %d → %d (%d blocks)",
		startPCB, headState.ParentChainBlockNumber, headState.ParentChainBlockNumber-startPCB)
	t.Logf("  Max txs in any L1 block:   %d", maxTxsPerL1Block)
	t.Logf("  Max gas in any L1 block:   %d", maxGasPerL1Block)
	t.Logf("  L2 messages extracted:     %d", headState.MsgCount-startMELState.MsgCount)
	t.Logf("  Delayed messages enqueued: %d", numDelayedMsgs)
	t.Logf("  Delayed messages read:     %d",
		headState.DelayedMessagesRead-startMELState.DelayedMessagesRead)
	t.Logf("  Batches in range:          %d (count went %d → %d)",
		finalBatchCount-initialBatchCount, initialBatchCount, finalBatchCount)
	t.Logf("  L2 txs sent to sequencer:  %d", numL2Txs)
	t.Logf("  Total preimages:           %d", totalPreimageCount)
	t.Logf("  Preimage raw bytes:        %d (%.2f MB)",
		totalPreimageBytes, float64(totalPreimageBytes)/(1024*1024))
	t.Logf("  Batch info size (b64):     %d bytes (%.2f MB)",
		totalBatchInfoBytes, float64(totalBatchInfoBytes)/(1024*1024))
	t.Logf("  InputJSON size:            %d bytes (%.2f MB)",
		len(jsonBytes), float64(len(jsonBytes))/(1024*1024))
	t.Logf("  Start MELStateHash:        %s", inputJSON.StartState.MELStateHash.Hex())
	t.Logf("  End ParentChainBlockHash:  %s", inputJSON.EndParentChainBlockHash.Hex())

	writer, err := inputs.NewWriter(
		inputs.WithSlug(slug),
		inputs.WithTimestampDirEnabled(true),
	)
	Require(t, err)
	Require(t, writer.Write(inputJSON))
	t.Log("")
	t.Logf("JSON written to ~/.arbitrum/validation-inputs/%s/<timestamp>/block_inputs_<id>.json", slug)
	t.Log("Run benchbin to measure arbitrator step count & timing:")
	t.Log("  ./target/release/benchbin \\")
	t.Log("    --json-inputs <above path> \\")
	t.Log("    --binary target/machines/latest/machine.v2.wavm.br")
}
