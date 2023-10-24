package main

import (
	"math/big"

	solimpl "github.com/OffchainLabs/bold/chain-abstraction/sol-implementation"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/crypto"
)

func main() {
	validatorPrivateKey, err := crypto.HexToECDSA("182fecf15bdf909556a0f617a63e05ab22f1493d25a9f1e27c228266c772a890")
	if err != nil {
		panic(err)
	}
	validatorTxOpts, err := bind.NewKeyedTransactorWithChainID(validatorPrivateKey, l1ChainId)
	if err != nil {
		panic(err)
	}
	mintTokens, ok := new(big.Int).SetString("10000", 10)
	if !ok {
		panic("could not set stake token value")
	}
	l1TransactionOpts.Value = mintTokens
	tx, err = tokenBindings.Deposit(l1TransactionOpts)
	if err != nil {
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

}
