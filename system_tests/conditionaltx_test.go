package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestSendRawTransactionConditional(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l2info, _, l2client, l2stack, _, _, _, l1stack := createTestNodeOnL1WithConfigImpl(t, ctx, true, nil, nil, nil)
	defer requireClose(t, l1stack)
	defer requireClose(t, l2stack)

	rpcClient, err := l2stack.Attach()

	l2info.GenerateAccount("User2")

	tx := l2info.PrepareTx("Owner", "User2", l2info.TransferGas, big.NewInt(1e12), nil)
	err = l2client.SendTransaction(ctx, tx)
	testhelpers.RequireImpl(t, err)
	receipt, err := WaitForTx(ctx, l2client, tx.Hash(), time.Second*5)
	testhelpers.RequireImpl(t, err)

	tx = l2info.PrepareTx("Owner", "User2", l2info.TransferGas, big.NewInt(1e12), nil)
	options := arbitrum.ConditionalOptions{KnownAccounts: map[common.Address]arbitrum.RootHashOrSlots{l2info.GetAddress("User2"): arbitrum.RootHashOrSlots{RootHash: &common.Hash{0}}}}

	err = arbitrum.SendConditionalTransactionRPC(ctx, rpcClient, tx, &options)
	testhelpers.RequireImpl(t, err)
	receipt, err = WaitForTx(ctx, l2client, tx.Hash(), time.Second*5)
	testhelpers.RequireImpl(t, err)
	if receipt.Status != types.ReceiptStatusSuccessful {
		testhelpers.FailImpl(t)
	}
}
