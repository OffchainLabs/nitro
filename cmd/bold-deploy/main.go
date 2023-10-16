// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math/big"
	"os"
	"time"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	solimpl "github.com/OffchainLabs/bold/chain-abstraction/sol-implementation"
	"github.com/OffchainLabs/bold/solgen/go/mocksgen"
	rollupgen "github.com/OffchainLabs/bold/solgen/go/rollupgen"
	challenge_testing "github.com/OffchainLabs/bold/testing"

	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/validator/server_common"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/cmd/util"
)

func main() {
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(log.LvlDebug)
	log.Root().SetHandler(glogger)
	log.Info("deploying rollup")

	ctx := context.Background()

	l1conn := flag.String("l1conn", "", "l1 connection")
	l1keystore := flag.String("l1keystore", "", "l1 private key store")
	deployAccount := flag.String("l1DeployAccount", "", "l1 seq account to use (default is first account in keystore)")
	ownerAddressString := flag.String("ownerAddress", "", "the rollup owner's address")
	sequencerAddressString := flag.String("sequencerAddress", "", "the sequencer's address")
	loserEscrowAddressString := flag.String("loserEscrowAddress", "", "the address which half of challenge loser's funds accumulate at")
	wasmmoduleroot := flag.String("wasmmoduleroot", "", "WASM module root hash")
	wasmrootpath := flag.String("wasmrootpath", "", "path to machine folders")
	l1passphrase := flag.String("l1passphrase", "passphrase", "l1 private key file passphrase")
	l1privatekey := flag.String("l1privatekey", "", "l1 private key")
	outfile := flag.String("l1deployment", "deploy.json", "deployment output json file")
	l1ChainIdUint := flag.Uint64("l1chainid", 1337, "L1 chain ID")
	l2ChainConfig := flag.String("l2chainconfig", "l2_chain_config.json", "L2 chain config json file")
	l2ChainName := flag.String("l2chainname", "", "L2 chain name (will be included in chain info output json file)")
	l2ChainInfo := flag.String("l2chaininfo", "l2_chain_info.json", "L2 chain info output json file")
	authorizevalidators := flag.Uint64("authorizevalidators", 0, "Number of validators to preemptively authorize")
	txTimeout := flag.Duration("txtimeout", 10*time.Minute, "Timeout when waiting for a transaction to be included in a block")
	prod := flag.Bool("prod", false, "Whether to configure the rollup for production or testing")
	flag.Parse()
	l1ChainId := new(big.Int).SetUint64(*l1ChainIdUint)

	if *prod {
		if *wasmmoduleroot == "" {
			panic("must specify wasm module root when launching prod chain")
		}
	}
	if *l2ChainName == "" {
		panic("must specify l2 chain name")
	}

	wallet := genericconf.WalletConfig{
		Pathname:   *l1keystore,
		Account:    *deployAccount,
		Password:   *l1passphrase,
		PrivateKey: *l1privatekey,
	}
	l1TransactionOpts, _, err := util.OpenWallet("l1", &wallet, l1ChainId)
	if err != nil {
		flag.Usage()
		log.Error("error reading keystore")
		panic(err)
	}

	l1client, err := ethclient.Dial(*l1conn)
	if err != nil {
		flag.Usage()
		log.Error("error creating l1client")
		panic(err)
	}

	if !common.IsHexAddress(*sequencerAddressString) && len(*sequencerAddressString) > 0 {
		panic("specified sequencer address is invalid")
	}
	if !common.IsHexAddress(*ownerAddressString) {
		panic("please specify a valid rollup owner address")
	}
	if *prod && !common.IsHexAddress(*loserEscrowAddressString) {
		panic("please specify a valid loser escrow address")
	}

	sequencerAddress := common.HexToAddress(*sequencerAddressString)
	ownerAddress := common.HexToAddress(*ownerAddressString)
	loserEscrowAddress := common.HexToAddress(*loserEscrowAddressString)
	if sequencerAddress != (common.Address{}) && ownerAddress != l1TransactionOpts.From {
		panic("cannot specify sequencer address if owner is not deployer")
	}

	var moduleRoot common.Hash
	if *wasmmoduleroot == "" {
		locator, err := server_common.NewMachineLocator(*wasmrootpath)
		if err != nil {
			panic(err)
		}
		moduleRoot = locator.LatestWasmModuleRoot()
	} else {
		moduleRoot = common.HexToHash(*wasmmoduleroot)
	}
	if moduleRoot == (common.Hash{}) {
		panic("wasmModuleRoot not found")
	}

	headerReaderConfig := headerreader.DefaultConfig
	headerReaderConfig.TxTimeout = *txTimeout

	chainConfigJson, err := os.ReadFile(*l2ChainConfig)
	if err != nil {
		panic(fmt.Errorf("failed to read l2 chain config file: %w", err))
	}
	var chainConfig params.ChainConfig
	err = json.Unmarshal(chainConfigJson, &chainConfig)
	if err != nil {
		panic(fmt.Errorf("failed to deserialize chain config: %w", err))
	}

	arbSys, _ := precompilesgen.NewArbSys(types.ArbSysAddress, l1client)
	l1Reader, err := headerreader.New(ctx, l1client, func() *headerreader.Config { return &headerReaderConfig }, arbSys)
	if err != nil {
		panic(fmt.Errorf("failed to create header reader: %w", err))
	}
	l1Reader.Start(ctx)
	defer l1Reader.StopAndWait()

	ensureTxSucceeds := func(tx *types.Transaction) {
		if waitErr := challenge_testing.WaitForTx(ctx, l1Reader.Client(), tx); waitErr != nil {
			panic(err)
		}
		receipt, err := l1Reader.Client().TransactionReceipt(ctx, tx.Hash())
		if err != nil {
			panic(err)
		}
		if receipt.Status != types.ReceiptStatusSuccessful {
			panic("receipt was not successful")
		}
	}
	stakeToken, tx, tokenBindings, err := mocksgen.DeployTestWETH9(
		l1TransactionOpts,
		l1Reader.Client(),
		"Weth",
		"WETH",
	)
	if err != nil {
		panic(err)
	}
	ensureTxSucceeds(tx)
	mintTokens, ok := new(big.Int).SetString("10000", 10)
	if !ok {
		panic("could not set stake token value")
	}
	l1TransactionOpts.Value = mintTokens
	tx, err = tokenBindings.Deposit(l1TransactionOpts)
	if err != nil {
		panic(err)
	}
	ensureTxSucceeds(tx)
	l1TransactionOpts.Value = big.NewInt(0)

	validatorPrivateKey, err := crypto.HexToECDSA("182fecf15bdf909556a0f617a63e05ab22f1493d25a9f1e27c228266c772a890")
	if err != nil {
		panic(err)
	}
	validatorTxOpts, err := bind.NewKeyedTransactorWithChainID(validatorPrivateKey, l1ChainId)
	if err != nil {
		panic(err)
	}

	// We then need to give the validator some funds from the stake token.
	validatorSeedTokens, ok := new(big.Int).SetString("1000", 10)
	if !ok {
		panic("not ok")
	}
	tx, err = tokenBindings.TestWETH9Transactor.Transfer(l1TransactionOpts, validatorTxOpts.From, validatorSeedTokens)
	if err != nil {
		panic(err)
	}
	ensureTxSucceeds(tx)

	miniStake := big.NewInt(1)
	genesisExecutionState := rollupgen.ExecutionState{
		GlobalState:   rollupgen.GlobalState{},
		MachineStatus: 1,
	}
	genesisInboxCount := big.NewInt(0)
	anyTrustFastConfirmer := common.Address{}
	rollupConfig := challenge_testing.GenerateRollupConfig(
		*prod,
		moduleRoot,
		l1TransactionOpts.From,
		chainConfig.ChainID,
		loserEscrowAddress,
		miniStake,
		stakeToken,
		genesisExecutionState,
		genesisInboxCount,
		anyTrustFastConfirmer,
		challenge_testing.WithLayerZeroHeights(&protocol.LayerZeroHeights{
			BlockChallengeHeight:     32,
			BigStepChallengeHeight:   32,
			SmallStepChallengeHeight: 32,
		}),
		challenge_testing.WithNumBigStepLevels(uint8(5)),       // TODO: Hardcoded.
		challenge_testing.WithConfirmPeriodBlocks(uint64(150)), // TODO: Hardcoded.
	)

	deployedAddresses, err := arbnode.DeployBOLDOnL1(
		ctx,
		l1Reader,
		l1TransactionOpts,
		sequencerAddress,
		*authorizevalidators,
		rollupConfig,
	)
	if err != nil {
		flag.Usage()
		log.Error("error deploying on l1")
		panic(err)
	}
	// We then have the validator itself authorize the rollup and challenge manager
	// contracts to spend its stake tokens.
	chain, err := solimpl.NewAssertionChain(
		ctx,
		deployedAddresses.Rollup,
		validatorTxOpts,
		l1Reader.Client(),
	)
	if err != nil {
		panic(err)
	}
	chalManager, err := chain.SpecChallengeManager(ctx)
	if err != nil {
		panic(err)
	}
	amountToApproveSpend, ok := new(big.Int).SetString("10000", 10)
	if !ok {
		panic("not ok")
	}
	tx, err = tokenBindings.TestWETH9Transactor.Approve(validatorTxOpts, deployedAddresses.Rollup, amountToApproveSpend)
	if err != nil {
		panic(err)
	}
	ensureTxSucceeds(tx)
	tx, err = tokenBindings.TestWETH9Transactor.Approve(validatorTxOpts, chalManager.Address(), amountToApproveSpend)
	if err != nil {
		panic(err)
	}
	ensureTxSucceeds(tx)

	deployData, err := json.Marshal(deployedAddresses)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(*outfile, deployData, 0600); err != nil {
		panic(err)
	}
	parentChainIsArbitrum := l1Reader.IsParentChainArbitrum()
	chainsInfo := []chaininfo.ChainInfo{
		{
			ChainName:             *l2ChainName,
			ParentChainId:         l1ChainId.Uint64(),
			ParentChainIsArbitrum: &parentChainIsArbitrum,
			ChainConfig:           &chainConfig,
			RollupAddresses: &chaininfo.RollupAddresses{
				Bridge:                 deployedAddresses.Bridge,
				Inbox:                  deployedAddresses.Inbox,
				SequencerInbox:         deployedAddresses.SequencerInbox,
				Rollup:                 deployedAddresses.Rollup,
				ValidatorUtils:         deployedAddresses.ValidatorUtils,
				ValidatorWalletCreator: deployedAddresses.ValidatorWalletCreator,
				DeployedAt:             deployedAddresses.DeployedAt,
			},
		},
	}
	chainsInfoJson, err := json.Marshal(chainsInfo)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(*l2ChainInfo, chainsInfoJson, 0600); err != nil {
		panic(err)
	}
}
