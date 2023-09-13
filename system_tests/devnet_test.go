// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/util/signature"
)

func shouldSkip(t *testing.T) {
	t.Helper()
	if os.Getenv("DEVNET_TESTS") == "" {
		t.Skip("Skipping Devnet tests")
	}
}

func fileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil
}

func TestNitroDevnet(t *testing.T) {
	shouldSkip(t)

	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(log.LvlTrace)
	log.Root().SetHandler(glogger)

	if os.Getenv("FAUCET_KEY") == "" {
		t.Fatal("No FAUCET_KEY was specified")
	}

	var rollupAddressesPath, l1AccountsPath string
	if os.Getenv("ROLLUP_ADDRESSES_DIR") != "" {
		rollupAddressesPath = os.Getenv("ROLLUP_ADDRESSES_DIR") + "/rollup.json"
		l1AccountsPath = os.Getenv("ROLLUP_ADDRESSES_DIR") + "/l1accounts.json"
	}

	faucetKey, err := crypto.HexToECDSA(os.Getenv("FAUCET_KEY"))
	Require(t, err)

	ctx := context.Background()
	_ = ctx

	l1ChainId := big.NewInt(32382)
	l1info := NewBlockChainTestInfo(t, types.NewCancunSigner(l1ChainId), big.NewInt(params.GWei*100), params.TxGas)

	l1client, err := ethclient.Dial("http://localhost:8545")
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

	rollupAddresses := &chaininfo.RollupAddresses{}
	chainConfig := params.ArbitrumDevTestChainConfig()

	if rollupAddressesPath == "" || !fileExists(rollupAddressesPath) || !fileExists(l1AccountsPath) {
		rollupAddresses, _ = DeployOnTestL1(t, ctx, l1info, l1client, chainConfig)

		if rollupAddressesPath != "" {
			rollupAddressesJson, err := json.MarshalIndent(*rollupAddresses, "", "  ")
			Require(t, err)

			err = ioutil.WriteFile(rollupAddressesPath, rollupAddressesJson, os.ModePerm)
			Require(t, err)

			l1AccountsJson, err := json.MarshalIndent(l1info.Accounts, "", "  ")
			Require(t, err)

			err = ioutil.WriteFile(l1AccountsPath, l1AccountsJson, os.ModePerm)
			Require(t, err)
		}
	} else {
		rollupAddressesFile, err := os.Open(rollupAddressesPath)
		Require(t, err)
		defer rollupAddressesFile.Close()
		rollupAddressesBytes, err := ioutil.ReadAll(rollupAddressesFile)
		Require(t, err)
		err = json.Unmarshal(rollupAddressesBytes, rollupAddresses)
		Require(t, err)

		l1AccountsFile, err := os.Open(l1AccountsPath)
		Require(t, err)
		defer l1AccountsFile.Close()
		l1AccountsBytes, err := ioutil.ReadAll(l1AccountsFile)
		Require(t, err)
		err = json.Unmarshal(l1AccountsBytes, &l1info.Accounts)
		Require(t, err)
	}

	t.Logf("rollupAddresses: %v", rollupAddresses)

	sequencerTxOpts := l1info.GetDefaultTransactOpts("Sequencer", ctx)
	dataSigner := signature.DataSignerFromPrivateKey(l1info.GetInfoWithPrivKey("Sequencer").PrivateKey)
	_, _ = sequencerTxOpts, dataSigner

	nodeConfig := arbnode.ConfigDefaultL1Test()
	nodeConfig.BatchPoster.EIP4844 = true
	nodeConfig.Forwarder.RedisUrl = ""
	nodeConfig.L1Reader.UseFinalityData = true
	l2info, l2stack, l2chainDb, l2arbDb, l2blockchain := createL2BlockChainWithStackConfig(t, nil, "", chainConfig, nil, nil)
	_, _, _, _, _ = l2info, l2stack, l2chainDb, l2arbDb, l2blockchain

	fatalErrChan := make(chan error, 10)
	currentNode, err := arbnode.CreateNode(
		ctx, l2stack, l2chainDb, l2arbDb, NewFetcherFromConfig(nodeConfig), l2blockchain, l1client,
		rollupAddresses, &sequencerTxOpts, &sequencerTxOpts, dataSigner, fatalErrChan,
	)
	Require(t, err)
	_ = currentNode

	Require(t, currentNode.Start(ctx))

	l2client := ClientForStack(t, l2stack)
	_ = l2client

	StartWatchChanErr(t, ctx, fatalErrChan, currentNode)

	/// Second node
	nodeConfigB := arbnode.ConfigDefaultL1NonSequencerTest()
	nodeConfigB.BatchPoster.EIP4844 = true
	nodeConfigB.Forwarder.RedisUrl = ""
	nodeConfigB.L1Reader.UseFinalityData = true
	nodeConfigB.BlobClient.BeaconChainUrl = "http://localhost:3500/"
	l2clientB, nodeB := Create2ndNodeWithConfigAndClient(t, ctx, currentNode, l1client, l1info, &l2info.ArbInitData, nodeConfigB, nil)
	defer nodeB.StopAndWait()

	l2info.GenerateAccount("User1")

	tx := l2info.PrepareTx("Owner", "User1", l2info.TransferGas, big.NewInt(1e12), nil)

	err = l2client.SendTransaction(ctx, tx)
	Require(t, err)

	_, err = EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)

	_, err = EnsureTxSucceeded(ctx, l2clientB, tx)
	Require(t, err)
}
