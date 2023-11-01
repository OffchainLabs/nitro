package main

import (
	"context"
	"flag"
	"fmt"
	"math/big"

	"github.com/OffchainLabs/bold/solgen/go/mocksgen"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

var (
	valPrivKey        = flag.String("validator-priv-key", "", "validator private key")
	l1ChainIdStr      = flag.String("l1-chain-id", "11155111", "l1 chain id")
	l1EndpointUrl     = flag.String("l1-endpoint", "ws://localhost:8546", "l1 endpoint")
	rollupAddrStr     = flag.String("rollup-address", "", "rollup address")
	stakeTokenAddrStr = flag.String("stake-token-address", "", "rollup address")
	tokensToDeposit   = flag.String("tokens-to-deposit", "100", "tokens to deposit")
)

func main() {
	flag.Parse()
	ctx := context.Background()
	endpoint, err := rpc.DialWebsocket(ctx, *l1EndpointUrl, "*")
	if err != nil {
		panic(err)
	}
	client := ethclient.NewClient(endpoint)
	l1ChainId, ok := new(big.Int).SetString(*l1ChainIdStr, 10)
	if !ok {
		panic("not big int")
	}
	if *valPrivKey == "" {
		panic("no validator private key set")
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
	// depositAmount, ok := new(big.Int).SetString(*tokensToDeposit, 10)
	// if !ok {
	// 	panic("could not set stake token value")
	// }
	txOpts.Value = big.NewInt(params.GWei * 10_000)
	tx, err := tokenBindings.Deposit(txOpts)
	if err != nil {
		panic(err)
	}
	txOpts.Value = big.NewInt(0)
	_ = tx
	rollupAddr := common.HexToAddress(*rollupAddrStr)

	allow, err := tokenBindings.Allowance(&bind.CallOpts{}, txOpts.From, rollupAddr)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%#x\n", allow.Bytes())

}
