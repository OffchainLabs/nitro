// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package arbtest

import (
	"encoding/binary"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/google/go-cmp/cmp"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestStylusTracer(t *testing.T) {
	const jit = false
	builder, auth, cleanup := setupProgramTest(t, jit)
	ctx := builder.ctx
	l2client := builder.L2.Client
	l2info := builder.L2Info
	rpcClient := builder.L2.Client.Client()
	defer cleanup()

	traceTransaction := func(tx common.Hash, tracer string) []gethexec.HostioTraceInfo {
		traceOpts := struct {
			Tracer string `json:"tracer"`
		}{
			Tracer: tracer,
		}
		var result []gethexec.HostioTraceInfo
		err := rpcClient.CallContext(ctx, &result, "debug_traceTransaction", tx, traceOpts)
		Require(t, err, "trace transaction")
		return result
	}

	// Deploy contracts
	stylusMulticall := deployWasm(t, ctx, auth, l2client, rustFile("multicall"))
	evmMulticall, tx, _, err := mocksgen.DeployMultiCallTest(&auth, builder.L2.Client)
	Require(t, err, "deploy evm multicall")
	_, err = EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err, "ensure evm multicall deployment")

	// Args for tests
	key := testhelpers.RandomHash()
	value := testhelpers.RandomHash()
	loadStoreArgs := multicallEmptyArgs()
	loadStoreArgs = multicallAppendLoad(loadStoreArgs, key, false)
	loadStoreArgs = multicallAppendStore(loadStoreArgs, key, value, false)
	callArgs := argsForMulticall(vm.CALL, stylusMulticall, nil, []byte{0})
	evmCall := argsForMulticall(vm.CALL, evmMulticall, nil, []byte{0})

	for _, testCase := range []struct {
		name     string
		contract common.Address
		args     []byte
		want     []gethexec.HostioTraceInfo
	}{
		{
			name:     "non-recursive hostios",
			contract: stylusMulticall,
			args:     loadStoreArgs,
			want: []gethexec.HostioTraceInfo{
				{Name: "user_entrypoint", Args: intToBe32(len(loadStoreArgs)), Outs: []byte{}},
				{Name: "pay_for_memory_grow", Args: []byte{0x00, 0x01}, Outs: []byte{}},
				{Name: "read_args", Args: []byte{}, Outs: loadStoreArgs},
				{Name: "storage_cache_bytes32", Args: append(key.Bytes(), value.Bytes()...), Outs: []byte{}},
				{Name: "storage_flush_cache", Args: []byte{0x00}, Outs: []byte{}},
				{Name: "storage_load_bytes32", Args: key.Bytes(), Outs: value.Bytes()},
				{Name: "storage_flush_cache", Args: []byte{0x00}, Outs: []byte{}},
				{Name: "write_result", Args: value.Bytes(), Outs: []byte{}},
				{Name: "user_returned", Args: []byte{}, Outs: intToBe32(0)},
			},
		},

		{
			name:     "call stylus contract",
			contract: stylusMulticall,
			args:     callArgs,
			want: []gethexec.HostioTraceInfo{
				{Name: "user_entrypoint", Outs: []byte{}, Args: intToBe32(len(callArgs))},
				{Name: "pay_for_memory_grow", Outs: []byte{}, Args: []byte{0x00, 0x01}},
				{Name: "read_args", Args: []byte{}, Outs: callArgs},
				{
					Name:    "call_contract",
					Args:    append(stylusMulticall.Bytes(), common.Hex2Bytes("ffffffffffffffff000000000000000000000000000000000000000000000000000000000000000000")...),
					Outs:    common.Hex2Bytes("0000000000"),
					Address: &stylusMulticall,
					Steps: &[]gethexec.HostioTraceInfo{
						{Name: "user_entrypoint", Args: intToBe32(1), Outs: []byte{}},
						{Name: "pay_for_memory_grow", Args: []byte{0x00, 0x01}, Outs: []byte{}},
						{Name: "read_args", Args: []byte{}, Outs: []byte{0x00}},
						{Name: "storage_flush_cache", Args: []byte{0x00}, Outs: []byte{}},
						{Name: "write_result", Args: []byte{}, Outs: []byte{}},
						{Name: "user_returned", Args: []byte{}, Outs: intToBe32(0)},
					},
				},
				{Name: "storage_flush_cache", Args: []byte{0x00}, Outs: []byte{}},
				{Name: "write_result", Args: []byte{}, Outs: []byte{}},
				{Name: "user_returned", Args: []byte{}, Outs: intToBe32(0)},
			},
		},

		{
			name:     "call evm contract",
			contract: stylusMulticall,
			args:     evmCall,
			want: []gethexec.HostioTraceInfo{
				{Name: "user_entrypoint", Args: intToBe32(len(callArgs)), Outs: []byte{}},
				{Name: "pay_for_memory_grow", Args: []byte{0x00, 0x01}, Outs: []byte{}},
				{Name: "read_args", Args: []byte{}, Outs: evmCall},
				{
					Name:    "call_contract",
					Args:    append(evmMulticall.Bytes(), common.Hex2Bytes("ffffffffffffffff000000000000000000000000000000000000000000000000000000000000000000")...),
					Outs:    common.Hex2Bytes("0000000000"),
					Address: &evmMulticall,
					Steps:   &[]gethexec.HostioTraceInfo{},
				},
				{Name: "storage_flush_cache", Args: []byte{0x00}, Outs: []byte{}},
				{Name: "write_result", Args: []byte{}, Outs: []byte{}},
				{Name: "user_returned", Args: []byte{}, Outs: intToBe32(0)},
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			to := testCase.contract
			tx := l2info.PrepareTxTo("Owner", &to, l2info.TransferGas, nil, testCase.args)
			err := l2client.SendTransaction(ctx, tx)
			Require(t, err, "send transaction")

			nativeResult := traceTransaction(tx.Hash(), "stylusTracer")
			clearInk(nativeResult)
			if diff := cmp.Diff(testCase.want, nativeResult); diff != "" {
				Fatal(t, "native tracer don't match wanted result", diff)
			}

			jsResult := traceTransaction(tx.Hash(), jsStylusTracer)
			clearInk(jsResult)
			if diff := cmp.Diff(jsResult, nativeResult); diff != "" {
				Fatal(t, "native tracer don't match js trace", diff)
			}
		})
	}
}

func intToBe32(v int) []byte {
	return binary.BigEndian.AppendUint32(nil, uint32(v))
}

func clearInk(trace []gethexec.HostioTraceInfo) {
	for i := range trace {
		trace[i].StartInk = 0
		trace[i].EndInk = 0
		if trace[i].Steps != nil {
			clearInk(*trace[i].Steps)
		}
	}
}

var jsStylusTracer = `
{
    "hostio": function(info) {
        info.args = toHex(info.args);
        info.outs = toHex(info.outs);
        if (this.nests.includes(info.name)) {
            Object.assign(info, this.open.pop());
        }
        this.open.push(info);
    },
    "enter": function(frame) {
        let inner = [];
        this.open.push({
            address: toHex(frame.getTo()),
            steps: inner,
        });
        this.stack.push(this.open); // save where we were
        this.open = inner;
    },
    "exit": function(result) {
        this.open = this.stack.pop();
    },
    "result": function() { return this.open; },
    "fault":  function() { return this.open; },
    stack: [],
    open: [],
    nests: ["call_contract", "delegate_call_contract", "static_call_contract"]
}
`
