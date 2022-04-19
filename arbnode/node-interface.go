// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/retryables"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/solgen/go/node_interfacegen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/merkletree"
	"github.com/offchainlabs/nitro/validator"
)

type Message = types.Message
type ExecutionResult = core.ExecutionResult

// To avoid creating new RPC methods for client-side tooling, nitro Geth's InterceptRPCMessage() hook provides
// an opportunity to swap out the message its handling before deriving a transaction from it.
//
// This function handles messages sent to 0xc8 and uses NodeInterface.sol to determine what to do. No contract
// actually exists at 0xc8, but the abi methods allow the incoming message's calldata to specify the arguments.
//
func ApplyNodeInterface(
	msg Message,
	ctx context.Context,
	statedb *state.StateDB,
	backend core.NodeInterfaceBackendAPI,
	nodeInterface abi.ABI,
) (Message, *ExecutionResult, error) {

	estimateMethod := nodeInterface.Methods["estimateRetryableTicket"]
	outboxMethod := nodeInterface.Methods["constructOutboxProof"]
	findBatchMethod := nodeInterface.Methods["findBatchContainingBlock"]
	l1ConfsMethod := nodeInterface.Methods["getL1Confirmations"]

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

		state, _ := arbosState.OpenSystemArbosState(statedb, nil, true)
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
	} else if bytes.Equal(outboxMethod.ID, calldata[:4]) {
		inputs, err := outboxMethod.Inputs.Unpack(calldata[4:])
		if err != nil {
			return msg, nil, err
		}
		size, _ := inputs[0].(uint64)
		leaf, _ := inputs[1].(uint64)

		if leaf >= size {
			return msg, nil, fmt.Errorf("leaf %v is newer than root of size %v", leaf, size)
		}

		res, err := nodeInterfaceConstructOutboxProof(msg, ctx, size, leaf, backend, outboxMethod)
		return msg, res, err
	} else if bytes.Equal(findBatchMethod.ID, calldata[:4]) {
		inputs, err := findBatchMethod.Inputs.Unpack(calldata[4:])
		if err != nil {
			return msg, nil, err
		}
		block, _ := inputs[0].(uint64)

		node, err := arbNodeFromNodeInterfaceBackend(backend)
		if err != nil {
			return msg, nil, err
		}
		genesis, err := node.TxStreamer.GetGenesisBlockNumber()
		if err != nil {
			return msg, nil, err
		}
		batch, err := findBatchContainingBlock(node, genesis, block)
		if err != nil {
			return msg, nil, err
		}
		returnData, err := findBatchMethod.Outputs.Pack(batch)
		if err != nil {
			return msg, nil, fmt.Errorf("internal error: failed to encode outputs: %w", err)
		}

		res := &ExecutionResult{
			UsedGas:       0,
			Err:           nil,
			ReturnData:    returnData,
			ScheduledTxes: nil,
		}
		return msg, res, err
	} else if bytes.Equal(l1ConfsMethod.ID, calldata[:4]) {
		inputs, err := l1ConfsMethod.Inputs.Unpack(calldata[4:])
		if err != nil {
			return msg, nil, err
		}
		blockHash, _ := inputs[0].([32]byte)

		node, err := arbNodeFromNodeInterfaceBackend(backend)
		if err != nil {
			return msg, nil, err
		}
		confs, err := getL1Confirmations(node, blockHash)
		if err != nil {
			return msg, nil, err
		}
		returnData, err := l1ConfsMethod.Outputs.Pack(confs)
		if err != nil {
			return msg, nil, fmt.Errorf("internal error: failed to encode outputs: %w", err)
		}

		res := &ExecutionResult{
			UsedGas:       0,
			Err:           nil,
			ReturnData:    returnData,
			ScheduledTxes: nil,
		}
		return msg, res, err
	}

	return msg, nil, errors.New("method does not exist in NodeInterface.sol")
}

func arbNodeFromNodeInterfaceBackend(backend core.NodeInterfaceBackendAPI) (*Node, error) {
	apiBackend, ok := backend.(*arbitrum.APIBackend)
	if !ok {
		return nil, errors.New("API backend isn't Arbitrum")
	}
	arbNode, ok := apiBackend.GetArbitrumNode().(*Node)
	if !ok {
		return nil, errors.New("failed to get Arbitrum Node from backend")
	}
	return arbNode, nil
}

var merkleTopic common.Hash
var withdrawTopic common.Hash

func nodeInterfaceConstructOutboxProof(
	msg Message,
	ctx context.Context,
	size, leaf uint64,
	backend core.NodeInterfaceBackendAPI,
	method abi.Method,
) (*ExecutionResult, error) {

	currentBlock := backend.CurrentBlock()
	currentBlockInfo, err := types.DeserializeHeaderExtraInformation(currentBlock.Header())
	if err != nil {
		return nil, err
	}
	if leaf > currentBlockInfo.SendCount {
		return nil, errors.New("leaf does not exist")
	}

	balanced := size == arbmath.NextPowerOf2(size)/2
	treeLevels := int(arbmath.Log2ceil(size)) // the # of levels in the tree
	proofLevels := treeLevels - 1             // the # of levels where a hash is needed (all but root)
	walkLevels := treeLevels                  // the # of levels we need to consider when building walks
	if balanced {
		walkLevels -= 1 // skip the root
	}

	// find which nodes we'll want in our proof up to a partial
	start := merkletree.NewLevelAndLeaf(0, leaf)
	query := []merkletree.LevelAndLeaf{start} // the nodes we'll query for
	nodes := []merkletree.LevelAndLeaf{}      // the nodes needed (might not be found from query)
	which := uint64(1)                        // which bit to flip & set
	place := leaf                             // where we are in the tree
	for level := 0; level < walkLevels; level++ {
		sibling := place ^ which
		position := merkletree.NewLevelAndLeaf(uint64(level), sibling)

		if sibling < size {
			// the sibling must not be newer than the root
			query = append(query, position)
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

				partial := merkletree.NewLevelAndLeaf(uint64(level), leaf)

				query = append(query, partial)
				partials[partial] = common.Hash{}
			}
			power >>= 1
		}
	}
	sort.Slice(query, func(i, j int) bool {
		return query[i].Leaf < query[j].Leaf
	})

	// collect the logs
	var search func(lo, hi uint64, find []merkletree.LevelAndLeaf)
	var searchLogs []*types.Log
	var searchErr error
	var searchPositions = make(map[common.Hash]struct{})
	for _, item := range query {
		hash := common.BigToHash(item.ToBigInt())
		searchPositions[hash] = struct{}{}
	}
	search = func(lo, hi uint64, find []merkletree.LevelAndLeaf) {

		mid := (lo + hi) / 2

		block, err := backend.BlockByNumber(ctx, rpc.BlockNumber(mid))
		if err != nil {
			searchErr = err
			return
		}

		if lo == hi {
			all, err := backend.GetLogs(ctx, block.Hash())
			if err != nil {
				searchErr = err
				return
			}
			for _, tx := range all {
				for _, log := range tx {
					if log.Address != types.ArbSysAddress {
						// log not produced by ArbOS
						continue
					}

					if log.Topics[0] != merkleTopic && log.Topics[0] != withdrawTopic {
						// log is unrelated
						continue
					}

					position := log.Topics[3]
					if _, ok := searchPositions[position]; ok {
						// ensure log is one we're looking for
						searchLogs = append(searchLogs, log)
					}
				}
			}
			return
		}

		info, err := types.DeserializeHeaderExtraInformation(block.Header())
		if err != nil {
			searchErr = err
			return
		}

		// Figure out which elements are above and below the midpoint
		//   lower includes leaves older than the midpoint
		//   upper includes leaves at least as new as the midpoint
		//   note: while a binary search is possible here, it doesn't change the complexity
		//
		lower := find
		for len(lower) > 0 && lower[len(lower)-1].Leaf >= info.SendCount {
			lower = lower[:len(lower)-1]
		}
		upper := find[len(lower):]

		if len(lower) > 0 {
			search(lo, mid, lower)
		}
		if len(upper) > 0 {
			search(mid+1, hi, upper)
		}
	}

	search(0, currentBlock.NumberU64(), query)

	if searchErr != nil {
		return nil, searchErr
	}

	known := make(map[merkletree.LevelAndLeaf]common.Hash) // all values in the tree we know
	partialsByLevel := make(map[uint64]common.Hash)        // maps for each level the partial it may have
	var minPartialPlace *merkletree.LevelAndLeaf           // the lowest-level partial
	var send common.Hash

	for _, log := range searchLogs {

		hash := log.Topics[2]
		position := log.Topics[3]

		level := new(big.Int).SetBytes(position[:8]).Uint64()
		leafAdded := new(big.Int).SetBytes(position[8:]).Uint64()

		if level == 0 && leafAdded == leaf {
			send = hash
		}

		if level == 0 {
			hash = crypto.Keccak256Hash(hash.Bytes())
		}

		place := merkletree.NewLevelAndLeaf(level, leafAdded)
		known[place] = hash

		if zero, ok := partials[place]; ok {
			if zero != (common.Hash{}) {
				return nil, errors.New("internal error constructing proof: duplicate partial")
			}
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
				return nil, errors.New("internal error constructing proof: bad step in walk")
			}

			left := curr
			right := curr

			if _, ok := partialsByLevel[step.Level]; ok {
				// a partial on the frontier can only appear on the left
				// moving leftward for a level l skips 2^l leaves
				step.Leaf -= 1 << step.Level
				partial, ok := known[step]
				if !ok {
					return nil, errors.New("internal error constructing proof: incomplete frontier")
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
		}
	}

	hashes := make([]common.Hash, len(nodes))
	for i, place := range nodes {
		hash, ok := known[place]
		if !ok {
			return nil, errors.New("internal error constructing proof: incomplete information")
		}
		hashes[i] = hash
	}

	// recover the root and check correctness
	recovery := crypto.Keccak256Hash(send.Bytes())
	recoveryStep := leaf
	for _, hash := range hashes {
		if recoveryStep&1 == 0 {
			recovery = crypto.Keccak256Hash(recovery.Bytes(), hash.Bytes())
		} else {
			recovery = crypto.Keccak256Hash(hash.Bytes(), recovery.Bytes())
		}
		recoveryStep >>= 1
	}
	root := recovery

	proof := merkletree.MerkleProof{
		RootHash:  root, // now resolved
		LeafHash:  crypto.Keccak256Hash(send.Bytes()),
		LeafIndex: leaf,
		Proof:     hashes,
	}
	if !proof.IsCorrect() {
		return nil, errors.New("internal error constructing proof: proof is wrong")
	}

	returnData, err := method.Outputs.Pack(send, root, hashes)
	if err != nil {
		return nil, fmt.Errorf("internal error: failed to encode outputs: %w", err)
	}

	result := &ExecutionResult{
		UsedGas:       0,
		Err:           nil,
		ReturnData:    returnData,
		ScheduledTxes: nil,
	}
	return result, nil
}

var blockInGenesis = errors.New("")
var blockAfterLatestBatch = errors.New("")

func findBatchContainingBlock(node *Node, genesis uint64, block uint64) (uint64, error) {
	if block <= genesis {
		return 0, fmt.Errorf("%wblock %v is part of genesis", blockInGenesis, block)
	}
	pos := arbutil.BlockNumberToMessageCount(block, genesis) - 1
	high, err := node.InboxTracker.GetBatchCount()
	if err != nil {
		return 0, err
	}
	high--
	latestCount, err := node.InboxTracker.GetBatchMessageCount(high)
	if err != nil {
		return 0, err
	}
	latestBlock := arbutil.MessageCountToBlockNumber(latestCount, genesis)
	if int64(block) > latestBlock {
		return 0, fmt.Errorf("%wrequested block %v is after latest on-chain block %v published in batch %v", blockAfterLatestBatch, block, latestBlock, high)
	}

	return validator.FindBatchContainingMessageIndex(node.InboxTracker, pos, high)
}

func getL1Confirmations(node *Node, blockHash common.Hash) (uint64, error) {
	if node.InboxReader == nil {
		return 0, nil
	}
	bc := node.ArbInterface.BlockChain()
	header := bc.GetHeaderByHash(blockHash)
	if header == nil {
		return 0, errors.New("unknown block hash")
	}
	blockNum := header.Number.Uint64()
	genesis, err := node.TxStreamer.GetGenesisBlockNumber()
	if err != nil {
		return 0, err
	}
	batch, err := findBatchContainingBlock(node, genesis, blockNum)
	if err != nil {
		if errors.Is(err, blockInGenesis) {
			batch = 0
		} else if errors.Is(err, blockAfterLatestBatch) {
			return 0, nil
		} else {
			return 0, err
		}
	}
	latestL1Block, latestBatchCount := node.InboxReader.GetLastReadBlockAndBatchCount()
	if latestBatchCount <= batch {
		return 0, nil // batch was reorg'd out?
	}
	meta, err := node.InboxTracker.GetBatchMetadata(batch)
	if err != nil {
		return 0, err
	}
	if latestL1Block < meta.L1Block || arbutil.BlockNumberToMessageCount(blockNum, genesis) > meta.MessageCount {
		return 0, nil
	}
	canonicalHash := bc.GetCanonicalHash(header.Number.Uint64())
	if canonicalHash != header.Hash() {
		return 0, errors.New("block hash is non-canonical")
	}
	confs := (latestL1Block - meta.L1Block) + 1 + node.InboxReader.GetDelayBlocks()
	return confs, nil
}

func init() {
	nodeInterface, err := abi.JSON(strings.NewReader(node_interfacegen.NodeInterfaceABI))
	if err != nil {
		panic(err)
	}
	core.InterceptRPCMessage = func(
		msg Message,
		ctx context.Context,
		statedb *state.StateDB,
		backend core.NodeInterfaceBackendAPI,
	) (Message, *ExecutionResult, error) {
		to := msg.To()
		arbosVersion := arbosState.ArbOSVersion(statedb) // check ArbOS has been installed
		if to == nil || *to != types.NodeInterfaceAddress || arbosVersion == 0 {
			return msg, nil, nil
		}
		return ApplyNodeInterface(msg, ctx, statedb, backend, nodeInterface)
	}

	core.InterceptRPCGasCap = func(gascap *uint64, msg Message, header *types.Header, statedb *state.StateDB) {
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
		poster, _ := state.L1PricingState().ReimbursableAggregatorForSender(msg.From())
		if poster == nil || header.BaseFee.Sign() == 0 {
			// if gas is free or there's no reimbursable poster, the user won't pay for L1 data costs
			return
		}
		posterCost, _ := state.L1PricingState().PosterDataCost(msg, msg.From(), *poster)
		posterCostInL2Gas := arbmath.BigToUintSaturating(arbmath.BigDiv(posterCost, header.BaseFee))
		*gascap = arbmath.SaturatingUAdd(*gascap, posterCostInL2Gas)
	}

	arbSys, err := precompilesgen.ArbSysMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	withdrawTopic = arbSys.Events["L2ToL1Transaction"].ID
	merkleTopic = arbSys.Events["SendMerkleUpdate"].ID
}
