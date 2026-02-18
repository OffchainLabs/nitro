// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/bold/testing/setup"
	"github.com/offchainlabs/nitro/deploy"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/testhelpers/deploycache"
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

func buildBoldCreatorCache() (*boldCreatorCacheValue, error) {
	alloc, creator, err := deploycache.DeployOnTempL1(func(ctx context.Context, client *ethclient.Client, auth *bind.TransactOpts) (*setup.CreatorAddresses, error) {
		return setup.DeployCreator(ctx, client, auth, true)
	})
	if err != nil {
		return nil, err
	}
	return &boldCreatorCacheValue{alloc: alloc, creator: creator}, nil
}

func buildLegacyCreatorCache() (*legacyCreatorCacheValue, error) {
	alloc, creator, err := deploycache.DeployOnTempL1(func(ctx context.Context, client *ethclient.Client, auth *bind.TransactOpts) (*deploy.LegacyCreatorAddresses, error) {
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
