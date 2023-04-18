// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/util/signature"
)

func shouldSkip(t *testing.T) {
	t.Helper()
	if os.Getenv("DEVNET_TESTS") == "" {
		t.Skip("Skipping Devnet tests")
	}
}

func TestNitroDevnet(t *testing.T) {
	shouldSkip(t)

	if os.Getenv("FAUCET_KEY") == "" {
		t.Fatal("No FAUCET_KEY was specified")
	}

	faucetKey, err := crypto.HexToECDSA(os.Getenv("FAUCET_KEY"))
	Require(t, err)

	ctx := context.Background()
	_ = ctx

	l1ChainId := big.NewInt(32382)
	l1info := NewBlockChainTestInfo(t, types.NewLondonSigner(l1ChainId), big.NewInt(params.GWei*100), params.TxGas)

	l1client, err := ethclient.Dial("ws://localhost:8546")
	Require(t, err)

	faucetAddress := crypto.PubkeyToAddress(faucetKey.PublicKey)

	faucetNonce, err := l1client.NonceAt(ctx, faucetAddress, nil)
	Require(t, err)

	t.Logf("Faucet nonce is %d", faucetNonce)

	// This is the faucet account that is configured in the devnet's genesis
	faucetAccount := AccountInfo{
		Address:    faucetAddress,
		PrivateKey: faucetKey,
		Nonce:      faucetNonce,
	}

	l1info.SetFullAccountInfo("Faucet", &faucetAccount)

	rollupAddresses := DeployOnTestL1(t, ctx, l1info, l1client, big.NewInt(412346))
	_ = rollupAddresses
	t.Logf("rollupAddresses: %v", rollupAddresses)

	sequencerTxOpts := l1info.GetDefaultTransactOpts("Sequencer", ctx)
	dataSigner := signature.DataSignerFromPrivateKey(l1info.GetInfoWithPrivKey("Sequencer").PrivateKey)

	nodeConfig := arbnode.ConfigDefaultL1Test()
	chainConfig := params.ArbitrumDevTestChainConfig()
	l2info, l2stack, l2chainDb, l2arbDb, l2blockchain := createL2BlockChainWithStackConfig(t, nil, "", chainConfig, nil)
	_ = l2info

	fatalErrChan := make(chan error, 10)
	currentNode, err := arbnode.CreateNode(
		ctx, l2stack, l2chainDb, l2arbDb, nodeConfig, l2blockchain, l1client,
		rollupAddresses, &sequencerTxOpts, dataSigner, fatalErrChan,
	)
	Require(t, err)

	Require(t, currentNode.Start(ctx))

	l2client := ClientForStack(t, l2stack)
	_ = l2client

	StartWatchChanErr(t, ctx, fatalErrChan, currentNode)

}
