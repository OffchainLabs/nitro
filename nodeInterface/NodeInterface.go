// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package nodeInterface

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/arbos/retryables"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/solgen/go/node_interfacegen"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/merkletree"
)

// To avoid creating new RPC methods for client-side tooling, nitro Geth's InterceptRPCMessage() hook provides
// an opportunity to swap out the message its handling before deriving a transaction from it.
//
// This mechanism handles messages sent to 0xc8 and uses NodeInterface.sol to determine what to do. No contract
// actually exists at 0xc8, but the abi methods allow the incoming message's calldata to specify the arguments.
type NodeInterface struct {
	Address       addr
	backend       core.NodeInterfaceBackendAPI
	context       context.Context
	header        *types.Header
	sourceMessage *core.Message
	returnMessage struct {
		message *core.Message
		changed *bool
	}
}

var merkleTopic common.Hash
var l2ToL1TxTopic common.Hash
var l2ToL1TransactionTopic common.Hash

var blockInGenesis = errors.New("")
var blockAfterLatestBatch = errors.New("")

func (n NodeInterface) NitroGenesisBlock(c ctx) (huge, error) {
	block := n.backend.ChainConfig().ArbitrumChainParams.GenesisBlockNum
	return arbmath.UintToBig(block), nil
}

func (n NodeInterface) FindBatchContainingBlock(c ctx, evm mech, blockNum uint64) (uint64, error) {
	node, err := arbNodeFromNodeInterfaceBackend(n.backend)
	if err != nil {
		return 0, err
	}
	return findBatchContainingBlock(node, node.TxStreamer.GenesisBlockNumber(), blockNum)
}

func (n NodeInterface) GetL1Confirmations(c ctx, evm mech, blockHash bytes32) (uint64, error) {
	node, err := arbNodeFromNodeInterfaceBackend(n.backend)
	if err != nil {
		return 0, err
	}
	if node.InboxReader == nil {
		return 0, nil
	}
	bc, err := blockchainFromNodeInterfaceBackend(n.backend)
	if err != nil {
		return 0, err
	}
	header := bc.GetHeaderByHash(blockHash)
	if header == nil {
		return 0, errors.New("unknown block hash")
	}
	blockNum := header.Number.Uint64()
	genesis := node.TxStreamer.GenesisBlockNumber()
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
	meta, err := node.InboxTracker.GetBatchMetadata(batch)
	if err != nil {
		return 0, err
	}
	if node.L1Reader.IsParentChainArbitrum() {
		parentChainClient := node.L1Reader.Client()
		parentNodeInterface, err := node_interfacegen.NewNodeInterface(types.NodeInterfaceAddress, parentChainClient)
		if err != nil {
			return 0, err
		}
		parentChainBlock, err := parentChainClient.BlockByNumber(n.context, new(big.Int).SetUint64(meta.ParentChainBlock))
		if err != nil {
			// Hide the parent chain RPC error from the client in case it contains sensitive information.
			// Likely though, this error is just "not found" because the block got reorg'd.
			return 0, fmt.Errorf("failed to get parent chain block %v containing batch", meta.ParentChainBlock)
		}
		confs, err := parentNodeInterface.GetL1Confirmations(&bind.CallOpts{Context: n.context}, parentChainBlock.Hash())
		if err != nil {
			log.Warn(
				"Failed to get L1 confirmations from parent chain",
				"blockNumber", meta.ParentChainBlock,
				"blockHash", parentChainBlock.Hash(), "err", err,
			)
			return 0, fmt.Errorf("failed to get L1 confirmations from parent chain for block %v", parentChainBlock.Hash())
		}
		return confs, nil
	}
	latestL1Block, latestBatchCount := node.InboxReader.GetLastReadBlockAndBatchCount()
	if latestBatchCount <= batch {
		return 0, nil // batch was reorg'd out?
	}
	if latestL1Block < meta.ParentChainBlock || arbutil.BlockNumberToMessageCount(blockNum, genesis) > meta.MessageCount {
		return 0, nil
	}
	canonicalHash := bc.GetCanonicalHash(header.Number.Uint64())
	if canonicalHash != header.Hash() {
		return 0, errors.New("block hash is non-canonical")
	}
	confs := (latestL1Block - meta.ParentChainBlock) + 1 + node.InboxReader.GetDelayBlocks()
	return confs, nil
}

func (n NodeInterface) EstimateRetryableTicket(
	c ctx,
	evm mech,
	sender addr,
	deposit huge,
	to addr,
	l2CallValue huge,
	excessFeeRefundAddress addr,
	callValueRefundAddress addr,
	data []byte,
) error {

	var pRetryTo *addr
	if to != (addr{}) {
		pRetryTo = &to
	}

	l1BaseFee, _ := c.State.L1PricingState().PricePerUnit()
	maxSubmissionFee := retryables.RetryableSubmissionFee(len(data), l1BaseFee)

	submitTx := &types.ArbitrumSubmitRetryableTx{
		ChainId:          nil,
		RequestId:        hash{},
		From:             util.RemapL1Address(sender),
		L1BaseFee:        l1BaseFee,
		DepositValue:     deposit,
		GasFeeCap:        n.sourceMessage.GasPrice,
		Gas:              n.sourceMessage.GasLimit,
		RetryTo:          pRetryTo,
		RetryValue:       l2CallValue,
		Beneficiary:      callValueRefundAddress,
		MaxSubmissionFee: maxSubmissionFee,
		FeeRefundAddr:    excessFeeRefundAddress,
		RetryData:        data,
	}

	// ArbitrumSubmitRetryableTx is unsigned so the following won't panic
	msg, err := core.TransactionToMessage(types.NewTx(submitTx), types.NewArbitrumSigner(nil), nil)
	if err != nil {
		return err
	}

	msg.TxRunMode = core.MessageGasEstimationMode
	*n.returnMessage.message = *msg
	*n.returnMessage.changed = true
	return nil
}

func (n NodeInterface) ConstructOutboxProof(c ctx, evm mech, size, leaf uint64) (bytes32, bytes32, []bytes32, error) {

	hash0 := bytes32{}

	currentBlock := n.backend.CurrentBlock()
	currentBlockInfo := types.DeserializeHeaderExtraInformation(currentBlock)
	if leaf > currentBlockInfo.SendCount {
		return hash0, hash0, nil, errors.New("leaf does not exist")
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
	partials := make(map[merkletree.LevelAndLeaf]hash)
	if !balanced {
		power := uint64(1) << proofLevels
		total := uint64(0)
		for level := proofLevels; level >= 0; level-- {

			if (power & size) > 0 { // the partials map to the binary representation of the size

				total += power    // The leaf for a given partial is the sum of the powers
				leaf := total - 1 // of 2 preceding it. It's 1 less since we count from 0

				partial := merkletree.NewLevelAndLeaf(uint64(level), leaf)

				query = append(query, partial)
				partials[partial] = hash0
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
	var searchPositions = make(map[hash]struct{})
	for _, item := range query {
		hash := common.BigToHash(item.ToBigInt())
		searchPositions[hash] = struct{}{}
	}
	search = func(lo, hi uint64, find []merkletree.LevelAndLeaf) {

		mid := (lo + hi) / 2

		block, err := n.backend.BlockByNumber(n.context, rpc.BlockNumber(mid))
		if err != nil {
			searchErr = err
			return
		}

		if lo == hi {
			all, err := n.backend.GetLogs(n.context, block.Hash(), block.NumberU64())
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

					// L2ToL1TransactionEventID is deprecated in upgrade 4, but it should to safe to make this code handle
					// both events ignoring the version.
					// TODO: Remove L2ToL1Transaction handling on next chain reset
					if log.Topics[0] != merkleTopic && log.Topics[0] != l2ToL1TxTopic && log.Topics[0] != l2ToL1TransactionTopic {
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

		info := types.DeserializeHeaderExtraInformation(block.Header())

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

	search(0, currentBlock.Number.Uint64(), query)

	if searchErr != nil {
		return hash0, hash0, nil, searchErr
	}

	known := make(map[merkletree.LevelAndLeaf]hash) // all values in the tree we know
	partialsByLevel := make(map[uint64]hash)        // maps for each level the partial it may have
	var minPartialPlace *merkletree.LevelAndLeaf    // the lowest-level partial
	var send hash

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
			if zero != hash0 {
				return hash0, hash0, nil, errors.New("internal error constructing proof: duplicate partial")
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

		step := *minPartialPlace
		step.Leaf += 1 << step.Level // we start on the min partial's zero-hash sibling
		known[step] = hash0

		for step.Level < uint64(treeLevels) {

			curr, ok := known[step]
			if !ok {
				return hash0, hash0, nil, errors.New("internal error constructing proof: bad step in walk")
			}

			left := curr
			right := curr

			if _, ok := partialsByLevel[step.Level]; ok {
				// a partial on the frontier can only appear on the left
				// moving leftward for a level l skips 2^l leaves
				step.Leaf -= 1 << step.Level
				partial, ok := known[step]
				if !ok {
					err := errors.New("internal error constructing proof: incomplete frontier")
					return hash0, hash0, nil, err
				}
				left = partial
			} else {
				// getting to the next partial means covering its mirror subtree, so go right
				// moving rightward for a level l skips 2^l leaves
				step.Leaf += 1 << step.Level
				known[step] = hash0
				right = hash0
			}

			// move to the parent
			step.Level += 1
			step.Leaf |= 1 << (step.Level - 1)
			known[step] = crypto.Keccak256Hash(left.Bytes(), right.Bytes())
		}
	}

	hashes := make([]hash, len(nodes))
	for i, place := range nodes {
		hash, ok := known[place]
		if !ok {
			return hash0, hash0, nil, errors.New("internal error constructing proof: incomplete information")
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
		return hash0, hash0, nil, errors.New("internal error constructing proof: proof is wrong")
	}

	hashes32 := make([]bytes32, len(hashes))
	for i, hash := range hashes {
		hashes32[i] = hash
	}
	return send, root, hashes32, nil
}

func (n NodeInterface) messageArgs(
	evm mech, value huge, to addr, contractCreation bool, data []byte,
) arbitrum.TransactionArgs {
	msg := n.sourceMessage
	from := msg.From
	gas := msg.GasLimit
	nonce := msg.Nonce
	maxFeePerGas := msg.GasFeeCap
	maxPriorityFeePerGas := msg.GasTipCap
	chainid := evm.ChainConfig().ChainID

	args := arbitrum.TransactionArgs{
		ChainID:              (*hexutil.Big)(chainid),
		From:                 &from,
		Gas:                  (*hexutil.Uint64)(&gas),
		MaxFeePerGas:         (*hexutil.Big)(maxFeePerGas),
		MaxPriorityFeePerGas: (*hexutil.Big)(maxPriorityFeePerGas),
		Value:                (*hexutil.Big)(value),
		Nonce:                (*hexutil.Uint64)(&nonce),
		Data:                 (*hexutil.Bytes)(&data),
	}
	if !contractCreation {
		args.To = &to
	}
	return args
}

func (n NodeInterface) GasEstimateL1Component(
	c ctx, evm mech, value huge, to addr, contractCreation bool, data []byte,
) (uint64, huge, huge, error) {

	// construct a similar message with a random gas limit to avoid underestimating
	args := n.messageArgs(evm, value, to, contractCreation, data)
	randomGas := l1pricing.RandomGas
	args.Gas = (*hexutil.Uint64)(&randomGas)

	// We set the run mode to eth_call mode here because we want an exact estimate, not a padded estimate
	msg, err := args.ToMessage(randomGas, n.header, evm.StateDB.(*state.StateDB), core.MessageEthcallMode)
	if err != nil {
		return 0, nil, nil, err
	}

	pricing := c.State.L1PricingState()
	l1BaseFeeEstimate, err := pricing.PricePerUnit()
	if err != nil {
		return 0, nil, nil, err
	}
	baseFee, err := c.State.L2PricingState().BaseFeeWei()
	if err != nil {
		return 0, nil, nil, err
	}

	// Compute the fee paid for L1 in L2 terms
	//   See in GasChargingHook that this does not induce truncation error
	//
	brotliCompressionLevel, err := c.State.BrotliCompressionLevel()
	if err != nil {
		return 0, nil, nil, fmt.Errorf("failed to get brotli compression level: %w", err)
	}
	feeForL1, _ := pricing.PosterDataCost(msg, l1pricing.BatchPosterAddress, brotliCompressionLevel)
	feeForL1 = arbmath.BigMulByBips(feeForL1, arbos.GasEstimationL1PricePadding)
	gasForL1 := arbmath.BigDiv(feeForL1, baseFee).Uint64()
	return gasForL1, baseFee, l1BaseFeeEstimate, nil
}

func (n NodeInterface) GasEstimateComponents(
	c ctx, evm mech, value huge, to addr, contractCreation bool, data []byte,
) (uint64, uint64, huge, huge, error) {
	if to == types.NodeInterfaceAddress || to == types.NodeInterfaceDebugAddress {
		return 0, 0, nil, nil, errors.New("cannot estimate virtual contract")
	}

	backend, ok := n.backend.(*arbitrum.APIBackend)
	if !ok {
		return 0, 0, nil, nil, errors.New("failed getting API backend")
	}

	context := n.context
	gasCap := backend.RPCGasCap()
	block := rpc.BlockNumberOrHashWithHash(n.header.Hash(), false)
	args := n.messageArgs(evm, value, to, contractCreation, data)

	totalRaw, err := arbitrum.EstimateGas(context, backend, args, block, nil, gasCap)
	if err != nil {
		return 0, 0, nil, nil, err
	}
	total := uint64(totalRaw)

	pricing := c.State.L1PricingState()

	// Setting the gas currently doesn't affect the PosterDataCost,
	// but we do it anyways for accuracy with potential future changes.
	args.Gas = &totalRaw
	msg, err := args.ToMessage(gasCap, n.header, evm.StateDB.(*state.StateDB), core.MessageGasEstimationMode)
	if err != nil {
		return 0, 0, nil, nil, err
	}
	brotliCompressionLevel, err := c.State.BrotliCompressionLevel()
	if err != nil {
		return 0, 0, nil, nil, fmt.Errorf("failed to get brotli compression level: %w", err)
	}
	feeForL1, _ := pricing.PosterDataCost(msg, l1pricing.BatchPosterAddress, brotliCompressionLevel)

	baseFee, err := c.State.L2PricingState().BaseFeeWei()
	if err != nil {
		return 0, 0, nil, nil, err
	}
	l1BaseFeeEstimate, err := pricing.PricePerUnit()
	if err != nil {
		return 0, 0, nil, nil, err
	}

	// Compute the fee paid for L1 in L2 terms
	gasForL1 := arbos.GetPosterGas(c.State, baseFee, core.MessageGasEstimationMode, feeForL1)

	return total, gasForL1, baseFee, l1BaseFeeEstimate, nil
}

func findBatchContainingBlock(node *arbnode.Node, genesis uint64, block uint64) (uint64, error) {
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
		return 0, fmt.Errorf(
			"%wrequested block %v is after latest on-chain block %v published in batch %v",
			blockAfterLatestBatch, block, latestBlock, high,
		)
	}
	return staker.FindBatchContainingMessageIndex(node.InboxTracker, pos, high)
}

func (n NodeInterface) LegacyLookupMessageBatchProof(c ctx, evm mech, batchNum huge, index uint64) (
	proof []bytes32, path huge, l2Sender addr, l1Dest addr, l2Block huge, l1Block huge, timestamp huge, amount huge, calldataForL1 []byte, err error) {

	node, err := arbNodeFromNodeInterfaceBackend(n.backend)
	if err != nil {
		return
	}
	if node.ClassicOutboxRetriever == nil {
		err = errors.New("this node doesnt support classicLookupMessageBatchProof")
		return
	}
	msg, err := node.ClassicOutboxRetriever.GetMsg(batchNum, index)
	if err != nil {
		return
	}
	proof = msg.ProofNodes
	path = msg.PathInt
	data := msg.Data
	if len(data) < 1+6*32 {
		err = errors.New("unexpected L2 to L1 tx result length")
		return
	}
	if data[0] != 0x3 {
		err = errors.New("unexpected type code")
		return
	}
	data = data[1:]
	l2Sender = common.BytesToAddress(data[:32])
	data = data[32:]
	l1Dest = common.BytesToAddress(data[:32])
	data = data[32:]
	l2Block = new(big.Int).SetBytes(data[:32])
	data = data[32:]
	l1Block = new(big.Int).SetBytes(data[:32])
	data = data[32:]
	timestamp = new(big.Int).SetBytes(data[:32])
	data = data[32:]
	amount = new(big.Int).SetBytes(data[:32])
	data = data[32:]
	calldataForL1 = data
	return
}

// L2BlockRangeForL1 fetches the L1 block number of a given l2 block number.
// c ctx and evm mech arguments are not used but supplied to match the precompile function type in NodeInterface contract
func (n NodeInterface) BlockL1Num(c ctx, evm mech, l2BlockNum uint64) (uint64, error) {
	blockHeader, err := n.backend.HeaderByNumber(n.context, rpc.BlockNumber(l2BlockNum))
	if err != nil {
		return 0, err
	}
	if blockHeader == nil {
		return 0, fmt.Errorf("nil header for l2 block: %d", l2BlockNum)
	}
	blockL1Num := types.DeserializeHeaderExtraInformation(blockHeader).L1BlockNumber
	return blockL1Num, nil
}

func (n NodeInterface) matchL2BlockNumWithL1(c ctx, evm mech, l2BlockNum uint64, l1BlockNum uint64) error {
	blockL1Num, err := n.BlockL1Num(c, evm, l2BlockNum)
	if err != nil {
		return fmt.Errorf("failed to get the L1 block number of the L2 block: %v. Error: %w", l2BlockNum, err)
	}
	if blockL1Num != l1BlockNum {
		return fmt.Errorf("no L2 block was found with the given L1 block number. Found L2 block: %v with L1 block number: %v, given L1 block number: %v", l2BlockNum, blockL1Num, l1BlockNum)
	}
	return nil
}

// L2BlockRangeForL1 finds the first and last L2 block numbers that have the given L1 block number
func (n NodeInterface) L2BlockRangeForL1(c ctx, evm mech, l1BlockNum uint64) (uint64, uint64, error) {
	currentBlockNum := n.backend.CurrentBlock().Number.Uint64()
	genesis := n.backend.ChainConfig().ArbitrumChainParams.GenesisBlockNum

	storedMids := map[uint64]uint64{}
	firstL2BlockForL1 := func(target uint64) (uint64, error) {
		low, high := genesis, currentBlockNum
		highBlockL1Num, err := n.BlockL1Num(c, evm, high)
		if err != nil {
			return 0, err
		}
		if highBlockL1Num < target {
			return high + 1, nil
		}
		for low < high {
			mid := arbmath.SaturatingUAdd(low, high) / 2
			if _, ok := storedMids[mid]; !ok {
				midBlockL1Num, err := n.BlockL1Num(c, evm, mid)
				if err != nil {
					return 0, err
				}
				storedMids[mid] = midBlockL1Num
			}
			if storedMids[mid] < target {
				low = mid + 1
			} else {
				high = mid
			}
		}
		return high, nil
	}

	firstBlock, err := firstL2BlockForL1(l1BlockNum)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get the first L2 block with the L1 block: %v. Error: %w", l1BlockNum, err)
	}
	lastBlock, err := firstL2BlockForL1(l1BlockNum + 1)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get the last L2 block with the L1 block: %v. Error: %w", l1BlockNum, err)
	}

	if err := n.matchL2BlockNumWithL1(c, evm, firstBlock, l1BlockNum); err != nil {
		return 0, 0, err
	}
	lastBlock -= 1
	if err = n.matchL2BlockNumWithL1(c, evm, lastBlock, l1BlockNum); err != nil {
		return 0, 0, err
	}
	return firstBlock, lastBlock, nil
}
