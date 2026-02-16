// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	ethparams "github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/bold/testing/setup"
	"github.com/offchainlabs/nitro/deploy"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

// boldCreatorCache and legacyCreatorCache hold pre-deployed L1 genesis
// state from deployRollupCreator. This is the expensive part of
// deployment; CreateRollup (which produces events) still runs at test time.
type boldCreatorCacheValue struct {
	alloc   types.GenesisAlloc
	creator *setup.CreatorAddresses
}

type legacyCreatorCacheValue struct {
	alloc   types.GenesisAlloc
	creator *deploy.LegacyCreatorAddresses
}

var (
	boldCreatorCache   *boldCreatorCacheValue
	legacyCreatorCache *legacyCreatorCacheValue
)

const creatorCacheTimeout = 5 * time.Minute

// initCreatorCaches builds both BOLD and legacy creator caches.
// Called from TestMain before any tests run.
func initCreatorCaches() error {
	start := time.Now()
	log.Info("building BOLD creator cache")
	var err error
	boldCreatorCache, err = buildBoldCreatorCache()
	if err != nil {
		return fmt.Errorf("building BOLD creator cache: %w", err)
	}
	log.Info("built BOLD creator cache", "elapsed", time.Since(start))
	legacyStart := time.Now()
	log.Info("building legacy creator cache")
	legacyCreatorCache, err = buildLegacyCreatorCache()
	if err != nil {
		return fmt.Errorf("building legacy creator cache: %w", err)
	}
	log.Info("built legacy creator cache", "elapsed", time.Since(legacyStart))
	log.Info("creator caches ready", "totalElapsed", time.Since(start))
	return nil
}

func dumpL1StateToGenesisAlloc(
	l1backend *eth.Ethereum,
) (types.GenesisAlloc, error) {
	statedb, err := l1backend.BlockChain().State()
	if err != nil {
		return nil, fmt.Errorf("getting L1 state: %w", err)
	}
	dump := statedb.RawDump(&state.DumpConfig{})
	alloc := make(types.GenesisAlloc)
	for addrStr, acct := range dump.Accounts {
		addr := common.HexToAddress(addrStr)
		storage := make(map[common.Hash]common.Hash)
		for k, v := range acct.Storage {
			storage[k] = common.HexToHash(v)
		}
		bal, ok := new(big.Int).SetString(acct.Balance, 10)
		if !ok {
			return nil, fmt.Errorf(
				"parsing balance for %s: %q", addrStr, acct.Balance,
			)
		}
		alloc[addr] = types.Account{
			Nonce:   acct.Nonce,
			Balance: bal,
			Code:    acct.Code,
			Storage: storage,
		}
	}
	return alloc, nil
}

// deployOnTempL1 spins up a temporary L1 chain, runs deployFn to deploy
// contracts, dumps the resulting state, and shuts down.
func deployOnTempL1[T any](deployFn func(ctx context.Context, client *ethclient.Client, auth *bind.TransactOpts) (T, error)) (types.GenesisAlloc, T, error) {
	ctx, cancel := context.WithTimeout(context.Background(), creatorCacheTimeout)
	defer cancel()

	var zero T
	deployerKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, zero, fmt.Errorf("generating deployer key: %w", err)
	}
	deployerAddr := crypto.PubkeyToAddress(deployerKey.PublicKey)

	l1Genesis := core.DeveloperGenesisBlock(15_000_000, &deployerAddr)
	l1Genesis.Coinbase = deployerAddr
	l1Genesis.BaseFee = big.NewInt(50 * ethparams.GWei)

	stackConfig := testhelpers.CreateStackConfigForTest("")
	stack, l1backend, _, err := startL1Backend(stackConfig, l1Genesis)
	if err != nil {
		return nil, zero, fmt.Errorf("creating L1 backend: %w", err)
	}
	if err := stack.Start(); err != nil {
		stack.Close()
		return nil, zero, fmt.Errorf("starting L1 stack: %w", err)
	}

	client := ethclient.NewClient(stack.Attach())
	deployerAuth, err := bind.NewKeyedTransactorWithChainID(
		deployerKey, l1Genesis.Config.ChainID,
	)
	if err != nil {
		stack.Close()
		return nil, zero, fmt.Errorf("creating deployer auth: %w", err)
	}

	result, err := deployFn(ctx, client, deployerAuth)
	if err != nil {
		stack.Close()
		return nil, zero, err
	}

	alloc, err := dumpL1StateToGenesisAlloc(l1backend)
	if err != nil {
		stack.Close()
		return nil, zero, fmt.Errorf("dumping L1 state: %w", err)
	}

	stack.Close()
	return alloc, result, nil
}

func buildBoldCreatorCache() (*boldCreatorCacheValue, error) {
	alloc, creator, err := deployOnTempL1(func(ctx context.Context, client *ethclient.Client, auth *bind.TransactOpts) (*setup.CreatorAddresses, error) {
		return setup.DeployCreator(ctx, client, auth, true)
	})
	if err != nil {
		return nil, err
	}
	return &boldCreatorCacheValue{alloc: alloc, creator: creator}, nil
}

func buildLegacyCreatorCache() (*legacyCreatorCacheValue, error) {
	alloc, creator, err := deployOnTempL1(func(ctx context.Context, client *ethclient.Client, auth *bind.TransactOpts) (*deploy.LegacyCreatorAddresses, error) {
		return deployLegacyCreator(ctx, client, auth)
	})
	if err != nil {
		return nil, err
	}
	return &legacyCreatorCacheValue{alloc: alloc, creator: creator}, nil
}

func deployLegacyCreator(
	ctx context.Context,
	client *ethclient.Client,
	auth *bind.TransactOpts,
) (*deploy.LegacyCreatorAddresses, error) {
	arbSys, _ := precompilesgen.NewArbSys(types.ArbSysAddress, client)
	parentChainReader, err := headerreader.New(
		ctx, client,
		func() *headerreader.Config { return &headerreader.TestConfig },
		arbSys,
	)
	if err != nil {
		return nil, err
	}
	parentChainReader.Start(ctx)
	defer parentChainReader.StopAndWait()

	return deploy.DeployCreator(
		ctx, parentChainReader, auth,
		big.NewInt(117964), true,
	)
}
