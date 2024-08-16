package arbtest

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

type multiCallRecurse struct {
	Name   string
	opcode vm.OpCode
}

func printRecurse(recurse []multiCallRecurse) string {
	result := ""
	for _, contract := range recurse {
		result = result + "(" + contract.Name + "," + contract.opcode.String() + ")"
	}
	return result
}

func testProgramRecursiveCall(t *testing.T, builder *NodeBuilder, slotVals map[string]common.Hash, rander *testhelpers.PseudoRandomDataSource, recurse []multiCallRecurse) uint64 {
	ctx := builder.ctx
	slot := common.HexToHash("0x11223344556677889900aabbccddeeff")
	val := common.Hash{}
	var args []byte
	if recurse[0].opcode == vm.SSTORE {
		// send event from storage on sstore
		val = rander.GetHash()
		args = append([]byte{0x1, 0, 0, 0, 65, 0x18}, slot[:]...)
		args = append(args, val[:]...)
	} else if recurse[0].opcode == vm.SLOAD {
		args = append([]byte{0x1, 0, 0, 0, 33, 0x11}, slot[:]...)
	} else {
		t.Fatal("first level must be sload or sstore")
	}
	shouldSucceed := true
	delegateChangesStorageDest := true
	storageDest := recurse[0].Name
	for i := 1; i < len(recurse); i++ {
		call := recurse[i]
		prev := recurse[i-1]
		args = argsForMulticall(call.opcode, builder.L2Info.GetAddress(prev.Name), nil, args)
		if call.opcode == vm.STATICCALL && recurse[0].opcode == vm.SSTORE {
			shouldSucceed = false
		}
		if delegateChangesStorageDest && call.opcode == vm.DELEGATECALL {
			storageDest = call.Name
		} else {
			delegateChangesStorageDest = false
		}
	}
	if recurse[0].opcode == vm.SLOAD {
		// send event from caller on sload
		args[5] = args[5] | 0x8
	}
	multiCaller, err := mocksgen.NewMultiCallTest(builder.L2Info.GetAddress(recurse[len(recurse)-1].Name), builder.L2.Client)
	Require(t, err)
	ownerTransact := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	ownerTransact.GasLimit = 10000000
	tx, err := multiCaller.Fallback(&ownerTransact, args)
	Require(t, err)
	receipt, err := WaitForTx(ctx, builder.L2.Client, tx.Hash(), time.Second*3)
	Require(t, err)

	if shouldSucceed {
		if receipt.Status != types.ReceiptStatusSuccessful {
			log.Error("error when shouldn't", "case", printRecurse(recurse))
			Fatal(t, arbutil.DetailTxError(ctx, builder.L2.Client, tx, receipt))
		}
		if len(receipt.Logs) != 1 {
			Fatal(t, "incorrect number of logs: ", len(receipt.Logs))
		}
		if recurse[0].opcode == vm.SSTORE {
			slotVals[storageDest] = val
			storageEvt, err := multiCaller.ParseStorage(*receipt.Logs[0])
			Require(t, err)
			gotData := common.BytesToHash(storageEvt.Data[:])
			gotSlot := common.BytesToHash(storageEvt.Slot[:])
			if gotData != val || gotSlot != slot || storageEvt.Write != (recurse[0].opcode == vm.SSTORE) {
				Fatal(t, "unexpected event", gotData, val, gotSlot, slot, storageEvt.Write, recurse[0].opcode)
			}
		} else {
			calledEvt, err := multiCaller.ParseCalled(*receipt.Logs[0])
			Require(t, err)
			gotData := common.BytesToHash(calledEvt.ReturnData)
			if gotData != slotVals[storageDest] {
				Fatal(t, "unexpected event", gotData, val, slotVals[storageDest])
			}
		}
	} else if receipt.Status == types.ReceiptStatusSuccessful {
		Fatal(t, "should have failed")
	}
	for contract, expected := range slotVals {
		found, err := builder.L2.Client.StorageAt(ctx, builder.L2Info.GetAddress(contract), slot, receipt.BlockNumber)
		Require(t, err)
		foundHash := common.BytesToHash(found)
		if expected != foundHash {
			Fatal(t, "contract", contract, "expected", expected, "found", foundHash)
		}
	}
	return receipt.BlockNumber.Uint64()
}

func testProgramResursiveCalls(t *testing.T, tests [][]multiCallRecurse, jit bool) {
	builder, auth, cleanup := setupProgramTest(t, jit)
	ctx := builder.ctx
	l2client := builder.L2.Client
	defer cleanup()

	// set-up contracts
	callsAddr := deployWasm(t, ctx, auth, l2client, rustFile("multicall"))
	builder.L2Info.SetContract("multicall-rust", callsAddr)
	multiCallWasm, _ := readWasmFile(t, rustFile("multicall"))
	auth.GasLimit = 32000000 // skip gas estimation
	multicallB := deployContract(t, ctx, auth, l2client, multiCallWasm)
	builder.L2Info.SetContract("multicall-rust-b", multicallB)
	multiAddr, tx, _, err := mocksgen.DeployMultiCallTest(&auth, builder.L2.Client)
	builder.L2Info.SetContract("multicall-evm", multiAddr)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, builder.L2.Client, tx)
	Require(t, err)
	slotVals := make(map[string]common.Hash)
	rander := testhelpers.NewPseudoRandomDataSource(t, 0)

	// set-up validator
	validatorConfig := arbnode.ConfigDefaultL1NonSequencerTest()
	validatorConfig.BlockValidator.Enable = true
	emptyRedisURL := ""
	defaultWasmRootPath := ""
	AddValNode(t, ctx, validatorConfig, jit, emptyRedisURL, defaultWasmRootPath)
	valClient, valCleanup := builder.Build2ndNode(t, &SecondNodeParams{nodeConfig: validatorConfig})
	defer valCleanup()

	// store initial values
	for _, contract := range []string{"multicall-rust", "multicall-rust-b", "multicall-evm"} {
		storeRecure := []multiCallRecurse{
			{
				Name:   contract,
				opcode: vm.SSTORE,
			},
		}
		testProgramRecursiveCall(t, builder, slotVals, rander, storeRecure)
	}

	// execute transactions
	blockNum := uint64(0)
	for {
		item := int(rander.GetUint64()/4) % len(tests)
		blockNum = testProgramRecursiveCall(t, builder, slotVals, rander, tests[item])
		tests[item] = tests[len(tests)-1]
		tests = tests[:len(tests)-1]
		if len(tests)%100 == 0 {
			log.Error("running transactions..", "blockNum", blockNum, "remaining", len(tests))
		}
		if len(tests) == 0 {
			break
		}
	}

	// wait for validation
	for {
		got := valClient.ConsensusNode.BlockValidator.WaitForPos(t, ctx, arbutil.MessageIndex(blockNum), time.Second*10)
		if got {
			break
		}
		log.Error("validating blocks..", "waiting for", blockNum, "validated", valClient.ConsensusNode.BlockValidator.GetValidated())
	}
}

func TestProgramCallSimple(t *testing.T) {
	tests := [][]multiCallRecurse{
		{
			{
				Name:   "multicall-rust",
				opcode: vm.SLOAD,
			},
			{
				Name:   "multicall-rust",
				opcode: vm.STATICCALL,
			},
			{
				Name:   "multicall-rust",
				opcode: vm.DELEGATECALL,
			},
		},
	}
	testProgramResursiveCalls(t, tests, true)
}
