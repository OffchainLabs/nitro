// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/nitro/solgen/go/node_interfacegen"
)

func getL1Confirmations(
	ctx context.Context,
	nodeInterface *node_interfacegen.NodeInterface,
	builder *NodeBuilder,
	block *types.Block,
) (uint64, uint64, error) {
	l1ConfsNodeInterface, err := nodeInterface.GetL1Confirmations(&bind.CallOpts{}, block.Hash())
	if err != nil {
		return 0, 0, err
	}

	var l1ConfsRPC uint64
	err = builder.L2.Client.Client().CallContext(ctx, &l1ConfsRPC, "arb_getL1Confirmations", block.Number())

	return l1ConfsNodeInterface, l1ConfsRPC, err
}

func TestGetL1Confirmations(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	cleanup := builder.Build(t)
	defer cleanup()

	// Wait so ConsensusNode.L1Reader has some time to read L1 headers,
	// which is needed for the RPC GetL1Confirmations call to work.
	time.Sleep(time.Second)

	nodeInterface, err := node_interfacegen.NewNodeInterface(types.NodeInterfaceAddress, builder.L2.Client)
	Require(t, err)

	genesisBlock, err := builder.L2.Client.BlockByNumber(ctx, big.NewInt(0))
	Require(t, err)

	l1ConfsNodeInterface, l1ConfsRPC, err := getL1Confirmations(ctx, nodeInterface, builder, genesisBlock)
	Require(t, err)

	numTransactions := 200

	// #nosec G115
	if l1ConfsNodeInterface >= uint64(numTransactions) || l1ConfsRPC >= uint64(numTransactions) {
		t.Fatalf("L1Confirmations for latest block %v is already l1ConfsNodeInterface=%v, l1ConfsRPC=%v, which is over %v",
			genesisBlock.Number(), l1ConfsNodeInterface, l1ConfsRPC, numTransactions)
	}

	for i := 0; i < numTransactions; i++ {
		builder.L1.TransferBalance(t, "User", "User", common.Big0, builder.L1Info)
	}

	l1ConfsNodeInterface, l1ConfsRPC, err = getL1Confirmations(ctx, nodeInterface, builder, genesisBlock)
	Require(t, err)

	// Allow a gap of 10 for asynchronicity, just in case
	// #nosec G115
	if (l1ConfsNodeInterface+10 < uint64(numTransactions)) || (l1ConfsRPC+10 < uint64(numTransactions)) {
		t.Fatalf("L1Confirmations for latest block %v is only l1ConfsNodeInterface=%v, l1ConfsRPC=%v (did not hit expected %v)",
			genesisBlock.Number(), l1ConfsNodeInterface, l1ConfsRPC, numTransactions)
	}
}
