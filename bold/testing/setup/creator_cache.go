// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package setup

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/offchainlabs/nitro/util/testhelpers/deploycache"
)

var mockCreatorCache struct {
	alloc   types.GenesisAlloc
	creator *CreatorAddresses
}

// InitMockCreatorCache eagerly deploys the mock-OSP rollup creator on a
// temporary L1 and caches the resulting genesis alloc + addresses. Call
// from TestMain before any tests run.
func InitMockCreatorCache() error {
	alloc, creator, err := deploycache.DeployOnTempL1(
		func(ctx context.Context, client *ethclient.Client, auth *bind.TransactOpts) (*CreatorAddresses, error) {
			return DeployCreatorWithMockOSP(ctx, client, auth, true)
		},
	)
	if err != nil {
		return err
	}
	mockCreatorCache.alloc = alloc
	mockCreatorCache.creator = creator
	return nil
}

// CachedMockCreator returns the pre-deployed genesis alloc and creator
// addresses. InitMockCreatorCache must be called first (typically from
// TestMain).
func CachedMockCreator() (types.GenesisAlloc, *CreatorAddresses, error) {
	if mockCreatorCache.creator == nil {
		return nil, nil, fmt.Errorf("mock creator cache not initialized; call setup.InitMockCreatorCache() from TestMain")
	}
	return mockCreatorCache.alloc, mockCreatorCache.creator, nil
}
