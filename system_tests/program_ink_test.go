package arbtest

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

// This file contains tests that check for the exact ink usage of each Hostio, ensuring it doesn't
// change when modifying the implementation.

const HOSTIO_INK uint64 = 8400
const PTR_INK uint64 = 5040
const EVM_API_INK uint64 = 59673

func TestSimpleInkUsage(t *testing.T) {
	t.Parallel()

	builder := setupGasCostTest(t)
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", builder.ctx)
	stylusProgram := deployWasm(t, builder.ctx, auth, builder.L2.Client, rustFile("hostio-test"))
	otherProgram := deployWasm(t, builder.ctx, auth, builder.L2.Client, rustFile("multicall"))
	matchSnake := regexp.MustCompile("_[a-z]")

	for _, tc := range []struct {
		hostio      string
		signature   string
		args        []any
		expectedInk uint64
	}{
		{
			hostio:      "exit_early",
			expectedInk: 0,
		},
		{
			hostio:      "transient_load_bytes32",
			args:        []any{common.HexToHash("dead")},
			expectedInk: HOSTIO_INK + 2*PTR_INK + EVM_API_INK + 1000000,
		},
		{
			hostio:      "transient_store_bytes32",
			args:        []any{common.HexToHash("dead"), common.HexToHash("beef")},
			expectedInk: HOSTIO_INK + 2*PTR_INK + EVM_API_INK + 1000000,
		},
		{
			hostio:      "return_data_size",
			expectedInk: HOSTIO_INK,
		},
		{
			hostio:      "account_balance",
			args:        []any{builder.L2Info.GetAddress("Owner")},
			expectedInk: HOSTIO_INK + 2*PTR_INK + EVM_API_INK + 1000000,
		},
		{
			hostio:      "account_code",
			args:        []any{otherProgram},
			expectedInk: 33160623,
		},
		{
			hostio:      "account_code_size",
			args:        []any{otherProgram},
			expectedInk: 33068073,
		},
		{
			hostio:      "account_codehash",
			args:        []any{otherProgram},
			expectedInk: 26078153,
		},
		{
			hostio:      "evm_gas_left",
			expectedInk: HOSTIO_INK,
		},
		{
			hostio:      "evm_ink_left",
			expectedInk: HOSTIO_INK,
		},
		{
			hostio:      "block_basefee",
			expectedInk: HOSTIO_INK + PTR_INK,
		},
		{
			hostio:      "chainid",
			expectedInk: HOSTIO_INK,
		},
		{
			hostio:      "block_coinbase",
			expectedInk: HOSTIO_INK + PTR_INK,
		},
		{
			hostio:      "block_gas_limit",
			expectedInk: HOSTIO_INK,
		},
		{
			hostio:      "block_number",
			expectedInk: HOSTIO_INK,
		},
		{
			hostio:      "block_timestamp",
			expectedInk: HOSTIO_INK,
		},
		{
			hostio:      "contract_address",
			expectedInk: HOSTIO_INK + PTR_INK,
		},
		{
			hostio:      "math_div",
			args:        []any{big.NewInt(1), big.NewInt(3)},
			expectedInk: 43520,
		},
		{
			hostio:      "math_mod",
			args:        []any{big.NewInt(1), big.NewInt(3)},
			expectedInk: 43520,
		},
		{
			hostio:      "math_add_mod",
			args:        []any{big.NewInt(1), big.NewInt(3), big.NewInt(5)},
			expectedInk: 49560,
		},
		{
			hostio:      "math_mul_mod",
			args:        []any{big.NewInt(1), big.NewInt(3), big.NewInt(5)},
			expectedInk: 52660,
		},
		{
			hostio:      "msg_sender",
			expectedInk: HOSTIO_INK + PTR_INK,
		},
		{
			hostio:      "msg_value",
			expectedInk: HOSTIO_INK + PTR_INK,
		},
		{
			hostio:      "tx_gas_price",
			expectedInk: HOSTIO_INK + PTR_INK,
		},
		{
			hostio:      "tx_ink_price",
			expectedInk: HOSTIO_INK,
		},
		{
			hostio:      "tx_origin",
			expectedInk: HOSTIO_INK + PTR_INK,
		},
	} {
		t.Run(tc.hostio, func(t *testing.T) {
			solFunc := matchSnake.ReplaceAllStringFunc(tc.hostio, func(s string) string {
				return strings.ToUpper(strings.TrimPrefix(s, "_"))
			})
			data := encodeHostioTestCalldata(t, solFunc, tc.args)
			checkInkUsage(t, builder, stylusProgram, tc.hostio, tc.hostio, data, nil, tc.expectedInk)
		})
	}
}

func TestPowInkUsage(t *testing.T) {
	t.Parallel()

	builder := setupGasCostTest(t)
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", builder.ctx)
	stylusProgram := deployWasm(t, builder.ctx, auth, builder.L2.Client, rustFile("hostio-test"))

	hostio := "math_pow"

	for _, tc := range []struct {
		exponentNumBytes uint
		expectedInk      uint64
	}{
		{exponentNumBytes: 1, expectedInk: 61520},
		{exponentNumBytes: 2, expectedInk: 79020},
		{exponentNumBytes: 10, expectedInk: 219020},
		{exponentNumBytes: 32, expectedInk: 604020},
	} {
		name := fmt.Sprintf("%v%v", hostio, tc.exponentNumBytes)
		t.Run(name, func(t *testing.T) {
			exponent := new(big.Int).Lsh(big.NewInt(1), tc.exponentNumBytes*8-1)
			args := []any{big.NewInt(1), exponent}
			data := encodeHostioTestCalldata(t, "mathPow", args)
			checkInkUsage(t, builder, stylusProgram, hostio, name, data, nil, tc.expectedInk)
		})
	}
}

func TestStorageInkCost(t *testing.T) {
	t.Parallel()

	builder := setupGasCostTest(t)
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", builder.ctx)
	stylusProgram := deployWasm(t, builder.ctx, auth, builder.L2.Client, rustFile("multicall"))

	store_hostio := "storage_flush_cache"
	load_hostio := "storage_load_bytes32"

	rander := testhelpers.NewPseudoRandomDataSource(t, 0)
	slot := rander.GetHash()

	writeRandAData := multicallEmptyArgs()
	writeRandAData = multicallAppendStore(writeRandAData, slot, rander.GetHash(), false, false)
	expectedInkValues := []uint64{221068073, 68073} // called twice
	checkInkUsage(t, builder, stylusProgram, store_hostio, "initialWrite", writeRandAData, nil, expectedInkValues...)

	readData := multicallEmptyArgs()
	readData = multicallAppendLoad(readData, slot, false)
	expectedInk := uint64(21068480)
	checkInkUsage(t, builder, stylusProgram, load_hostio, "read", readData, nil, expectedInk)

	writeRandBData := multicallEmptyArgs()
	writeRandBData = multicallAppendStore(writeRandBData, slot, rander.GetHash(), false, false)
	expectedInkValues = []uint64{50068073, 68073} // called twice
	checkInkUsage(t, builder, stylusProgram, store_hostio, "writeAgain", writeRandBData, nil, expectedInkValues...)

	writeZeroData := multicallEmptyArgs()
	writeZeroData = multicallAppendStore(writeZeroData, slot, common.Hash{}, false, false)
	expectedInkValues = []uint64{50068073, 68073} // called twice
	checkInkUsage(t, builder, stylusProgram, store_hostio, "delete", writeZeroData, nil, expectedInkValues...)

	expectedInk = uint64(21068480)
	checkInkUsage(t, builder, stylusProgram, load_hostio, "readZeros", readData, nil, expectedInk)

	expectedInkValues = []uint64{221068073, 68073} // called twice
	checkInkUsage(t, builder, stylusProgram, store_hostio, "writeAgainAgain", writeRandAData, nil, expectedInkValues...)
}

func TestLogInkUsage(t *testing.T) {
	t.Parallel()

	builder := setupGasCostTest(t)
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", builder.ctx)
	stylusProgram := deployWasm(t, builder.ctx, auth, builder.L2.Client, rustFile("hostio-test"))

	hostio := "emit_log"

	for _, tc := range []struct {
		ntopics     int8
		dataSize    uint64
		expectedInk uint64
	}{
		{ntopics: 0, dataSize: 0, expectedInk: 3834454},
		{ntopics: 0, dataSize: 10, expectedInk: 4634454},
		{ntopics: 0, dataSize: 100, expectedInk: 11838194},
		{ntopics: 1, dataSize: 100, expectedInk: 15589954},
		{ntopics: 2, dataSize: 100, expectedInk: 19341714},
		{ntopics: 3, dataSize: 100, expectedInk: 23093474},
		{ntopics: 4, dataSize: 100, expectedInk: 26845234},
	} {
		name := fmt.Sprintf("emitLog%dData%d", tc.ntopics, tc.dataSize)
		t.Run(name, func(t *testing.T) {
			args := []any{
				testhelpers.RandomSlice(tc.dataSize),
				tc.ntopics,
			}
			for t := 0; t < 4; t++ {
				args = append(args, testhelpers.RandomHash())
			}
			data := encodeHostioTestCalldata(t, "emitLog", args)
			checkInkUsage(t, builder, stylusProgram, hostio, name, data, nil, tc.expectedInk)
		})
	}
}

func TestReturnDataInkUsage(t *testing.T) {
	t.Parallel()

	builder := setupGasCostTest(t)
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", builder.ctx)
	stylusProgram := deployWasm(t, builder.ctx, auth, builder.L2.Client, rustFile("multicall"))
	otherStylusProgram := deployWasm(t, builder.ctx, auth, builder.L2.Client, rustFile("hostio-test"))
	otherData := encodeHostioTestCalldata(t, "msgValue", nil)

	hostio := "read_return_data"
	data := argsForMulticall(vm.CALL, otherStylusProgram, nil, otherData)
	expectedInk := uint64(73113)
	checkInkUsage(t, builder, stylusProgram, hostio, hostio, data, nil, expectedInk)
}

func TestCallInkUsage(t *testing.T) {
	t.Parallel()

	builder := setupGasCostTest(t)
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", builder.ctx)
	stylusProgram := deployWasm(t, builder.ctx, auth, builder.L2.Client, rustFile("multicall"))
	otherStylusProgram := deployWasm(t, builder.ctx, auth, builder.L2.Client, watFile("bf/cat"))
	otherEvmProgram := deployEvmContract(t, builder.ctx, auth, builder.L2.Client, mocksgen.HostioTestMetaData)
	otherData := encodeHostioTestCalldata(t, "msgValue", nil)

	for _, tc := range []struct {
		hostio string
		opcode vm.OpCode
	}{
		{hostio: "call_contract", opcode: vm.CALL},
		{hostio: "delegate_call_contract", opcode: vm.DELEGATECALL},
		{hostio: "static_call_contract", opcode: vm.STATICCALL},
	} {
		name := tc.hostio + "/burnGas"
		t.Run(name, func(t *testing.T) {
			arbTest := common.HexToAddress("0x0000000000000000000000000000000000000069")
			burnArbGas, _ := util.NewCallParser(precompilesgen.ArbosTestABI, "burnArbGas")
			burnData, err := burnArbGas(big.NewInt(0))
			Require(t, err)
			data := argsForMulticall(tc.opcode, arbTest, nil, burnData)
			expectedInk := uint64(1146395)
			checkInkUsage(t, builder, stylusProgram, tc.hostio, name, data, nil, expectedInk)
		})

		name = tc.hostio + "/evmContract"
		t.Run(name, func(t *testing.T) {
			data := argsForMulticall(tc.opcode, otherEvmProgram, nil, otherData)
			expectedInk := uint64(28325955)
			checkInkUsage(t, builder, stylusProgram, tc.hostio, name, data, nil, expectedInk)
		})

		name = tc.hostio + "/stylusContract"
		t.Run(name, func(t *testing.T) {
			data := argsForMulticall(tc.opcode, otherStylusProgram, nil, otherData)
			expectedInk := uint64(129905955)
			checkInkUsage(t, builder, stylusProgram, tc.hostio, name, data, nil, expectedInk)
		})
	}

	name := "call_contract/evmContractWithValue"
	t.Run(name, func(t *testing.T) {
		value := big.NewInt(1000)
		data := argsForMulticall(vm.CALL, otherEvmProgram, value, otherData)
		expectedInk := uint64(118325955)
		checkInkUsage(t, builder, stylusProgram, "call_contract", name, data, value, expectedInk)
	})
}

func TestCreateInkUsage(t *testing.T) {
	t.Parallel()

	builder := setupGasCostTest(t)
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", builder.ctx)
	stylusProgram := deployWasm(t, builder.ctx, auth, builder.L2.Client, rustFile("create"))
	deployCode := common.FromHex(mocksgen.ProgramTestMetaData.Bin)

	hostio := "create1"
	data := []byte{0x01}
	data = append(data, (common.Hash{}).Bytes()...) // endowment
	data = append(data, deployCode...)
	expectedInk := uint64(9544172725)
	checkInkUsage(t, builder, stylusProgram, hostio, hostio, data, nil, expectedInk)

	hostio = "create2"
	data = []byte{0x02}
	data = append(data, (common.Hash{}).Bytes()...)            // endowment
	data = append(data, (common.HexToHash("beef")).Bytes()...) // salt
	data = append(data, deployCode...)
	expectedInk = uint64(9552877765)
	checkInkUsage(t, builder, stylusProgram, hostio, hostio, data, nil, expectedInk)
}

func TestKeccakInkUsage(t *testing.T) {
	t.Parallel()

	builder := setupGasCostTest(t)
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", builder.ctx)
	stylusProgram := deployWasm(t, builder.ctx, auth, builder.L2.Client, rustFile("hostio-test"))

	hostio := "native_keccak256"

	for _, tc := range []struct {
		size        uint64
		expectedInk uint64
	}{
		{size: 10, expectedInk: 121800},
		{size: 100, expectedInk: 163800},
		{size: 1000, expectedInk: 751800},
	} {
		name := fmt.Sprintf("keccak%d", tc.size)
		t.Run(name, func(t *testing.T) {
			preImage := testhelpers.RandomSlice(tc.size)
			preImage[len(preImage)-1] = 0
			data := encodeHostioTestCalldata(t, "keccak", []any{preImage})
			checkInkUsage(t, builder, stylusProgram, hostio, name, data, nil, tc.expectedInk)
		})
	}
}

func TestWriteResultInkUsage(t *testing.T) {
	t.Parallel()

	builder := setupGasCostTest(t)
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", builder.ctx)
	stylusProgram := deployWasm(t, builder.ctx, auth, builder.L2.Client, rustFile("hostio-test"))

	hostio := "write_result"

	// writeResultEmpty doesn't return any value
	testname := "write_result_empty"
	data := encodeHostioFromSignature(t, "writeResultEmpty()", nil)
	expectedInk := HOSTIO_INK + 16381*2
	checkInkUsage(t, builder, stylusProgram, hostio, testname, data, nil, expectedInk)

	// writeResult(uint256) returns an array of uint256
	testname = "write_result_10000"
	numberOfElementsInReturnedArray := uint64(10000)
	data = encodeHostioFromSignature(t, "writeResult(uint256)", []uint64{numberOfElementsInReturnedArray})
	arrayOverhead := uint64(32 + 32) // 32 bytes for the array length and 32 bytes for the array offset
	expectedInk = HOSTIO_INK + (16381+55*(32*numberOfElementsInReturnedArray+arrayOverhead-32))*2
	checkInkUsage(t, builder, stylusProgram, hostio, testname, data, nil, expectedInk)

	testname = "write_result_0"
	numberOfElementsInReturnedArray = 0
	data = encodeHostioFromSignature(t, "writeResult(uint256)", []uint64{numberOfElementsInReturnedArray})
	expectedInk = HOSTIO_INK + (16381+55*(arrayOverhead-32))*2
	checkInkUsage(t, builder, stylusProgram, hostio, testname, data, nil, expectedInk)
}

func TestReadArgsInkUsage(t *testing.T) {
	t.Parallel()

	builder := setupGasCostTest(t)
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", builder.ctx)
	stylusProgram := deployWasm(t, builder.ctx, auth, builder.L2.Client, rustFile("hostio-test"))

	hostio := "read_args"

	testname := "read_args_0"
	data := encodeHostioFromSignature(t, "readArgsNoArgs()", nil)
	expectedInk := HOSTIO_INK + 5040
	checkInkUsage(t, builder, stylusProgram, hostio, testname, data, nil, expectedInk)

	testname = "read_args_1"
	data = encodeHostioFromSignature(t, "readArgsOneArg(uint256)", []uint64{1})
	signatureOverhead := uint64(4)
	expectedInk = HOSTIO_INK + 5040 + 30*(32+signatureOverhead-32)
	checkInkUsage(t, builder, stylusProgram, hostio, testname, data, nil, expectedInk)

	testname = "read_args_3"
	signature := "readArgsThreeArgs(uint256,uint256,uint256)"
	data = encodeHostioFromSignature(t, signature, []uint64{1, 2, 3})
	expectedInk = HOSTIO_INK + 5040 + 30*(3*32+signatureOverhead-32)
	checkInkUsage(t, builder, stylusProgram, hostio, testname, data, nil, expectedInk)
}

func TestMsgReentrantInkUsage(t *testing.T) {
	t.Parallel()

	builder := setupGasCostTest(t)
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", builder.ctx)
	stylusProgram := deployWasm(t, builder.ctx, auth, builder.L2.Client, rustFile("hostio-test"))

	hostio := "msg_reentrant"

	data := encodeHostioFromSignature(t, "writeResultEmpty()", nil)
	checkInkUsage(t, builder, stylusProgram, hostio, hostio, data, nil, HOSTIO_INK)
}

func TestStorageCacheBytes32InkUsage(t *testing.T) {
	t.Parallel()

	builder := setupGasCostTest(t)
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", builder.ctx)
	stylusProgram := deployWasm(t, builder.ctx, auth, builder.L2.Client, rustFile("hostio-test"))

	hostio := "storage_cache_bytes32"

	data := encodeHostioFromSignature(t, "storageCacheBytes32()", nil)
	expectedInk := HOSTIO_INK + (13440-HOSTIO_INK)*2
	checkInkUsage(t, builder, stylusProgram, hostio, hostio, data, nil, expectedInk)
}

func TestPayForMemoryGrowInkUsage(t *testing.T) {
	t.Parallel()

	builder := setupGasCostTest(t)
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", builder.ctx)
	stylusProgram := deployWasm(t, builder.ctx, auth, builder.L2.Client, rustFile("hostio-test"))

	hostio := "pay_for_memory_grow"
	signature := "payForMemoryGrow(uint256)"

	testname := "pay_for_memory_grow_100"
	data := encodeHostioFromSignature(t, signature, []uint64{100})
	expectedInk := uint64(9320660000)
	checkInkUsage(t, builder, stylusProgram, hostio, testname, data, nil, expectedInk)

	testname = "pay_for_memory_grow_0"
	data = encodeHostioFromSignature(t, signature, []uint64{0})
	expectedInk = HOSTIO_INK
	checkInkUsage(t, builder, stylusProgram, hostio, testname, data, nil, expectedInk)
}

func checkInkUsage(
	t *testing.T,
	builder *NodeBuilder,
	stylusProgram common.Address,
	hostio string,
	testName string,
	data []byte,
	value *big.Int,
	expectedInkValues ...uint64,
) {
	const txGas uint64 = 32_000_000
	tx := builder.L2Info.PrepareTxTo("Owner", &stylusProgram, txGas, value, data)

	err := builder.L2.Client.SendTransaction(builder.ctx, tx)
	Require(t, err, "testName", testName)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err, "testName", testName)

	stylusInkUsage, err := stylusHostiosInkUsage(builder.ctx, builder.L2.Client.Client(), tx)
	Require(t, err, "testName", testName)

	_, ok := stylusInkUsage[hostio]
	if !ok {
		Fatal(t, "hostio not found in ink usage", "hostio", hostio, "stylusInkUsage", stylusInkUsage, "testName", testName)
	}

	if len(stylusInkUsage[hostio]) != len(expectedInkValues) {
		Fatal(t, "unexpected number of ink usage", "hostio", hostio, "stylusInkUsage", stylusInkUsage, "testName", testName)
	}

	for i, expectedInk := range expectedInkValues {
		returnedInk := stylusInkUsage[hostio][i]
		if expectedInk != returnedInk {
			Fatal(t, "unexpected ink usage", "hostio", hostio, "expected", expectedInk, "returned", returnedInk, "testName", testName)
		}
	}
}

func stylusHostiosInkUsage(ctx context.Context, rpcClient rpc.ClientInterface, tx *types.Transaction) (
	map[string][]uint64, error) {

	traceOpts := struct {
		Tracer string `json:"tracer"`
	}{
		Tracer: "stylusTracer",
	}
	var result []gethexec.HostioTraceInfo
	err := rpcClient.CallContext(ctx, &result, "debug_traceTransaction", tx.Hash(), traceOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to trace stylus call: %w", err)
	}

	inkUsage := map[string][]uint64{}
	for _, hostioLog := range result {
		inkCost := hostioLog.StartInk - hostioLog.EndInk
		inkUsage[hostioLog.Name] = append(inkUsage[hostioLog.Name], inkCost)
	}
	return inkUsage, nil
}

func encodeHostioTestCalldata(t *testing.T, solFunc string, args []any) []byte {
	packer, _ := util.NewCallParser(mocksgen.HostioTestABI, solFunc)
	data, err := packer(args...)
	Require(t, err)
	return data
}

// For the functions that are not in the Hostio interface, we encoded them manually
func encodeHostioFromSignature(t *testing.T, signature string, args []uint64) []byte {
	data := crypto.Keccak256([]byte(signature))[:4]
	for _, arg := range args {
		data = append(data, make([]byte, 24)...) // padding
		data = binary.BigEndian.AppendUint64(data, arg)
	}
	return data
}
