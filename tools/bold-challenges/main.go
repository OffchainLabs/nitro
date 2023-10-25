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
	valPrivKey        = flag.String("validator-priv-key", "93690ac9d039285ed00f874a2694d951c1777ac3a165732f36ea773f16179a89", "validator private key")
	l1ChainIdStr      = flag.String("l1-chain-id", "11155111", "l1 chain id")
	l1EndpointUrl     = flag.String("l1-endpoint", "ws://localhost:8546", "l1 endpoint")
	rollupAddrStr     = flag.String("rollup-address", "0x3f511ad19ad3cd977052af5af35e764ce45bc992", "rollup address")
	stakeTokenAddrStr = flag.String("stake-token-address", "0x931afe52da2da212de18ff7f24deeba3c3869310", "rollup address")
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
	validatorPrivateKey, err := crypto.HexToECDSA(*valPrivKey)
	if err != nil {
		panic(err)
	}
	txOpts, err := bind.NewKeyedTransactorWithChainID(validatorPrivateKey, l1ChainId)
	if err != nil {
		panic(err)
	}

	// if *bridgeFunds {
	// 	inboxAddr := common.HexToAddress(*inboxAddrStr)
	// 	fmt.Println(inboxAddr)
	// 	//"0x9af37196d657d562feb5d332152c6d40cf3ae31a"
	// 	data := hexutil.MustDecode("0x0f4d14e9000000000000000000000000000000000000000000000000000082f79cd90000")
	// 	nonce, err := client.PendingNonceAt(ctx, txOpts.From)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	txOpts.Value = big.NewInt(params.GWei * 100)
	// 	txData := types.DynamicFeeTx{
	// 		To:        &inboxAddr,
	// 		Data:      data,
	// 		Nonce:     nonce,
	// 		Gas:       23000,
	// 		GasFeeCap: big.NewInt(params.GWei * 100),
	// 		GasTipCap: big.NewInt(params.GWei * 3),
	// 		Value:     big.NewInt(params.GWei * 100),
	// 	}
	// 	tx := types.NewTx(&txData)
	// 	signedTx, err := txOpts.Signer(txOpts.From, tx)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	encoded, err := signedTx.MarshalJSON()
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	fmt.Printf("%s\n", encoded)
	// 	if err = client.SendTransaction(ctx, signedTx); err != nil {
	// 		panic(err)
	// 	}
	// 	err = challenge_testing.WaitForTx(ctx, client, signedTx)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	return
	// }

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

	// maxUint256 := new(big.Int)
	// // Set it to 2^256 - 1
	// maxUint256.Exp(big.NewInt(2), big.NewInt(256), nil).Sub(maxUint256, big.NewInt(1))
	// // We then have the validator itself authorize the rollup and challenge manager
	// // contracts to spend its stake tokens.
	// chain, err := solimpl.NewAssertionChain(
	// 	ctx,
	// 	rollupAddr,
	// 	txOpts,
	// 	client,
	// )
	// if err != nil {
	// 	panic(err)
	// }
	// chalManager, err := chain.SpecChallengeManager(ctx)
	// if err != nil {
	// 	panic(err)
	// }
	// tx, err := tokenBindings.TestWETH9Transactor.Approve(txOpts, rollupAddr, maxUint256)
	// if err != nil {
	// 	panic(err)
	// }
	// _ = tx
	// tx, err = tokenBindings.TestWETH9Transactor.Approve(txOpts, chalManager.Address(), maxUint256)
	// if err != nil {
	// 	panic(err)
	// }
	// _ = tx

	allow, err := tokenBindings.Allowance(&bind.CallOpts{}, txOpts.From, rollupAddr)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%#x\n", allow.Bytes())

}
