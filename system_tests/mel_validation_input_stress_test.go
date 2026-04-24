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
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
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
//              PUSH2 runtimeLen, PUSH1 0, RETURN — 15 bytes total.
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
		numL1Blocks = 15
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
