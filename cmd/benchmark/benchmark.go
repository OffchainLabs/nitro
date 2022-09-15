// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/nodeInterface"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/solgen/go/node_interfacegen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/statetransfer"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/colors"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/validator"
)

func main() {
	args := os.Args[1:]
	machinePath := "./target/machines/"
	if len(args) > 0 {
		machinePath = args[0]
		if _, err := os.Stat(machinePath); err != nil {
			panic(fmt.Sprintf("%v%v%v", colors.Red, err, colors.Clear))
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	arbstate.RequireHookedGeth()
	nodeInterface.RequireVirtualContracts()

	nodeConfig := arbnode.ConfigDefaultL1Test()
	chainConfig := params.ArbitrumDevTestChainConfig()
	largeBalance := arbmath.UintToBig(1e19)
	l1ChainId := big.NewInt(1337)
	l2ChainId := chainConfig.ChainID

	authKey, authAddr := keypair(0)
	sequencerKey, sequencerAddr := keypair(1)
	colors.PrintBlue("Auth: ", authAddr)
	colors.PrintBlue("Sequencer: ", sequencerAddr)

	l1Auth, err := bind.NewKeyedTransactorWithChainID(authKey, l1ChainId)
	Require(err)
	l2Auth, err := bind.NewKeyedTransactorWithChainID(authKey, chainConfig.ChainID)
	Require(err)
	l1Auth.Context = ctx
	l2Auth.Context = ctx
	l2Auth.GasLimit = 2 * l2pricing.InitialPerBlockGasLimitV6 // fill the block

	tempDir, err := os.MkdirTemp("", "nitro-benchmark-")
	Require(err)
	defer os.RemoveAll(tempDir)

	stackConf := node.DefaultConfig
	stackConf.HTTPPort = 0
	stackConf.WSPort = 0
	stackConf.UseLightweightKDF = true
	stackConf.P2P.ListenAddr = ""
	stackConf.P2P.NoDial = true
	stackConf.P2P.NoDiscovery = true
	stackConf.P2P.NAT = nil
	stackConf.DataDir = tempDir
	l1Stack, err := node.New(&stackConf)
	Require(err)

	ethNodeConf := ethconfig.Defaults
	ethNodeConf.NetworkId = chainConfig.ChainID.Uint64()
	l1Genesis := core.DeveloperGenesisBlock(0, 15_000_000, authAddr)
	l1Genesis.Alloc[authAddr] = core.GenesisAccount{
		Balance: largeBalance,
		Nonce:   0,
	}
	l1Genesis.Alloc[sequencerAddr] = core.GenesisAccount{
		Balance: largeBalance,
		Nonce:   0,
	}
	l1Genesis.BaseFee = big.NewInt(l2pricing.InitialBaseFeeWei)
	ethNodeConf.Genesis = l1Genesis
	ethNodeConf.Miner.Etherbase = authAddr

	l1backend, err := eth.New(l1Stack, &ethNodeConf)
	Require(err)
	tempKeyStore := keystore.NewPlaintextKeyStore(tempDir)
	faucetAccount, err := tempKeyStore.ImportECDSA(authKey, "passphrase")
	Require(err)
	Require(tempKeyStore.Unlock(faucetAccount, "passphrase"))
	l1backend.AccountManager().AddBackend(tempKeyStore)
	l1backend.SetEtherbase(authAddr)

	Require(l1Stack.Start())
	Require(l1backend.StartMining(1))
	l1rpcClient, err := l1Stack.Attach()
	Require(err)
	l1client := ethclient.NewClient(l1rpcClient)

	fatalErrChan := make(chan error, 10)
	go func() {
		err := <-fatalErrChan
		panic(fmt.Sprintf("Encountered fatal error: %v", err))
	}()

	cranelift := false
	machineConfig := validator.DefaultNitroMachineConfig
	machineConfig.RootPath = machinePath
	machineConfig.JitCranelift = cranelift
	machineLoader := validator.NewNitroMachineLoader(machineConfig, fatalErrChan)
	moduleRoot, err := machineConfig.ReadLatestWasmModuleRoot()
	Require(err)

	headerReaderConfig := headerreader.TestConfig
	headerReaderConfig.OldHeaderTimeout = 20 * time.Minute
	headerReaderConfigFetcher := func() *headerreader.Config { return &headerReaderConfig }

	addresses, err := arbnode.DeployOnL1(
		ctx,
		l1client,
		l1Auth,
		sequencerAddr,
		0,
		headerReaderConfigFetcher,
		machineConfig,
		arbnode.GenerateRollupConfig(false, moduleRoot, l1Auth.From, l2ChainId, common.Address{}),
	)
	Require(err)

	l2Stack, err := arbnode.CreateDefaultStackForTest("")
	Require(err)
	l2ChainDb, err := l2Stack.OpenDatabase("chaindb", 0, 0, "", false)
	Require(err)
	l2ArbDb, err := l2Stack.OpenDatabase("arbdb", 0, 0, "", false)
	Require(err)
	initReader := statetransfer.NewMemoryInitDataReader(&statetransfer.ArbosInitializationInfo{
		Accounts: []statetransfer.AccountInitializationInfo{
			{Addr: authAddr, EthBalance: largeBalance},
			{Addr: sequencerAddr, EthBalance: largeBalance},
		},
	})
	l2Blockchain, err := arbnode.WriteOrTestBlockChain(
		l2ChainDb, nil, initReader, chainConfig, arbnode.ConfigDefaultL2Test(), 0,
	)
	Require(err)

	sequencerOpts, err := bind.NewKeyedTransactorWithChainID(sequencerKey, l1ChainId)
	Require(err)
	feedErrChan := make(chan error, 10)
	node, err := arbnode.CreateNode(
		ctx, l2Stack, l2ChainDb, l2ArbDb, nodeConfig, l2Blockchain,
		l1client, addresses, sequencerOpts, nil, feedErrChan,
	)
	Require(err)

	Require(l2Stack.Start())

	l2rpcClient, err := l2Stack.Attach()
	Require(err)
	l2client := ethclient.NewClient(l2rpcClient)

	redo := func(message string, lambda func() bool) {
		for i := 0; i < 16; i++ {
			done := lambda()
			if !done {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			return
		}
		panic(message)
	}

	waitForTx := func(tx *types.Transaction, err error, expectation uint64) *types.Receipt {
		Require(err)
		var receipt *types.Receipt
		redo("failed to get reciept", func() bool {
			receipt, err = l2client.TransactionReceipt(ctx, tx.Hash())
			if err == nil && receipt.Status != expectation {
				panic("unexpected tx result")
			}
			return err == nil
		})
		return receipt
	}

	_, tx, simple, err := mocksgen.DeploySimple(l2Auth, l2client)
	waitForTx(tx, err, types.ReceiptStatusSuccessful)
	nodeInterface, err := node_interfacegen.NewNodeInterface(types.NodeInterfaceAddress, l2client)
	waitForTx(tx, err, types.ReceiptStatusSuccessful)
	ArbGasInfo, err := precompilesgen.NewArbGasInfo(types.ArbGasInfoAddress, l2client)
	waitForTx(tx, err, types.ReceiptStatusSuccessful)

	tx, err = simple.Exhaust(l2Auth)
	receipt := waitForTx(tx, err, types.ReceiptStatusFailed)

	speedLimit, _, _, err := ArbGasInfo.GetGasAccountingParams(&bind.CallOpts{})
	Require(err)

	redo("failed to confirm on L1", func() bool {
		confs, err := nodeInterface.GetL1Confirmations(&bind.CallOpts{}, receipt.BlockHash)
		Require(err)
		return confs != 0
	})

	validatorConf := validator.DefaultBlockValidatorConfig
	validatorConf.Enable = false
	validatorConf.ArbitratorValidator = false
	validatorConf.JitValidator = false
	validatorConf.JitValidatorCranelift = cranelift

	prover, err := validator.NewStatelessBlockValidator(
		machineLoader,
		node.InboxReader,
		node.InboxTracker,
		node.TxStreamer,
		l2Blockchain,
		rawdb.NewMemoryDatabase(),
		nil,
		&validatorConf,
		fatalErrChan,
	)
	Require(err)

	header := l2Blockchain.GetHeaderByHash(receipt.BlockHash)
	intrinsic, err := core.IntrinsicGas(tx.Data(), tx.AccessList(), false, true, true)
	Require(err)
	gasUsed := receipt.GasUsed - receipt.GasUsedForL1 - intrinsic

	measure := func(kind string, jit bool) {
		start := time.Now()
		if jit {
			_, err = prover.MachineLoader.GetJitMachine(ctx, moduleRoot, true)
		} else {
			_, err = prover.MachineLoader.GetMachine(ctx, moduleRoot, true)
		}
		if err != nil {
			panic(fmt.Sprintf("Failed to create %v machine: %v", kind, err))
		}
		fmt.Printf(
			"Created %v machine in %v\n",
			colors.PinkStr("%v", kind),
			colors.PinkStr("%v", time.Since(start)),
		)

		start = time.Now()
		valid, err := prover.ValidateBlock(ctx, header, !jit, moduleRoot)
		Require(err)
		if !valid {
			panic(fmt.Sprintf("%v validation failed", kind))
		}
		delay := time.Since(start)

		fmt.Printf(
			"Validated block of %v gas with %v in %v\n",
			colors.PinkStr("%v", gasUsed),
			colors.PinkStr("%v", kind),
			colors.PinkStr("%v", delay),
		)
		gasPerSecond := float64(gasUsed) / delay.Seconds()
		coresNeeded := float64(speedLimit.Uint64()) / gasPerSecond

		fmt.Printf(
			"Validated @ %v gas/s,\nso %v cores are needed for a %v gas/s speed limit\n\n",
			colors.PinkStr("%.2f", gasPerSecond),
			colors.PinkStr("%.2f", coresNeeded),
			colors.PinkStr("%v", speedLimit),
		)
	}

	measure("JIT", true)
	measure("Arbitrator", false)
}

func keypair(seed byte) (*ecdsa.PrivateKey, common.Address) {
	source := make([]byte, 128)
	source[0] = seed
	key, err := ecdsa.GenerateKey(crypto.S256(), bytes.NewReader(source))
	Require(err)
	addr := crypto.PubkeyToAddress(key.PublicKey)
	return key, addr
}

func Require(err error) {
	if err != nil {
		panic(fmt.Sprintf("%v%v%v", colors.Red, err, colors.Clear))
	}
}
