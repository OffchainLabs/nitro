package arbtest

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers/logger"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/solgen/go/localgen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestProgramSimpleCost(t *testing.T) {
	builder := setupGasCostTest(t)
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", builder.ctx)
	stylusProgram := deployWasm(t, builder.ctx, auth, builder.L2.Client, rustFile("hostio-test"))
	evmProgram := deployEvmContract(t, builder.ctx, auth, builder.L2.Client, localgen.HostioTestMetaData)
	otherProgram := deployWasm(t, builder.ctx, auth, builder.L2.Client, rustFile("storage"))
	matchSnake := regexp.MustCompile("_[a-z]")

	for _, tc := range []struct {
		hostio  string
		opcode  vm.OpCode
		params  []any
		maxDiff float64
	}{
		{hostio: "exit_early", opcode: vm.STOP},
		{hostio: "transient_load_bytes32", opcode: vm.TLOAD, params: []any{common.HexToHash("dead")}},
		{hostio: "transient_store_bytes32", opcode: vm.TSTORE, params: []any{common.HexToHash("dead"), common.HexToHash("beef")}},
		{hostio: "return_data_size", opcode: vm.RETURNDATASIZE, maxDiff: 1.5},
		{hostio: "account_balance", opcode: vm.BALANCE, params: []any{builder.L2Info.GetAddress("Owner")}},
		{hostio: "account_code", opcode: vm.EXTCODECOPY, params: []any{otherProgram}},
		{hostio: "account_code_size", opcode: vm.EXTCODESIZE, params: []any{otherProgram}, maxDiff: 0.3},
		{hostio: "account_codehash", opcode: vm.EXTCODEHASH, params: []any{otherProgram}},
		{hostio: "evm_gas_left", opcode: vm.GAS, maxDiff: 1.5},
		{hostio: "evm_ink_left", opcode: vm.GAS, maxDiff: 1.5},
		{hostio: "block_basefee", opcode: vm.BASEFEE, maxDiff: 0.5},
		{hostio: "chainid", opcode: vm.CHAINID, maxDiff: 1.5},
		{hostio: "block_coinbase", opcode: vm.COINBASE, maxDiff: 0.5},
		{hostio: "block_gas_limit", opcode: vm.GASLIMIT, maxDiff: 1.5},
		{hostio: "block_number", opcode: vm.NUMBER, maxDiff: 1.5},
		{hostio: "block_timestamp", opcode: vm.TIMESTAMP, maxDiff: 1.5},
		{hostio: "contract_address", opcode: vm.ADDRESS, maxDiff: 0.5},
		{hostio: "math_div", opcode: vm.DIV, params: []any{big.NewInt(1), big.NewInt(3)}},
		{hostio: "math_mod", opcode: vm.MOD, params: []any{big.NewInt(1), big.NewInt(3)}},
		{hostio: "math_add_mod", opcode: vm.ADDMOD, params: []any{big.NewInt(1), big.NewInt(3), big.NewInt(5)}, maxDiff: 0.7},
		{hostio: "math_mul_mod", opcode: vm.MULMOD, params: []any{big.NewInt(1), big.NewInt(3), big.NewInt(5)}, maxDiff: 0.7},
		{hostio: "msg_sender", opcode: vm.CALLER, maxDiff: 0.5},
		{hostio: "msg_value", opcode: vm.CALLVALUE, maxDiff: 0.5},
		{hostio: "tx_gas_price", opcode: vm.GASPRICE, maxDiff: 0.5},
		{hostio: "tx_ink_price", opcode: vm.GASPRICE, maxDiff: 1.5},
		{hostio: "tx_origin", opcode: vm.ORIGIN, maxDiff: 0.5},
	} {
		t.Run(tc.hostio, func(t *testing.T) {
			solFunc := matchSnake.ReplaceAllStringFunc(tc.hostio, func(s string) string {
				return strings.ToUpper(strings.TrimPrefix(s, "_"))
			})
			packer, _ := util.NewCallParser(localgen.HostioTestABI, solFunc)
			data, err := packer(tc.params...)
			Require(t, err)
			compareGasUsage(t, builder, evmProgram, stylusProgram, data, nil, compareGasForEach, tc.maxDiff, compareGasPair{tc.opcode, tc.hostio})
		})
	}
}

func TestProgramPowCost(t *testing.T) {
	builder := setupGasCostTest(t)
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", builder.ctx)
	stylusProgram := deployWasm(t, builder.ctx, auth, builder.L2.Client, rustFile("hostio-test"))
	evmProgram := deployEvmContract(t, builder.ctx, auth, builder.L2.Client, localgen.HostioTestMetaData)
	packer, _ := util.NewCallParser(localgen.HostioTestABI, "mathPow")

	for _, exponentNumBytes := range []uint{1, 2, 10, 32} {
		name := fmt.Sprintf("exponentNumBytes%v", exponentNumBytes)
		t.Run(name, func(t *testing.T) {
			exponent := new(big.Int).Lsh(big.NewInt(1), exponentNumBytes*8-1)
			params := []any{big.NewInt(1), exponent}
			data, err := packer(params...)
			Require(t, err)
			evmGasUsage, stylusGasUsage := measureGasUsage(t, builder, evmProgram, stylusProgram, data, nil)
			expectedGas := 2.652 + 1.75*float64(exponentNumBytes+1)
			t.Logf("evm EXP usage: %v - stylus math_pow usage: %v - expected math_pow usage: %v",
				evmGasUsage[vm.EXP][0], stylusGasUsage["math_pow"][0], expectedGas)
			checkPercentDiff(t, stylusGasUsage["math_pow"][0], expectedGas, 0.001)
		})
	}
}

func TestProgramStorageCost(t *testing.T) {
	builder := setupGasCostTest(t)
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", builder.ctx)
	stylusMulticall := deployWasm(t, builder.ctx, auth, builder.L2.Client, rustFile("multicall"))
	evmMulticall := deployEvmContract(t, builder.ctx, auth, builder.L2.Client, localgen.MultiCallTestMetaData)

	const numSlots = 42
	rander := testhelpers.NewPseudoRandomDataSource(t, 0)
	readData := multicallEmptyArgs()
	writeRandAData := multicallEmptyArgs()
	writeRandBData := multicallEmptyArgs()
	writeZeroData := multicallEmptyArgs()
	for i := 0; i < numSlots; i++ {
		slot := rander.GetHash()
		readData = multicallAppendLoad(readData, slot, false)
		writeRandAData = multicallAppendStore(writeRandAData, slot, rander.GetHash(), false, false)
		writeRandBData = multicallAppendStore(writeRandBData, slot, rander.GetHash(), false, false)
		writeZeroData = multicallAppendStore(writeZeroData, slot, common.Hash{}, false, false)
	}

	writePair := compareGasPair{vm.SSTORE, "storage_flush_cache"}
	readPair := compareGasPair{vm.SLOAD, "storage_load_bytes32"}

	for _, tc := range []struct {
		name string
		data []byte
		pair compareGasPair
	}{
		{"initialWrite", writeRandAData, writePair},
		{"read", readData, readPair},
		{"writeAgain", writeRandBData, writePair},
		{"delete", writeZeroData, writePair},
		{"readZeros", readData, readPair},
		{"writeAgainAgain", writeRandAData, writePair},
	} {
		t.Run(tc.name, func(t *testing.T) {
			compareGasUsage(t, builder, evmMulticall, stylusMulticall, tc.data, nil, compareGasSum, 0, tc.pair)
		})
	}
}

func TestProgramLogCost(t *testing.T) {
	builder := setupGasCostTest(t)
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", builder.ctx)
	stylusProgram := deployWasm(t, builder.ctx, auth, builder.L2.Client, rustFile("hostio-test"))
	evmProgram := deployEvmContract(t, builder.ctx, auth, builder.L2.Client, localgen.HostioTestMetaData)
	packer, _ := util.NewCallParser(localgen.HostioTestABI, "emitLog")

	for ntopics := int8(0); ntopics < 5; ntopics++ {
		for _, dataSize := range []uint64{10, 100, 1000} {
			name := fmt.Sprintf("emitLog%dData%d", ntopics, dataSize)
			t.Run(name, func(t *testing.T) {
				args := []any{
					testhelpers.RandomSlice(dataSize),
					ntopics,
				}
				for t := 0; t < 4; t++ {
					args = append(args, testhelpers.RandomHash())
				}
				data, err := packer(args...)
				Require(t, err)
				opcode := vm.LOG0 + vm.OpCode(ntopics)
				compareGasUsage(t, builder, evmProgram, stylusProgram, data, nil, compareGasForEach, 0, compareGasPair{opcode, "emit_log"})
			})
		}
	}
}

func TestProgramCallCost(t *testing.T) {
	builder := setupGasCostTest(t)
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", builder.ctx)
	stylusMulticall := deployWasm(t, builder.ctx, auth, builder.L2.Client, rustFile("multicall"))
	evmMulticall := deployEvmContract(t, builder.ctx, auth, builder.L2.Client, localgen.MultiCallTestMetaData)
	otherStylusProgram := deployWasm(t, builder.ctx, auth, builder.L2.Client, rustFile("hostio-test"))
	otherEvmProgram := deployEvmContract(t, builder.ctx, auth, builder.L2.Client, localgen.HostioTestMetaData)
	packer, _ := util.NewCallParser(localgen.HostioTestABI, "msgValue")
	otherData, err := packer()
	Require(t, err)

	for _, pair := range []compareGasPair{
		{vm.CALL, "call_contract"},
		{vm.DELEGATECALL, "delegate_call_contract"},
		{vm.STATICCALL, "static_call_contract"},
	} {
		t.Run(pair.hostio+"/burnGas", func(t *testing.T) {
			arbTest := common.HexToAddress("0x0000000000000000000000000000000000000069")
			burnArbGas, _ := util.NewCallParser(precompilesgen.ArbosTestABI, "burnArbGas")
			burnData, err := burnArbGas(big.NewInt(0))
			Require(t, err)
			data := argsForMulticall(pair.opcode, arbTest, nil, burnData)
			compareGasUsage(t, builder, evmMulticall, stylusMulticall, data, nil, compareGasForEach, 0, pair)
		})

		t.Run(pair.hostio+"/evmContract", func(t *testing.T) {
			data := argsForMulticall(pair.opcode, otherEvmProgram, nil, otherData)
			compareGasUsage(t, builder, evmMulticall, stylusMulticall, data, nil, compareGasForEach, 0, pair,
				compareGasPair{vm.RETURNDATACOPY, "read_return_data"})
		})

		t.Run(pair.hostio+"/stylusContract", func(t *testing.T) {
			data := argsForMulticall(pair.opcode, otherStylusProgram, nil, otherData)
			compareGasUsage(t, builder, evmMulticall, stylusMulticall, data, nil, compareGasForEach, 0, pair,
				compareGasPair{vm.RETURNDATACOPY, "read_return_data"})
		})

		t.Run(pair.hostio+"/multipleTimes", func(t *testing.T) {
			data := multicallEmptyArgs()
			for i := 0; i < 9; i++ {
				data = multicallAppend(data, pair.opcode, otherEvmProgram, otherData)
			}
			compareGasUsage(t, builder, evmMulticall, stylusMulticall, data, nil, compareGasForEach, 0, pair)
		})
	}

	t.Run("call_contract/evmContractWithValue", func(t *testing.T) {
		value := big.NewInt(1000)
		data := argsForMulticall(vm.CALL, otherEvmProgram, value, otherData)
		compareGasUsage(t, builder, evmMulticall, stylusMulticall, data, value, compareGasForEach, 0, compareGasPair{vm.CALL, "call_contract"})
	})
}

func TestProgramCreateCost(t *testing.T) {
	builder := setupGasCostTest(t)
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", builder.ctx)
	stylusCreate := deployWasm(t, builder.ctx, auth, builder.L2.Client, rustFile("create"))
	evmCreate := deployEvmContract(t, builder.ctx, auth, builder.L2.Client, localgen.CreateTestMetaData)
	deployCode := common.FromHex(localgen.ProgramTestMetaData.Bin)

	t.Run("create1", func(t *testing.T) {
		data := []byte{0x01}
		data = append(data, (common.Hash{}).Bytes()...)
		data = append(data, deployCode...)
		compareGasUsage(t, builder, evmCreate, stylusCreate, data, nil, compareGasForEach, 0, compareGasPair{vm.CREATE, "create1"})
	})

	t.Run("create2", func(t *testing.T) {
		data := []byte{0x02}
		data = append(data, (common.Hash{}).Bytes()...)
		data = append(data, (common.HexToHash("beef")).Bytes()...)
		data = append(data, deployCode...)
		compareGasUsage(t, builder, evmCreate, stylusCreate, data, nil, compareGasForEach, 0, compareGasPair{vm.CREATE2, "create2"})
	})
}

func TestProgramKeccakCost(t *testing.T) {
	builder := setupGasCostTest(t)
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", builder.ctx)
	stylusProgram := deployWasm(t, builder.ctx, auth, builder.L2.Client, rustFile("hostio-test"))
	evmProgram := deployEvmContract(t, builder.ctx, auth, builder.L2.Client, localgen.HostioTestMetaData)
	packer, _ := util.NewCallParser(localgen.HostioTestABI, "keccak")

	for i := 1; i < 5; i++ {
		size := uint64(math.Pow10(i))
		name := fmt.Sprintf("keccak%d", size)
		t.Run(name, func(t *testing.T) {
			preImage := testhelpers.RandomSlice(size)
			preImage[len(preImage)-1] = 0
			data, err := packer(preImage)
			Require(t, err)
			const maxDiff = 2.5
			compareGasUsage(t, builder, evmProgram, stylusProgram, data, nil, compareGasForEach, maxDiff, compareGasPair{vm.KECCAK256, "native_keccak256"})
		})
	}
}

func setupGasCostTest(t *testing.T) *NodeBuilder {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	cleanup := builder.Build(t)
	t.Cleanup(cleanup)
	return builder
}

func deployEvmContract(t *testing.T, ctx context.Context, auth bind.TransactOpts, client *ethclient.Client, metadata *bind.MetaData) common.Address {
	t.Helper()
	parsed, err := metadata.GetAbi()
	Require(t, err)
	address, tx, _, err := bind.DeployContract(&auth, *parsed, common.FromHex(metadata.Bin), client)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)
	return address
}

func measureGasUsage(
	t *testing.T,
	builder *NodeBuilder,
	evmContract common.Address,
	stylusContract common.Address,
	txData []byte,
	txValue *big.Int,
) (map[vm.OpCode][]uint64, map[string][]float64) {
	const txGas uint64 = 32_000_000
	txs := []*types.Transaction{
		builder.L2Info.PrepareTxTo("Owner", &evmContract, txGas, txValue, txData),
		builder.L2Info.PrepareTxTo("Owner", &stylusContract, txGas, txValue, txData),
	}
	receipts := builder.L2.SendWaitTestTransactions(t, txs)

	evmGas := receipts[0].GasUsedForL2()
	evmGasUsage, err := evmOpcodesGasUsage(builder.ctx, builder.L2.Client.Client(), txs[0])
	Require(t, err)

	stylusGas := receipts[1].GasUsedForL2()
	stylusInkUsage, err := stylusHostiosInkUsage(builder.ctx, builder.L2.Client.Client(), txs[1])
	Require(t, err)
	stylusGasUsage := inkToGasMap(stylusInkUsage)

	t.Logf("evm total usage: %v - stylus total usage: %v", evmGas, stylusGas)

	return evmGasUsage, stylusGasUsage
}

type compareGasPair struct {
	opcode vm.OpCode
	hostio string
}

type compareGasMode int

const (
	compareGasForEach compareGasMode = iota
	compareGasSum
)

func compareGasUsage(
	t *testing.T,
	builder *NodeBuilder,
	evmContract common.Address,
	stylusContract common.Address,
	txData []byte,
	txValue *big.Int,
	mode compareGasMode,
	maxAllowedDifference float64,
	pairs ...compareGasPair,
) {
	if evmContract == stylusContract {
		Fatal(t, "evm and stylus contract are the same")
	}
	evmGasUsage, stylusGasUsage := measureGasUsage(t, builder, evmContract, stylusContract, txData, txValue)
	for i := range pairs {
		opcode := pairs[i].opcode
		hostio := pairs[i].hostio
		switch mode {
		case compareGasForEach:
			if len(evmGasUsage[opcode]) != len(stylusGasUsage[hostio]) {
				Fatal(t, "mismatch between opcode ", opcode, " - ", evmGasUsage[opcode], " and hostio ", hostio, " - ", stylusGasUsage[hostio])
			}
			for i := range evmGasUsage[opcode] {
				opcodeGas := evmGasUsage[opcode][i]
				hostioGas := stylusGasUsage[hostio][i]
				t.Logf("evm %v usage: %v - stylus %v usage: %v", opcode, opcodeGas, hostio, hostioGas)
				checkPercentDiff(t, float64(opcodeGas), hostioGas, maxAllowedDifference)
			}
		case compareGasSum:
			evmSum := float64(0)
			for _, v := range evmGasUsage[opcode] {
				evmSum += float64(v)
			}
			stylusSum := float64(0)
			for _, v := range stylusGasUsage[hostio] {
				stylusSum += v
			}
			t.Logf("evm %v usage: %v - stylus %v usage: %v", opcode, evmSum, hostio, stylusSum)
			checkPercentDiff(t, evmSum, stylusSum, maxAllowedDifference)
		}
	}
}

func evmOpcodesGasUsage(ctx context.Context, rpcClient rpc.ClientInterface, tx *types.Transaction) (
	map[vm.OpCode][]uint64, error) {

	var result logger.ExecutionResult
	err := rpcClient.CallContext(ctx, &result, "debug_traceTransaction", tx.Hash(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to trace evm call: %w", err)
	}

	gasUsage := map[vm.OpCode][]uint64{}
	var structLogs []StructLogRes
	for i := range result.StructLogs {
		var structLog StructLogRes
		err := json.Unmarshal(result.StructLogs[i], &structLog)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal struct log: %w", err)
		}
		structLogs = append(structLogs, structLog)
	}
	for i := range structLogs {
		op := vm.StringToOp(structLogs[i].Op)
		gasUsed := uint64(0)
		if op == vm.CALL || op == vm.STATICCALL || op == vm.DELEGATECALL || op == vm.CREATE || op == vm.CREATE2 {
			var gasAfterCall uint64
			var found bool
			for j := i + 1; j < len(structLogs); j++ {
				if structLogs[j].Depth == structLogs[i].Depth {
					gasAfterCall = structLogs[j].Gas + structLogs[j].GasCost
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("malformed log: didn't get back to call original depth")
			}
			if i == 0 {
				return nil, fmt.Errorf("malformed log: call is first opcode")
			}
			gasUsed = structLogs[i-1].Gas - gasAfterCall
		} else {
			gasUsed = structLogs[i].GasCost
		}
		gasUsage[op] = append(gasUsage[op], gasUsed)
	}
	return gasUsage, nil
}

func inkToGasMap(inkUsage map[string][]uint64) map[string][]float64 {
	const InkPerGas = 10000
	gasUsage := map[string][]float64{}
	for hostio, inkArr := range inkUsage {
		gasArr := make([]float64, len(inkArr))
		for i, ink := range inkArr {
			gasArr[i] = float64(ink) / InkPerGas
		}
		gasUsage[hostio] = gasArr
	}
	return gasUsage
}

func checkPercentDiff(t *testing.T, a, b float64, maxAllowedDifference float64) {
	t.Helper()
	if maxAllowedDifference == 0 {
		maxAllowedDifference = 0.25
	}
	percentageDifference := (max(a, b) / min(a, b)) - 1
	if percentageDifference > maxAllowedDifference {
		Fatal(t, fmt.Sprintf("gas usages are too different; got %v, max allowed is %v", percentageDifference, maxAllowedDifference))
	}
}

// Cross-client test implementations

func testProgramSimpleCostWithReplica(t *testing.T, executionClientMode ExecutionClientMode) {
	builder := setupGasCostTest(t)
	replica, replicaCleanup := BuildReplicaWithExecutionMode(t, builder, executionClientMode)
	defer replicaCleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", builder.ctx)
	stylusProgram := deployWasm(t, builder.ctx, auth, builder.L2.Client, rustFile("hostio-test"))
	evmProgram := deployEvmContract(t, builder.ctx, auth, builder.L2.Client, localgen.HostioTestMetaData)
	otherProgram := deployWasm(t, builder.ctx, auth, builder.L2.Client, rustFile("storage"))
	matchSnake := regexp.MustCompile("_[a-z]")

	for _, tc := range []struct {
		hostio  string
		opcode  vm.OpCode
		params  []any
		maxDiff float64
	}{
		{hostio: "exit_early", opcode: vm.STOP},
		{hostio: "transient_load_bytes32", opcode: vm.TLOAD, params: []any{common.HexToHash("dead")}},
		{hostio: "transient_store_bytes32", opcode: vm.TSTORE, params: []any{common.HexToHash("dead"), common.HexToHash("beef")}},
		{hostio: "return_data_size", opcode: vm.RETURNDATASIZE, maxDiff: 1.5},
		{hostio: "account_balance", opcode: vm.BALANCE, params: []any{builder.L2Info.GetAddress("Owner")}},
		{hostio: "account_code", opcode: vm.EXTCODECOPY, params: []any{otherProgram}},
		{hostio: "account_code_size", opcode: vm.EXTCODESIZE, params: []any{otherProgram}, maxDiff: 0.3},
		{hostio: "account_codehash", opcode: vm.EXTCODEHASH, params: []any{otherProgram}},
		{hostio: "evm_gas_left", opcode: vm.GAS, maxDiff: 1.5},
		{hostio: "evm_ink_left", opcode: vm.GAS, maxDiff: 1.5},
		{hostio: "block_basefee", opcode: vm.BASEFEE, maxDiff: 0.5},
		{hostio: "chainid", opcode: vm.CHAINID, maxDiff: 1.5},
		{hostio: "block_coinbase", opcode: vm.COINBASE, maxDiff: 0.5},
		{hostio: "block_gas_limit", opcode: vm.GASLIMIT, maxDiff: 1.5},
		{hostio: "block_number", opcode: vm.NUMBER, maxDiff: 1.5},
		{hostio: "block_timestamp", opcode: vm.TIMESTAMP, maxDiff: 1.5},
		{hostio: "contract_address", opcode: vm.ADDRESS, maxDiff: 0.5},
		{hostio: "math_div", opcode: vm.DIV, params: []any{big.NewInt(1), big.NewInt(3)}},
		{hostio: "math_mod", opcode: vm.MOD, params: []any{big.NewInt(1), big.NewInt(3)}},
		{hostio: "math_add_mod", opcode: vm.ADDMOD, params: []any{big.NewInt(1), big.NewInt(3), big.NewInt(5)}, maxDiff: 0.7},
		{hostio: "math_mul_mod", opcode: vm.MULMOD, params: []any{big.NewInt(1), big.NewInt(3), big.NewInt(5)}, maxDiff: 0.7},
		{hostio: "msg_sender", opcode: vm.CALLER, maxDiff: 0.5},
		{hostio: "msg_value", opcode: vm.CALLVALUE, maxDiff: 0.5},
		{hostio: "tx_gas_price", opcode: vm.GASPRICE, maxDiff: 0.5},
		{hostio: "tx_ink_price", opcode: vm.GASPRICE, maxDiff: 1.5},
		{hostio: "tx_origin", opcode: vm.ORIGIN, maxDiff: 0.5},
	} {
		t.Run(tc.hostio, func(t *testing.T) {
			solFunc := matchSnake.ReplaceAllStringFunc(tc.hostio, func(s string) string {
				return strings.ToUpper(strings.TrimPrefix(s, "_"))
			})
			packer, _ := util.NewCallParser(localgen.HostioTestABI, solFunc)
			data, err := packer(tc.params...)
			Require(t, err)
			compareGasUsage(t, builder, evmProgram, stylusProgram, data, nil, compareGasForEach, tc.maxDiff, compareGasPair{tc.opcode, tc.hostio})
		})
	}

	WaitForReplicaSync(builder.ctx, t, builder.L2.Client, replica.Client, 300)
}

func TestProgramSimpleCostInternal(t *testing.T) {
	testProgramSimpleCostWithReplica(t, ExecutionClientModeInternal)
}
func TestProgramSimpleCostExternal(t *testing.T) {
	testProgramSimpleCostWithReplica(t, ExecutionClientModeExternal)
}
func TestProgramSimpleCostComparison(t *testing.T) {
	testProgramSimpleCostWithReplica(t, ExecutionClientModeComparison)
}

func testProgramPowCostWithReplica(t *testing.T, executionClientMode ExecutionClientMode) {
	builder := setupGasCostTest(t)
	replica, replicaCleanup := BuildReplicaWithExecutionMode(t, builder, executionClientMode)
	defer replicaCleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", builder.ctx)
	stylusProgram := deployWasm(t, builder.ctx, auth, builder.L2.Client, rustFile("hostio-test"))
	evmProgram := deployEvmContract(t, builder.ctx, auth, builder.L2.Client, localgen.HostioTestMetaData)
	packer, _ := util.NewCallParser(localgen.HostioTestABI, "mathPow")

	for _, exponentNumBytes := range []uint{1, 2, 10, 32} {
		name := fmt.Sprintf("exponentNumBytes%v", exponentNumBytes)
		t.Run(name, func(t *testing.T) {
			exponent := new(big.Int).Lsh(big.NewInt(1), exponentNumBytes*8-1)
			params := []any{big.NewInt(1), exponent}
			data, err := packer(params...)
			Require(t, err)
			evmGasUsage, stylusGasUsage := measureGasUsage(t, builder, evmProgram, stylusProgram, data, nil)
			expectedGas := 2.652 + 1.75*float64(exponentNumBytes+1)
			t.Logf("evm EXP usage: %v - stylus math_pow usage: %v - expected math_pow usage: %v",
				evmGasUsage[vm.EXP][0], stylusGasUsage["math_pow"][0], expectedGas)
			checkPercentDiff(t, stylusGasUsage["math_pow"][0], expectedGas, 0.001)
		})
	}

	WaitForReplicaSync(builder.ctx, t, builder.L2.Client, replica.Client, 300)
}

func TestProgramPowCostInternal(t *testing.T) {
	testProgramPowCostWithReplica(t, ExecutionClientModeInternal)
}
func TestProgramPowCostExternal(t *testing.T) {
	testProgramPowCostWithReplica(t, ExecutionClientModeExternal)
}
func TestProgramPowCostComparison(t *testing.T) {
	testProgramPowCostWithReplica(t, ExecutionClientModeComparison)
}

func testProgramStorageCostWithReplica(t *testing.T, executionClientMode ExecutionClientMode) {
	builder := setupGasCostTest(t)
	replica, replicaCleanup := BuildReplicaWithExecutionMode(t, builder, executionClientMode)
	defer replicaCleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", builder.ctx)
	stylusMulticall := deployWasm(t, builder.ctx, auth, builder.L2.Client, rustFile("multicall"))
	evmMulticall := deployEvmContract(t, builder.ctx, auth, builder.L2.Client, localgen.MultiCallTestMetaData)

	const numSlots = 42
	rander := testhelpers.NewPseudoRandomDataSource(t, 0)
	readData := multicallEmptyArgs()
	writeRandAData := multicallEmptyArgs()
	writeRandBData := multicallEmptyArgs()
	writeZeroData := multicallEmptyArgs()
	for i := 0; i < numSlots; i++ {
		slot := rander.GetHash()
		readData = multicallAppendLoad(readData, slot, false)
		writeRandAData = multicallAppendStore(writeRandAData, slot, rander.GetHash(), false, false)
		writeRandBData = multicallAppendStore(writeRandBData, slot, rander.GetHash(), false, false)
		writeZeroData = multicallAppendStore(writeZeroData, slot, common.Hash{}, false, false)
	}

	writePair := compareGasPair{vm.SSTORE, "storage_flush_cache"}
	readPair := compareGasPair{vm.SLOAD, "storage_load_bytes32"}

	for _, tc := range []struct {
		name string
		data []byte
		pair compareGasPair
	}{
		{"initialWrite", writeRandAData, writePair},
		{"read", readData, readPair},
		{"writeAgain", writeRandBData, writePair},
		{"delete", writeZeroData, writePair},
		{"readZeros", readData, readPair},
		{"writeAgainAgain", writeRandAData, writePair},
	} {
		t.Run(tc.name, func(t *testing.T) {
			compareGasUsage(t, builder, evmMulticall, stylusMulticall, tc.data, nil, compareGasSum, 0, tc.pair)
		})
	}

	WaitForReplicaSync(builder.ctx, t, builder.L2.Client, replica.Client, 300)
}

func TestProgramStorageCostInternal(t *testing.T) {
	testProgramStorageCostWithReplica(t, ExecutionClientModeInternal)
}
func TestProgramStorageCostExternal(t *testing.T) {
	testProgramStorageCostWithReplica(t, ExecutionClientModeExternal)
}
func TestProgramStorageCostComparison(t *testing.T) {
	testProgramStorageCostWithReplica(t, ExecutionClientModeComparison)
}

func testProgramLogCostWithReplica(t *testing.T, executionClientMode ExecutionClientMode) {
	builder := setupGasCostTest(t)
	replica, replicaCleanup := BuildReplicaWithExecutionMode(t, builder, executionClientMode)
	defer replicaCleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", builder.ctx)
	stylusProgram := deployWasm(t, builder.ctx, auth, builder.L2.Client, rustFile("hostio-test"))
	evmProgram := deployEvmContract(t, builder.ctx, auth, builder.L2.Client, localgen.HostioTestMetaData)
	packer, _ := util.NewCallParser(localgen.HostioTestABI, "emitLog")

	for ntopics := int8(0); ntopics < 5; ntopics++ {
		for _, dataSize := range []uint64{10, 100, 1000} {
			name := fmt.Sprintf("emitLog%dData%d", ntopics, dataSize)
			t.Run(name, func(t *testing.T) {
				args := []any{
					testhelpers.RandomSlice(dataSize),
					ntopics,
				}
				for i := 0; i < 4; i++ {
					args = append(args, testhelpers.RandomHash())
				}
				data, err := packer(args...)
				Require(t, err)
				opcode := vm.LOG0 + vm.OpCode(ntopics)
				compareGasUsage(t, builder, evmProgram, stylusProgram, data, nil, compareGasForEach, 0, compareGasPair{opcode, "emit_log"})
			})
		}
	}

	WaitForReplicaSync(builder.ctx, t, builder.L2.Client, replica.Client, 300)
}

func TestProgramLogCostInternal(t *testing.T) {
	testProgramLogCostWithReplica(t, ExecutionClientModeInternal)
}
func TestProgramLogCostExternal(t *testing.T) {
	testProgramLogCostWithReplica(t, ExecutionClientModeExternal)
}
func TestProgramLogCostComparison(t *testing.T) {
	testProgramLogCostWithReplica(t, ExecutionClientModeComparison)
}

func testProgramCallCostWithReplica(t *testing.T, executionClientMode ExecutionClientMode) {
	builder := setupGasCostTest(t)
	replica, replicaCleanup := BuildReplicaWithExecutionMode(t, builder, executionClientMode)
	defer replicaCleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", builder.ctx)
	stylusMulticall := deployWasm(t, builder.ctx, auth, builder.L2.Client, rustFile("multicall"))
	evmMulticall := deployEvmContract(t, builder.ctx, auth, builder.L2.Client, localgen.MultiCallTestMetaData)
	otherStylusProgram := deployWasm(t, builder.ctx, auth, builder.L2.Client, rustFile("hostio-test"))
	otherEvmProgram := deployEvmContract(t, builder.ctx, auth, builder.L2.Client, localgen.HostioTestMetaData)
	packer, _ := util.NewCallParser(localgen.HostioTestABI, "msgValue")
	otherData, err := packer()
	Require(t, err)

	for _, pair := range []compareGasPair{
		{vm.CALL, "call_contract"},
		{vm.DELEGATECALL, "delegate_call_contract"},
		{vm.STATICCALL, "static_call_contract"},
	} {
		t.Run(pair.hostio+"/burnGas", func(t *testing.T) {
			arbTest := common.HexToAddress("0x0000000000000000000000000000000000000069")
			burnArbGas, _ := util.NewCallParser(precompilesgen.ArbosTestABI, "burnArbGas")
			burnData, err := burnArbGas(big.NewInt(0))
			Require(t, err)
			data := argsForMulticall(pair.opcode, arbTest, nil, burnData)
			compareGasUsage(t, builder, evmMulticall, stylusMulticall, data, nil, compareGasForEach, 0, pair)
		})

		t.Run(pair.hostio+"/evmContract", func(t *testing.T) {
			data := argsForMulticall(pair.opcode, otherEvmProgram, nil, otherData)
			compareGasUsage(t, builder, evmMulticall, stylusMulticall, data, nil, compareGasForEach, 0, pair,
				compareGasPair{vm.RETURNDATACOPY, "read_return_data"})
		})

		t.Run(pair.hostio+"/stylusContract", func(t *testing.T) {
			data := argsForMulticall(pair.opcode, otherStylusProgram, nil, otherData)
			compareGasUsage(t, builder, evmMulticall, stylusMulticall, data, nil, compareGasForEach, 0, pair,
				compareGasPair{vm.RETURNDATACOPY, "read_return_data"})
		})

		t.Run(pair.hostio+"/multipleTimes", func(t *testing.T) {
			data := multicallEmptyArgs()
			for i := 0; i < 9; i++ {
				data = multicallAppend(data, pair.opcode, otherEvmProgram, otherData)
			}
			compareGasUsage(t, builder, evmMulticall, stylusMulticall, data, nil, compareGasForEach, 0, pair)
		})
	}

	t.Run("call_contract/evmContractWithValue", func(t *testing.T) {
		value := big.NewInt(1000)
		data := argsForMulticall(vm.CALL, otherEvmProgram, value, otherData)
		compareGasUsage(t, builder, evmMulticall, stylusMulticall, data, value, compareGasForEach, 0, compareGasPair{vm.CALL, "call_contract"})
	})

	WaitForReplicaSync(builder.ctx, t, builder.L2.Client, replica.Client, 300)
}

func TestProgramCallCostInternal(t *testing.T) {
	testProgramCallCostWithReplica(t, ExecutionClientModeInternal)
}
func TestProgramCallCostExternal(t *testing.T) {
	testProgramCallCostWithReplica(t, ExecutionClientModeExternal)
}
func TestProgramCallCostComparison(t *testing.T) {
	testProgramCallCostWithReplica(t, ExecutionClientModeComparison)
}

func testProgramCreateCostWithReplica(t *testing.T, executionClientMode ExecutionClientMode) {
	builder := setupGasCostTest(t)
	replica, replicaCleanup := BuildReplicaWithExecutionMode(t, builder, executionClientMode)
	defer replicaCleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", builder.ctx)
	stylusCreate := deployWasm(t, builder.ctx, auth, builder.L2.Client, rustFile("create"))
	evmCreate := deployEvmContract(t, builder.ctx, auth, builder.L2.Client, localgen.CreateTestMetaData)
	deployCode := common.FromHex(localgen.ProgramTestMetaData.Bin)

	t.Run("create1", func(t *testing.T) {
		data := []byte{0x01}
		data = append(data, (common.Hash{}).Bytes()...)
		data = append(data, deployCode...)
		compareGasUsage(t, builder, evmCreate, stylusCreate, data, nil, compareGasForEach, 0, compareGasPair{vm.CREATE, "create1"})
	})

	t.Run("create2", func(t *testing.T) {
		data := []byte{0x02}
		data = append(data, (common.Hash{}).Bytes()...)
		data = append(data, (common.HexToHash("beef")).Bytes()...)
		data = append(data, deployCode...)
		compareGasUsage(t, builder, evmCreate, stylusCreate, data, nil, compareGasForEach, 0, compareGasPair{vm.CREATE2, "create2"})
	})

	WaitForReplicaSync(builder.ctx, t, builder.L2.Client, replica.Client, 300)
}

func TestProgramCreateCostInternal(t *testing.T) {
	testProgramCreateCostWithReplica(t, ExecutionClientModeInternal)
}
func TestProgramCreateCostExternal(t *testing.T) {
	testProgramCreateCostWithReplica(t, ExecutionClientModeExternal)
}
func TestProgramCreateCostComparison(t *testing.T) {
	testProgramCreateCostWithReplica(t, ExecutionClientModeComparison)
}

func testProgramKeccakCostWithReplica(t *testing.T, executionClientMode ExecutionClientMode) {
	builder := setupGasCostTest(t)
	replica, replicaCleanup := BuildReplicaWithExecutionMode(t, builder, executionClientMode)
	defer replicaCleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", builder.ctx)
	stylusProgram := deployWasm(t, builder.ctx, auth, builder.L2.Client, rustFile("hostio-test"))
	evmProgram := deployEvmContract(t, builder.ctx, auth, builder.L2.Client, localgen.HostioTestMetaData)
	packer, _ := util.NewCallParser(localgen.HostioTestABI, "keccak")

	for i := 1; i < 5; i++ {
		size := uint64(math.Pow10(i))
		name := fmt.Sprintf("keccak%d", size)
		t.Run(name, func(t *testing.T) {
			preImage := testhelpers.RandomSlice(size)
			preImage[len(preImage)-1] = 0
			data, err := packer(preImage)
			Require(t, err)
			const maxDiff = 2.5
			compareGasUsage(t, builder, evmProgram, stylusProgram, data, nil, compareGasForEach, maxDiff, compareGasPair{vm.KECCAK256, "native_keccak256"})
		})
	}

	WaitForReplicaSync(builder.ctx, t, builder.L2.Client, replica.Client, 300)
}

func TestProgramKeccakCostInternal(t *testing.T) {
	testProgramKeccakCostWithReplica(t, ExecutionClientModeInternal)
}
func TestProgramKeccakCostExternal(t *testing.T) {
	testProgramKeccakCostWithReplica(t, ExecutionClientModeExternal)
}
func TestProgramKeccakCostComparison(t *testing.T) {
	testProgramKeccakCostWithReplica(t, ExecutionClientModeComparison)
}
