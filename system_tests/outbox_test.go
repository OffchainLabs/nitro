// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"encoding/hex"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/gethhook"
	"github.com/offchainlabs/nitro/solgen/go/node_interfacegen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/merkletree"
)

func TestP256VerifyEnabled(t *testing.T) {
	gethhook.RequireHookedGeth()
	for _, tc := range []struct {
		arbOSVersion   uint64
		wantP256Verify bool
	}{
		{
			arbOSVersion:   20,
			wantP256Verify: false,
		},
		{
			arbOSVersion:   30,
			wantP256Verify: true,
		},
	} {
		addresses := arbosState.GetArbitrumOnlyGenesisPrecompiles(&params.ChainConfig{
			ArbitrumChainParams: params.ArbitrumChainParams{
				EnableArbOS:         true,
				InitialArbOSVersion: tc.arbOSVersion,
			},
		})
		got := false
		for _, a := range addresses {
			got = got || (a == common.BytesToAddress([]byte{0x01, 0x00}))
		}
		if got != tc.wantP256Verify {
			t.Errorf("Got P256Verify enabled: %t, want: %t", got, tc.wantP256Verify)
		}
	}
}

func TestOutboxProofs(t *testing.T) {
	t.Parallel()
	gethhook.RequireHookedGeth()
	rand.Seed(time.Now().UTC().UnixNano())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	arbSysAbi, err := precompilesgen.ArbSysMetaData.GetAbi()
	Require(t, err, "failed to get abi")
	withdrawTopic := arbSysAbi.Events["L2ToL1Tx"].ID
	merkleTopic := arbSysAbi.Events["SendMerkleUpdate"].ID

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)

	arbSys, err := precompilesgen.NewArbSys(types.ArbSysAddress, builder.L2.Client)
	Require(t, err)
	nodeInterface, err := node_interfacegen.NewNodeInterface(types.NodeInterfaceAddress, builder.L2.Client)
	Require(t, err)

	txnCount := int64(1 + rand.Intn(16))

	// represents a send we should be able to prove exists
	type proofPair struct {
		hash common.Hash
		leaf uint64
	}

	// represents a historical root we'll prove against
	type proofRoot struct {
		root common.Hash
		size uint64
	}

	provables := make([]proofPair, 0)
	roots := make([]proofRoot, 0)
	txns := []common.Hash{}

	for i := int64(0); i < txnCount; i++ {
		auth.Value = big.NewInt(i * 1000000000)
		auth.Nonce = big.NewInt(i + 1)
		tx, err := arbSys.WithdrawEth(&auth, common.Address{})
		Require(t, err, "ArbSys failed")
		txns = append(txns, tx.Hash())

		time.Sleep(4 * time.Millisecond) // Geth takes a few ms for the receipt to show up
		_, err = builder.L2.Client.TransactionReceipt(ctx, tx.Hash())
		if err == nil {
			merkleState, err := arbSys.SendMerkleTreeState(&bind.CallOpts{})
			Require(t, err, "could not get merkle root")

			root := proofRoot{
				root: merkleState.Root,          // we assume the user knows the root and size
				size: merkleState.Size.Uint64(), //
			}
			roots = append(roots, root)
		}
	}

	for _, tx := range txns {
		var receipt *types.Receipt
		receipt, err = builder.L2.Client.TransactionReceipt(ctx, tx)
		Require(t, err, "No receipt for txn")

		if receipt.Status != types.ReceiptStatusSuccessful {
			Fatal(t, "Tx failed with status code:", receipt)
		}
		if len(receipt.Logs) == 0 {
			Fatal(t, "Tx didn't emit any logs")
		}

		for _, log := range receipt.Logs {

			if log.Topics[0] == withdrawTopic {
				parsedLog, err := arbSys.ParseL2ToL1Tx(*log)
				Require(t, err, "Failed to parse log")

				provables = append(provables, proofPair{
					hash: common.BigToHash(parsedLog.Hash),
					leaf: parsedLog.Position.Uint64(),
				})
			}
		}
	}

	t.Log("Proving against", len(roots), "historical roots among the", txnCount, "ever")
	t.Log("Will query against topics\n\tmerkle:   ", merkleTopic, "\n\twithdraw: ", withdrawTopic)

	for _, root := range roots {
		rootHash := root.root
		treeSize := root.size

		balanced := treeSize == arbmath.NextPowerOf2(treeSize)/2
		treeLevels := int(arbmath.Log2ceil(treeSize)) // the # of levels in the tree
		proofLevels := treeLevels - 1                 // the # of levels where a hash is needed (all but root)
		walkLevels := treeLevels                      // the # of levels we need to consider when building walks
		if balanced {
			walkLevels -= 1 // skip the root
		}

		t.Log("Tree has", treeSize, "leaves and", treeLevels, "levels")
		t.Log("Root hash", hex.EncodeToString(rootHash[:]))
		t.Log("Balanced:", balanced)

		// using only the root and position, we'll prove the send hash exists for each leaf
		for _, provable := range provables {
			if provable.leaf >= treeSize {
				continue
			}

			t.Log("Proving leaf", provable.leaf)

			// find which nodes we'll want in our proof up to a partial
			query := make([]common.Hash, 0)             // the nodes we'll query for
			nodes := make([]merkletree.LevelAndLeaf, 0) // the nodes needed (might not be found from query)
			which := uint64(1)                          // which bit to flip & set
			place := provable.leaf                      // where we are in the tree
			for level := 0; level < walkLevels; level++ {
				sibling := place ^ which

				position := merkletree.LevelAndLeaf{
					Level: uint64(level),
					Leaf:  sibling,
				}

				if sibling < treeSize {
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

					if (power & treeSize) > 0 { // the partials map to the binary representation of the tree size

						total += power    // The actual leaf for a given partial is the sum of the powers of 2
						leaf := total - 1 // preceding it. We subtract 1 since we count from 0

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
			t.Log("Found", len(partials), "partials")

			// in one lookup, query geth for all the data we need to construct a proof
			var logs []types.Log
			if len(query) > 0 {
				logs, err = builder.L2.Client.FilterLogs(ctx, ethereum.FilterQuery{
					Addresses: []common.Address{
						types.ArbSysAddress,
					},
					Topics: [][]common.Hash{
						{merkleTopic, withdrawTopic},
						nil,
						nil,
						query,
					},
				})
				Require(t, err, "couldn't get logs")
			}

			t.Log("Querried for", len(query), "positions", query)
			t.Log("Found", len(logs), "logs for proof", provable.leaf, "of", treeSize)

			known := make(map[merkletree.LevelAndLeaf]common.Hash) // all values in the tree we know
			partialsByLevel := make(map[uint64]common.Hash)        // maps for each level the partial it may have
			var minPartialPlace *merkletree.LevelAndLeaf           // the lowest-level partial

			for _, log := range logs {

				hash := log.Topics[2]
				position := log.Topics[3]

				level := new(big.Int).SetBytes(position[:8]).Uint64()
				leaf := new(big.Int).SetBytes(position[8:]).Uint64()

				if level == 0 {
					hash = crypto.Keccak256Hash(hash.Bytes())
				}

				place := merkletree.LevelAndLeaf{
					Level: level,
					Leaf:  leaf,
				}

				t.Log("Log:\n\tposition: level", level, "leaf", leaf, "\n\thash:    ", hash)
				known[place] = hash

				if zero, ok := partials[place]; ok {
					if zero != (common.Hash{}) {
						Fatal(t, "Somehow got 2 partials for the same level\n\t1st:", zero, "\n\t2nd:", hash)
					}
					partials[place] = hash
					partialsByLevel[level] = hash
					if minPartialPlace == nil || level < minPartialPlace.Level {
						minPartialPlace = &place
					}
				}
			}

			for place, hash := range known {
				t.Log("known  ", place.Level, hash, "@", place)
			}
			t.Log(len(known), "values are known\n")

			for place, hash := range partials {
				t.Log("partial", place.Level, hash, "@", place)
			}
			t.Log("resolving frontiers\n")

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
						Fatal(t, "We should know the current node's value")
					}

					left := curr
					right := curr

					if _, ok := partialsByLevel[step.Level]; ok {
						// a partial on the frontier can only appear on the left
						// moving leftward for a level l skips 2^l leaves
						step.Leaf -= 1 << step.Level
						partial, ok := known[step]
						if !ok {
							Fatal(t, "There should be a partial here")
						}
						left = partial
					} else {
						// getting to the next partial means covering its mirror subtree, so we look right
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

				if known[step] != rootHash {
					// a correct walk of the frontier should end with resolving the root
					t.Log("Walking up the tree didn't re-create the root", known[step], "vs", rootHash)
				}

				for place, hash := range known {
					t.Log("known", place, hash)
				}
			}

			t.Log("Complete proof of leaf", provable.leaf)

			hashes := make([]common.Hash, len(nodes))
			for i, place := range nodes {
				hash, ok := known[place]
				if !ok {
					Fatal(t, "We're missing data for the node at position", place)
				}
				hashes[i] = hash
				t.Log("node", place, hash)
			}

			proof := merkletree.MerkleProof{
				RootHash:  rootHash,
				LeafHash:  crypto.Keccak256Hash(provable.hash.Bytes()),
				LeafIndex: provable.leaf,
				Proof:     hashes,
			}

			if !proof.IsCorrect() {
				Fatal(t, "Proof is wrong")
			}

			// Check NodeInterface.sol produces equivalent proofs
			outboxProof, err := nodeInterface.ConstructOutboxProof(
				&bind.CallOpts{}, treeSize, provable.leaf,
			)
			Require(t, err, "failed to construct outbox proof using NodeInterface.sol")
			nodeRoot := common.Hash(outboxProof.Root)
			nodeProof := outboxProof.Proof
			nodeSend := outboxProof.Send

			if nodeRoot != rootHash {
				Fatal(t, "NodeInterface root differs\n", nodeRoot, "\n", rootHash)
			}
			if len(hashes) != len(nodeProof) {
				Fatal(t, "NodeInterface proof is the wrong size", len(nodeProof), len(hashes))
			}
			for i, correct := range hashes {
				if nodeProof[i] != correct {
					t.Error("NodeInterface proof differs", i, correct, nodeProof[i])
				}
			}
			if nodeSend != provable.hash {
				Fatal(t, "NodeInterface send differs\n", nodeSend, "\n", provable.hash)
			}
		}
	}
}
