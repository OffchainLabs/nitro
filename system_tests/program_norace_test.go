// race detection makes things slow and miss timeouts
//go:build !race
// +build !race

package arbtest

import (
	"encoding/binary"
	"encoding/json"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/colors"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

// used in program test
func validateBlocks(
	t *testing.T, start uint64, jit bool, builder *NodeBuilder,
) {
	t.Helper()
	if jit || start == 0 {
		start = 1
	}

	blockHeight, err := builder.L2.Client.BlockNumber(builder.ctx)
	Require(t, err)

	blocks := []uint64{}
	for i := start; i <= blockHeight; i++ {
		blocks = append(blocks, i)
	}
	validateBlockRange(t, blocks, jit, builder)
}

// used in program test
func validateBlockRange(
	t *testing.T, blocks []uint64, jit bool,
	builder *NodeBuilder,
) {
	ctx := builder.ctx
	waitForSequencer(t, builder, arbmath.MaxInt(blocks...))
	blockHeight, err := builder.L2.Client.BlockNumber(ctx)
	Require(t, err)

	// validate everything
	if jit {
		blocks = []uint64{}
		for i := uint64(1); i <= blockHeight; i++ {
			blocks = append(blocks, i)
		}
	}

	success := true
	for _, block := range blocks {
		// no classic data, so block numbers are message indicies
		inboxPos := arbutil.MessageIndex(block)

		now := time.Now()
		correct, _, err := builder.L2.ConsensusNode.StatelessBlockValidator.ValidateResult(ctx, inboxPos, false, common.Hash{})
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
		Fatal(t)
	}
}

func TestProgramEvmData(t *testing.T) {
	t.Parallel()
	testEvmData(t, true)
}

func testEvmData(t *testing.T, jit bool) {
	builder, auth, cleanup := setupProgramTest(t, jit)
	ctx := builder.ctx
	l2info := builder.L2Info
	l2client := builder.L2.Client
	defer cleanup()
	evmDataAddr := deployWasm(t, ctx, auth, l2client, rustFile("evm-data"))

	ensure := func(tx *types.Transaction, err error) *types.Receipt {
		t.Helper()
		Require(t, err)
		receipt, err := EnsureTxSucceeded(ctx, l2client, tx)
		Require(t, err)
		return receipt
	}
	burnArbGas, _ := util.NewCallParser(precompilesgen.ArbosTestABI, "burnArbGas")

	_, tx, mock, err := mocksgen.DeployProgramTest(&auth, l2client)
	ensure(tx, err)

	evmDataGas := uint64(1000000000)
	gasToBurn := uint64(1000000)
	callBurnData, err := burnArbGas(new(big.Int).SetUint64(gasToBurn))
	Require(t, err)
	fundedAddr := l2info.Accounts["Faucet"].Address
	ethPrecompile := common.BigToAddress(big.NewInt(1))
	arbTestAddress := types.ArbosTestAddress

	evmDataData := []byte{}
	evmDataData = append(evmDataData, fundedAddr.Bytes()...)
	evmDataData = append(evmDataData, ethPrecompile.Bytes()...)
	evmDataData = append(evmDataData, arbTestAddress.Bytes()...)
	evmDataData = append(evmDataData, evmDataAddr.Bytes()...)
	evmDataData = append(evmDataData, callBurnData...)
	opts := bind.CallOpts{
		From: testhelpers.RandomAddress(),
	}

	result, err := mock.StaticcallEvmData(&opts, evmDataAddr, fundedAddr, evmDataGas, evmDataData)
	Require(t, err)

	advance := func(count int, name string) []byte {
		t.Helper()
		if len(result) < count {
			Fatal(t, "not enough data left", name, count, len(result))
		}
		data := result[:count]
		result = result[count:]
		return data
	}
	getU32 := func(name string) uint32 {
		t.Helper()
		return binary.BigEndian.Uint32(advance(4, name))
	}
	getU64 := func(name string) uint64 {
		t.Helper()
		return binary.BigEndian.Uint64(advance(8, name))
	}

	inkPrice := uint64(getU32("ink price"))
	gasLeftBefore := getU64("gas left before")
	inkLeftBefore := getU64("ink left before")
	gasLeftAfter := getU64("gas left after")
	inkLeftAfter := getU64("ink left after")

	gasUsed := gasLeftBefore - gasLeftAfter
	calculatedGasUsed := (inkLeftBefore - inkLeftAfter) / inkPrice

	// Should be within 1 gas
	if !arbmath.Within(gasUsed, calculatedGasUsed, 1) {
		Fatal(t, "gas and ink converted to gas don't match", gasUsed, calculatedGasUsed, inkPrice)
	}

	tx = l2info.PrepareTxTo("Owner", &evmDataAddr, evmDataGas, nil, evmDataData)
	ensure(tx, l2client.SendTransaction(ctx, tx))

	// test hostio tracing
	js := `{
            "hostio": function(info) { this.names.push(info.name); },
            "result": function() { return this.names; },
            "fault":  function() { return this.names; },
            names: []
        }`
	var trace json.RawMessage
	traceConfig := &tracers.TraceConfig{
		Tracer: &js,
	}
	rpc := l2client.Client()
	err = rpc.CallContext(ctx, &trace, "debug_traceTransaction", tx.Hash(), traceConfig)
	Require(t, err)

	for _, item := range []string{"user_entrypoint", "read_args", "write_result", "user_returned"} {
		if !strings.Contains(string(trace), item) {
			Fatal(t, "tracer missing hostio ", item, " ", trace)
		}
	}
	colors.PrintGrey("trace: ", string(trace))

	validateBlocks(t, 1, jit, builder)
}
