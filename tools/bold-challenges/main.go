package main

import (
	"context"
	"flag"
	"math/big"

	solimpl "github.com/OffchainLabs/bold/chain-abstraction/sol-implementation"
	"github.com/OffchainLabs/bold/solgen/go/mocksgen"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

var (
	valPrivKey        = flag.String("validator-priv-key", "", "validator private key")
	l1ChainIdStr      = flag.String("l1-chain-id", "", "l1 chain id")
	l1EndpointUrl     = flag.String("l1-endpoint", "", "l1 endpoint")
	rollupAddrStr     = flag.String("rollup-address", "", "rollup address")
	stakeTokenAddrStr = flag.String("stake-token-address", "", "rollup address")
	tokensToDeposit   = flag.String("tokens-to-deposit", "5", "tokens to deposit")
)

func main() {
	ctx := context.Background()
	// validatorPrivateKey, err := crypto.HexToECDSA("182fecf15bdf909556a0f617a63e05ab22f1493d25a9f1e27c228266c772a890")
	// if err != nil {
	// 	panic(err)
	// }
	endpoint, err := rpc.Dial(*l1EndpointUrl)
	if err != nil {
		panic(err)
	}
	client := ethclient.NewClient(endpoint)
	l1ChainId, ok := new(big.Int).SetString(*l1ChainIdStr, 10)
	if !ok {
		panic("not big int")
	}
	validatorPrivateKey, err := crypto.HexToECDSA(*valPrivKey)
	if err != nil {
		panic(err)
	}
	txOpts, err := bind.NewKeyedTransactorWithChainID(validatorPrivateKey, l1ChainId)
	if err != nil {
		panic(err)
	}
	stakeTokenAddr := common.HexToAddress(*stakeTokenAddrStr)
	tokenBindings, err := mocksgen.NewTestWETH9(stakeTokenAddr, client)
	if err != nil {
		panic(err)
	}
	depositAmount, ok := new(big.Int).SetString(*tokensToDeposit, 10)
	if !ok {
		panic("could not set stake token value")
	}
	txOpts.Value = depositAmount
	tx, err := tokenBindings.Deposit(txOpts)
	if err != nil {
		panic(err)
	}
	_ = tx

	rollupAddr := common.HexToAddress(*rollupAddrStr)
	// We then have the validator itself authorize the rollup and challenge manager
	// contracts to spend its stake tokens.
	chain, err := solimpl.NewAssertionChain(
		ctx,
		rollupAddr,
		txOpts,
		client,
	)
	if err != nil {
		panic(err)
	}
	chalManager, err := chain.SpecChallengeManager(ctx)
	if err != nil {
		panic(err)
	}
	amountToApproveSpend := depositAmount
	tx, err = tokenBindings.TestWETH9Transactor.Approve(txOpts, rollupAddr, amountToApproveSpend)
	if err != nil {
		panic(err)
	}
	tx, err = tokenBindings.TestWETH9Transactor.Approve(txOpts, chalManager.Address(), amountToApproveSpend)
	if err != nil {
		panic(err)
	}
	_ = tx
}
