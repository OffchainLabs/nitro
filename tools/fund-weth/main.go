package main

import (
	"context"
	"flag"
	"fmt"
	"math/big"
	"strings"

	retry "github.com/OffchainLabs/bold/runtime"
	"github.com/OffchainLabs/bold/solgen/go/mocksgen"
	"github.com/OffchainLabs/bold/solgen/go/rollupgen"
	challenge_testing "github.com/OffchainLabs/bold/testing"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

var (
	valPrivKeys       = flag.String("validator-priv-keys", "", "comma-separated, validator private keys to fund and approve mock ERC20 stake token")
	l1ChainIdStr      = flag.String("l1-chain-id", "11155111", "l1 chain id (sepolia default)")
	l1EndpointUrl     = flag.String("l1-endpoint", "", "l1 endpoint")
	rollupAddrStr     = flag.String("rollup-address", "", "rollup address")
	stakeTokenAddrStr = flag.String("stake-token-address", "", "rollup address")
	gweiToDeposit     = flag.Uint64("gwei-to-deposit", 10_000, "tokens to deposit")
)

func main() {
	flag.Parse()
	ctx := context.Background()
	endpoint, err := rpc.DialContext(ctx, *l1EndpointUrl)
	if err != nil {
		panic(err)
	}
	client := ethclient.NewClient(endpoint)
	l1ChainId, ok := new(big.Int).SetString(*l1ChainIdStr, 10)
	if !ok {
		panic("not big int")
	}
	if *valPrivKeys == "" {
		panic("no validator private keys set")
	}
	privKeyStrings := strings.Split(*valPrivKeys, ",")
	for _, privKeyStr := range privKeyStrings {
		validatorPrivateKey, err := crypto.HexToECDSA(privKeyStr)
		if err != nil {
			panic(err)
		}
		txOpts, err := bind.NewKeyedTransactorWithChainID(validatorPrivateKey, l1ChainId)
		if err != nil {
			panic(err)
		}

		rollupAddr := common.HexToAddress(*rollupAddrStr)
		rollupBindings, err := rollupgen.NewRollupUserLogicCaller(rollupAddr, client)
		if err != nil {
			panic(err)
		}
		chalManagerAddr, err := rollupBindings.ChallengeManager(&bind.CallOpts{})
		if err != nil {
			panic(err)
		}
		stakeTokenAddr := common.HexToAddress(*stakeTokenAddrStr)

		tokenBindings, err := mocksgen.NewTestWETH9(stakeTokenAddr, client)
		if err != nil {
			panic(err)
		}
		allow, err := tokenBindings.Allowance(&bind.CallOpts{}, txOpts.From, rollupAddr)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Addr %#x gave rollup %#x allowance of %#x\n", txOpts.From, rollupAddr, allow.Bytes())

		allow, err = tokenBindings.Allowance(&bind.CallOpts{}, txOpts.From, chalManagerAddr)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Addr %#x gave chal manager addr %#x allowance of %#x\n", txOpts.From, chalManagerAddr, allow.Bytes())

		depositAmount := new(big.Int).SetUint64(*gweiToDeposit * params.GWei)
		txOpts.Value = depositAmount
		if _, err = retry.UntilSucceeds[bool](ctx, func() (bool, error) {
			tx, err := tokenBindings.Deposit(txOpts)
			if err != nil {
				return false, err
			}
			if err = challenge_testing.WaitForTx(ctx, client, tx); err != nil {
				return false, err
			}
			return true, nil
		}); err != nil {
			panic(err)
		}
		txOpts.Value = big.NewInt(0)
		maxUint256 := new(big.Int)
		maxUint256.Exp(big.NewInt(2), big.NewInt(256), nil).Sub(maxUint256, big.NewInt(1))
		if _, err = retry.UntilSucceeds[bool](ctx, func() (bool, error) {
			tx, err := tokenBindings.Approve(txOpts, rollupAddr, maxUint256)
			if err != nil {
				return false, err
			}
			if err = challenge_testing.WaitForTx(ctx, client, tx); err != nil {
				return false, err
			}
			return true, nil
		}); err != nil {
			panic(err)
		}
		if _, err = retry.UntilSucceeds[bool](ctx, func() (bool, error) {
			tx, err := tokenBindings.Approve(txOpts, chalManagerAddr, maxUint256)
			if err != nil {
				return false, nil
			}
			if err = challenge_testing.WaitForTx(ctx, client, tx); err != nil {
				return false, err
			}
			return true, nil
		}); err != nil {
			panic(err)
		}

	}
}
