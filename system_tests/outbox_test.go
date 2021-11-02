//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbtest

import (
	"context"
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/arbstate/arbstate"
	"github.com/offchainlabs/arbstate/solgen/go/precompilesgen"
	"github.com/offchainlabs/arbstate/util/merkletree"
)

func TestOutboxProofs(t *testing.T) {
	arbstate.RequireHookedGeth()

	arbSysAbi, err := precompilesgen.ArbSysMetaData.GetAbi()
	failOnError(t, err, "failed to get abi")
	withdrawTopic := arbSysAbi.Events["L2ToL1Transaction"].ID
	merkleTopic := arbSysAbi.Events["SendMerkleUpdate"].ID

	backend, l2info := CreateTestL2(t)
	client := ClientForArbBackend(t, backend)
	arbSys, err := precompilesgen.NewArbSys(common.HexToAddress("0x65"), client)
	if err != nil {
		t.Fatal(err)
	}
	ownerOps := l2info.GetDefaultTransactOpts("Owner")

	ctx := context.Background()
	txnCount := int64(2 + rand.Intn(1024))

	// represents a send we should be able to prove exists
	type proofPair struct {
		hash common.Hash
		leaf uint64
	}

	provables := make([]proofPair, 0)

	for i := int64(0); i < txnCount; i++ {
		ownerOps.Value = big.NewInt(i * 1000000000)
		tx, err := arbSys.WithdrawEth(&ownerOps, common.Address{})
		failOnError(t, err, "ArbSys failed")
		receipt, err := client.TransactionReceipt(ctx, tx.Hash())
		failOnError(t, err, "No receipt for txn")

		for _, log := range receipt.Logs {
			if log.Topics[0] == withdrawTopic {
				parsedLog, err := arbSys.ParseL2ToL1Transaction(*log)
				failOnError(t, err, "Failed to parse log")

				provables = append(provables, proofPair{
					hash: common.BigToHash(parsedLog.Hash),
					leaf: parsedLog.Position.Uint64(),
				})
			}
		}
	}

	merkleState, err := arbSys.SendMerkleTreeState(&bind.CallOpts{})
	failOnError(t, err, "could not get merkle root")
	rootHash := merkleState.Root // we assume the user knows the root and size

	// using only the root and position, we'll prove the send hash exists for each node
	for _, provable := range provables {
		proof := merkletree.MerkleProof{
			RootHash:  rootHash,
			LeafHash:  provable.hash,
			LeafIndex: provable.leaf,
			Proof:     []common.Hash{},
		}

		if !proof.IsCorrect() {
			t.Fatal("Proof is wrong")
		}
	}

	_ = merkleTopic
}

func failOnError(t *testing.T, err error, msg string) {
	if err != nil {
		t.Fatal(msg+":", err)
	}
}
