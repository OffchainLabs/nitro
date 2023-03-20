// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package nodeInterface

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/gethhook"
	"github.com/offchainlabs/nitro/precompiles"
	"github.com/offchainlabs/nitro/solgen/go/node_interfacegen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/arbmath"
)

type addr = common.Address
type mech = *vm.EVM
type huge = *big.Int
type hash = common.Hash
type bytes32 = [32]byte
type ctx = *precompiles.Context

type Message = types.Message
type BackendAPI = core.NodeInterfaceBackendAPI
type ExecutionResult = core.ExecutionResult

func init() {
	gethhook.RequireHookedGeth()

	nodeInterfaceImpl := &NodeInterface{Address: types.NodeInterfaceAddress}
	nodeInterfaceMeta := node_interfacegen.NodeInterfaceMetaData
	_, nodeInterface := precompiles.MakePrecompile(nodeInterfaceMeta, nodeInterfaceImpl)

	nodeInterfaceDebugImpl := &NodeInterfaceDebug{Address: types.NodeInterfaceDebugAddress}
	nodeInterfaceDebugMeta := node_interfacegen.NodeInterfaceDebugMetaData
	_, nodeInterfaceDebug := precompiles.MakePrecompile(nodeInterfaceDebugMeta, nodeInterfaceDebugImpl)

	core.InterceptRPCMessage = func(
		msg Message,
		ctx context.Context,
		statedb *state.StateDB,
		header *types.Header,
		backend core.NodeInterfaceBackendAPI,
	) (Message, *ExecutionResult, error) {
		to := msg.To()
		arbosVersion := arbosState.ArbOSVersion(statedb) // check ArbOS has been installed
		if to != nil && arbosVersion != 0 {
			var precompile precompiles.ArbosPrecompile
			var swapMessages bool
			returnMessage := &Message{}
			var address addr

			switch *to {
			case types.NodeInterfaceAddress:
				address = types.NodeInterfaceAddress
				duplicate := *nodeInterfaceImpl
				duplicate.backend = backend
				duplicate.context = ctx
				duplicate.header = header
				duplicate.sourceMessage = msg
				duplicate.returnMessage.message = returnMessage
				duplicate.returnMessage.changed = &swapMessages
				precompile = nodeInterface.CloneWithImpl(&duplicate)
			case types.NodeInterfaceDebugAddress:
				address = types.NodeInterfaceDebugAddress
				duplicate := *nodeInterfaceDebugImpl
				duplicate.backend = backend
				duplicate.context = ctx
				duplicate.header = header
				duplicate.sourceMessage = msg
				duplicate.returnMessage.message = returnMessage
				duplicate.returnMessage.changed = &swapMessages
				precompile = nodeInterfaceDebug.CloneWithImpl(&duplicate)
			default:
				return msg, nil, nil
			}

			evm, vmError, err := backend.GetEVM(ctx, msg, statedb, header, &vm.Config{NoBaseFee: true})
			if err != nil {
				return msg, nil, err
			}
			go func() {
				<-ctx.Done()
				evm.Cancel()
			}()
			core.ReadyEVMForL2(evm, msg)

			output, gasLeft, err := precompile.Call(
				msg.Data(), address, address, msg.From(), msg.Value(), false, msg.Gas(), evm,
			)
			if err != nil {
				return msg, nil, err
			}
			if swapMessages {
				return *returnMessage, nil, nil
			}
			res := &ExecutionResult{
				UsedGas:       msg.Gas() - gasLeft,
				Err:           nil,
				ReturnData:    output,
				ScheduledTxes: nil,
			}
			return msg, res, vmError()
		}
		return msg, nil, nil
	}

	core.InterceptRPCGasCap = func(gascap *uint64, msg Message, header *types.Header, statedb *state.StateDB) {
		if *gascap == 0 {
			// It's already unlimited
			return
		}
		arbosVersion := arbosState.ArbOSVersion(statedb)
		if arbosVersion == 0 {
			// ArbOS hasn't been installed, so use the vanilla gas cap
			return
		}
		state, err := arbosState.OpenSystemArbosState(statedb, nil, true)
		if err != nil {
			log.Error("failed to open ArbOS state", "err", err)
			return
		}
		if header.BaseFee.Sign() == 0 {
			// if gas is free or there's no reimbursable poster, the user won't pay for L1 data costs
			return
		}

		posterCost, _ := state.L1PricingState().PosterDataCost(msg, l1pricing.BatchPosterAddress)
		posterCostInL2Gas := arbmath.BigToUintSaturating(arbmath.BigDiv(posterCost, header.BaseFee))
		*gascap = arbmath.SaturatingUAdd(*gascap, posterCostInL2Gas)
	}

	core.GetArbOSSpeedLimitPerSecond = func(statedb *state.StateDB) (uint64, error) {
		arbosVersion := arbosState.ArbOSVersion(statedb)
		if arbosVersion == 0 {
			return 0.0, errors.New("ArbOS not installed")
		}
		state, err := arbosState.OpenSystemArbosState(statedb, nil, true)
		if err != nil {
			log.Error("failed to open ArbOS state", "err", err)
			return 0.0, err
		}
		pricing := state.L2PricingState()
		speedLimit, err := pricing.SpeedLimitPerSecond()
		if err != nil {
			log.Error("failed to get the speed limit", "err", err)
			return 0.0, err
		}
		return speedLimit, nil
	}

	arbSys, err := precompilesgen.ArbSysMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	l2ToL1TxTopic = arbSys.Events["L2ToL1Tx"].ID
	l2ToL1TransactionTopic = arbSys.Events["L2ToL1Transaction"].ID
	merkleTopic = arbSys.Events["SendMerkleUpdate"].ID
}

func gethExecFromNodeInterfaceBackend(backend BackendAPI) (*gethexec.ExecutionNode, error) {
	apiBackend, ok := backend.(*arbitrum.APIBackend)
	if !ok {
		return nil, errors.New("API backend isn't Arbitrum")
	}
	exec, ok := apiBackend.GetArbitrumNode().(*gethexec.ExecutionNode)
	if !ok {
		return nil, errors.New("failed to get Arbitrum Node from backend")
	}
	return exec, nil
}

func blockchainFromNodeInterfaceBackend(backend BackendAPI) (*core.BlockChain, error) {
	apiBackend, ok := backend.(*arbitrum.APIBackend)
	if !ok {
		return nil, errors.New("API backend isn't Arbitrum")
	}
	bc := apiBackend.BlockChain()
	if bc == nil {
		return nil, errors.New("failed to get Blockchain from backend")
	}
	return bc, nil
}
