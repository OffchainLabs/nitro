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
	valPrivKey        = flag.String("validator-priv-key", "ee3c0bf39d962a78dba87aee083cae443cabc814f93677f302cbabde844237db", "validator private key")
	l1ChainIdStr      = flag.String("l1-chain-id", "11155111", "l1 chain id")
	l1EndpointUrl     = flag.String("l1-endpoint", "ws://localhost:8546", "l1 endpoint")
	rollupAddrStr     = flag.String("rollup-address", "0x24fb41ee084350a521d878be2edee270b30e7421", "rollup address")
	stakeTokenAddrStr = flag.String("stake-token-address", "0x0b713b4de531438b2917e5e73dd43b6360239253", "rollup address")
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
	// 	//"0x03936b232ed50d59d4bdd2dc32feb2fbe91a1b77"
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
	depositAmount, ok := new(big.Int).SetString(*tokensToDeposit, 10)
	if !ok {
		panic("could not set stake token value")
	}
	txOpts.Value = depositAmount
	tx, err := tokenBindings.Deposit(txOpts)
	if err != nil {
		panic(err)
	}
	txOpts.Value = big.NewInt(0)
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
	amountToApproveSpend := big.NewInt(100000)
	tx, err = tokenBindings.TestWETH9Transactor.Approve(txOpts, rollupAddr, amountToApproveSpend)
	if err != nil {
		panic(err)
	}
	_ = tx
	tx, err = tokenBindings.TestWETH9Transactor.Approve(txOpts, chalManager.Address(), amountToApproveSpend)
	if err != nil {
		panic(err)
	}
	_ = tx
}
