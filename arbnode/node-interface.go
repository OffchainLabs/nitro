//
// Copyright 2021-2022, Offchain Labs, Inc. All rights reserved.
//

package arbnode

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/retryables"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/solgen/go/node_interfacegen"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/merkletree"
)

type Message = types.Message
type ExecutionResult = core.ExecutionResult

// To avoid creating new RPC methods for client-side tooling, nitro Geth's InterceptRPCMessage() hook provides
// an opportunity to swap out the message its handling before deriving a transaction from it.
//
// This function handles messages sent to 0xc8 and uses NodeInterface.sol to determine what to do. No contract
// actually exists at 0xc8, but the abi methods allow the incoming message's calldata to specify the arguments.
//
func ApplyNodeInterface(msg Message, statedb *state.StateDB, nodeInterface abi.ABI) (Message, *ExecutionResult, error) {

	estimateMethod := nodeInterface.Methods["estimateRetryableTicket"]
	outboxMethod := nodeInterface.Methods["constructOutboxProof"]

	calldata := msg.Data()
	if len(calldata) < 4 {
		return msg, nil, errors.New("calldata for NodeInterface.sol is too short")
	}

	if bytes.Equal(estimateMethod.ID, calldata[:4]) {
		inputs, err := estimateMethod.Inputs.Unpack(calldata[4:])
		if err != nil {
			return msg, nil, err
		}
		sender, _ := inputs[0].(common.Address)
		deposit, _ := inputs[1].(*big.Int)
		retryTo, _ := inputs[2].(common.Address)
		l2CallValue, _ := inputs[3].(*big.Int)
		excessFeeRefundAddress, _ := inputs[4].(common.Address)
		callValueRefundAddress, _ := inputs[5].(common.Address)
		retryData, _ := inputs[6].([]byte)

		var pRetryTo *common.Address
		if retryTo != (common.Address{}) {
			pRetryTo = &retryTo
		}

		state, _ := arbosState.OpenSystemArbosState(statedb, true)
		l1BaseFee, _ := state.L1PricingState().L1BaseFeeEstimateWei()
		maxSubmissionFee := retryables.RetryableSubmissionFee(len(retryData), l1BaseFee)

		submitTx := &types.ArbitrumSubmitRetryableTx{
			ChainId:          nil,
			RequestId:        common.Hash{},
			From:             util.RemapL1Address(sender),
			L1BaseFee:        l1BaseFee,
			DepositValue:     deposit,
			GasFeeCap:        msg.GasPrice(),
			Gas:              msg.Gas(),
			RetryTo:          pRetryTo,
			Value:            l2CallValue,
			Beneficiary:      callValueRefundAddress,
			MaxSubmissionFee: maxSubmissionFee,
			FeeRefundAddr:    excessFeeRefundAddress,
			RetryData:        retryData,
		}

		// ArbitrumSubmitRetryableTx is unsigned so the following won't panic
		msg, err := types.NewTx(submitTx).AsMessage(types.NewArbitrumSigner(nil), nil)
		return msg, nil, err
	}

	if bytes.Equal(outboxMethod.ID, calldata[:4]) {
		inputs, err := outboxMethod.Inputs.Unpack(calldata[4:])
		if err != nil {
			return msg, nil, err
		}
		send, _ := inputs[0].(common.Hash)
		root, _ := inputs[1].(common.Hash)
		size, _ := inputs[2].(uint64)
		leaf, _ := inputs[3].(uint64)

		if leaf > size {
			return msg, nil, fmt.Errorf("leaf %v is newer than root %v", leaf, size)
		}

		res, err := nodeInterfaceConstructOutboxProof(msg, send, root, size, leaf)
		return msg, res, err

	}

	return msg, nil, errors.New("method does not exist in NodeInterface.sol")
}

func nodeInterfaceConstructOutboxProof(msg Message, send, root common.Hash, size, leaf uint64) (*ExecutionResult, error) {

	internalError := errors.New("internal error constructing proof")

	balanced := size == arbmath.NextPowerOf2(size)/2
	treeLevels := int(arbmath.Log2ceil(size)) // the # of levels in the tree
	proofLevels := treeLevels - 1             // the # of levels where a hash is needed (all but root)
	walkLevels := treeLevels                  // the # of levels we need to consider when building walks
	if balanced {
		walkLevels -= 1 // skip the root
	}

	// find which nodes we'll want in our proof up to a partial
	query := make([]common.Hash, 0)             // the nodes we'll query for
	nodes := make([]merkletree.LevelAndLeaf, 0) // the nodes needed (might not be found from query)
	which := uint64(1)                          // which bit to flip & set
	place := leaf                               // where we are in the tree
	for level := 0; level < walkLevels; level++ {
		sibling := place ^ which

		position := merkletree.LevelAndLeaf{
			Level: uint64(level),
			Leaf:  sibling,
		}

		if sibling < size {
			// the sibling must not be newer than the root
			query = append(query, common.BigToHash(position.ToBigInt()))
		}
		nodes = append(nodes, position)
		place |= which // set the bit so that we approach from the right
		which <<= 1    // advance to the next bit
	}

	// find all the partials
	partials := make(map[merkletree.LevelAndLeaf]common.Hash)
	if !balanced {
		power := uint64(1) << proofLevels
		total := uint64(0)
		for level := proofLevels; level >= 0; level-- {

			if (power & size) > 0 { // the partials map to the binary representation of the size

				total += power    // The leaf for a given partial is the sum of the powers
				leaf := total - 1 // of 2 preceding it. It's 1 less since we count from 0

				partial := merkletree.LevelAndLeaf{
					Level: uint64(level),
					Leaf:  leaf,
				}

				query = append(query, common.BigToHash(partial.ToBigInt()))
				partials[partial] = common.Hash{}
			}
			power >>= 1
		}
	}

	// collect the logs
	// var logs []types.Log
	logs := []types.Log{}
	if len(query) > 0 {
		/*logs, err = client.FilterLogs(ctx, ethereum.FilterQuery{
			Addresses: []common.Address{
				arbSysAddress,
			},
			Topics: [][]common.Hash{
				{merkleTopic, withdrawTopic},
				nil,
				nil,
				query,
			},
		})
		if err != nil {
			return nil, internalError
		}*/
		_ = query
	}

	known := make(map[merkletree.LevelAndLeaf]common.Hash) // all values in the tree we know
	partialsByLevel := make(map[uint64]common.Hash)        // maps for each level the partial it may have
	var minPartialPlace *merkletree.LevelAndLeaf           // the lowest-level partial

	for _, log := range logs {

		hash := log.Topics[2]
		position := log.Topics[3]

		level := new(big.Int).SetBytes(position[:8]).Uint64()
		leaf := new(big.Int).SetBytes(position[8:]).Uint64()

		place := merkletree.LevelAndLeaf{
			Level: level,
			Leaf:  leaf,
		}

		known[place] = hash

		if _, ok := partials[place]; ok {
			partials[place] = hash
			partialsByLevel[level] = hash
			if minPartialPlace == nil || level < minPartialPlace.Level {
				minPartialPlace = &place
			}
		}
	}

	if !balanced {
		// This tree isn't balanced, so we'll need to use the partials to recover the missing info.
		// To do this, we'll walk the boundry of what's known, computing hashes along the way

		zero := common.Hash{}

		step := *minPartialPlace
		step.Leaf += 1 << step.Level // we start on the min partial's zero-hash sibling
		known[step] = zero

		for step.Level < uint64(treeLevels) {

			curr, ok := known[step]
			if !ok {
				return nil, internalError
			}

			left := curr
			right := curr

			if _, ok := partialsByLevel[step.Level]; ok {
				// a partial on the frontier can only appear on the left
				// moving leftward for a level l skips 2^l leaves
				step.Leaf -= 1 << step.Level
				partial, ok := known[step]
				if !ok {
					return nil, internalError
				}
				left = partial
			} else {
				// getting to the next partial means covering its mirror subtree, so go right
				// moving rightward for a level l skips 2^l leaves
				step.Leaf += 1 << step.Level
				known[step] = zero
				right = zero
			}

			// move to the parent
			step.Level += 1
			step.Leaf |= 1 << (step.Level - 1)
			known[step] = crypto.Keccak256Hash(left.Bytes(), right.Bytes())

			if known[step] != root {
				// a correct walk of the frontier should end with resolving the root
				return nil, internalError
			}
		}
	}

	hashes := make([]common.Hash, len(nodes))
	for i, place := range nodes {
		hash, ok := known[place]
		if !ok {
			return nil, internalError
		}
		hashes[i] = hash
	}

	proof := merkletree.MerkleProof{
		RootHash:  root,
		LeafHash:  send,
		LeafIndex: leaf,
		Proof:     hashes,
	}
	if !proof.IsCorrect() {
		return nil, internalError
	}

	proofBytes := []byte{}
	for _, hash := range hashes {
		proofBytes = append(proofBytes, hash.Bytes()...)
	}

	result := &ExecutionResult{
		UsedGas:       0,
		Err:           nil,
		ReturnData:    proofBytes,
		ScheduledTxes: nil,
	}
	return result, nil
}

func init() {

	nodeInterface, err := abi.JSON(strings.NewReader(node_interfacegen.NodeInterfaceABI))
	if err != nil {
		panic(err)
	}
	core.InterceptRPCMessage = func(msg Message, statedb *state.StateDB) (Message, *ExecutionResult, error) {
		to := msg.To()
		arbosVersion := arbosState.ArbOSVersion(statedb) // check ArbOS has been installed
		if to == nil || *to != common.HexToAddress("0xc8") || arbosVersion == 0 {
			return msg, nil, nil
		}
		return ApplyNodeInterface(msg, statedb, nodeInterface)
	}

	core.InterceptRPCGasCap = func(gascap *uint64, msg Message, header *types.Header, statedb *state.StateDB) {
		arbosVersion := arbosState.ArbOSVersion(statedb)
		if arbosVersion == 0 {
			// ArbOS hasn't been installed, so use the vanilla gas cap
			return
		}
		state, err := arbosState.OpenSystemArbosState(statedb, true)
		if err != nil {
			log.Error("failed to open ArbOS state", "err", err)
			return
		}
		poster, _ := state.L1PricingState().ReimbursableAggregatorForSender(msg.From())
		if poster == nil || header.BaseFee.Sign() == 0 {
			// if gas is free or there's no reimbursable poster, the user won't pay for L1 data costs
			return
		}
		posterCost, _ := state.L1PricingState().PosterDataCost(msg, msg.From(), *poster)
		posterCostInL2Gas := arbmath.BigToUintSaturating(arbmath.BigDiv(posterCost, header.BaseFee))
		*gascap = arbmath.SaturatingUAdd(*gascap, posterCostInL2Gas)
	}
}
